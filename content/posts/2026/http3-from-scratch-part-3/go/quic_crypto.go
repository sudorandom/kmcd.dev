package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"hash"
)

// RFC 9001 Section 5.2 Initial Salt for QUIC v1 (0x00000001)
var quicV1InitialSalt = []byte{
	0x38, 0x76, 0x2c, 0xf7, 0xf5, 0x59, 0x34, 0xb3,
	0x4d, 0x17, 0x9a, 0xe6, 0xa4, 0xc8, 0x0c, 0xad,
	0xcc, 0xbb, 0x7f, 0x0a,
}

type packetKeys struct {
	aead  cipher.AEAD
	iv    []byte
	hpKey []byte
}

func deriveInitialKeys(destConnID []byte, isClient bool) (*packetKeys, *packetKeys, error) {
	initialSecret, err := hkdf.Extract(sha256.New, destConnID, quicV1InitialSalt)
	if err != nil {
		return nil, nil, err
	}
	clientSecret := hkdfExpandLabel(sha256.New, initialSecret, "client in", nil, 32)
	serverSecret := hkdfExpandLabel(sha256.New, initialSecret, "server in", nil, 32)

	clientKeys, err := derivePacketKeys(tls.TLS_AES_128_GCM_SHA256, clientSecret)
	if err != nil {
		return nil, nil, err
	}
	serverKeys, err := derivePacketKeys(tls.TLS_AES_128_GCM_SHA256, serverSecret)
	if err != nil {
		return nil, nil, err
	}
	if isClient {
		return clientKeys, serverKeys, nil
	}
	return serverKeys, clientKeys, nil
}

func derivePacketKeys(suite uint16, secret []byte) (*packetKeys, error) {
	var hashFunc func() hash.Hash = sha256.New
	keyLen := 16
	if suite == tls.TLS_AES_256_GCM_SHA384 {
		hashFunc = sha512.New384
		keyLen = 32
	}

	key := hkdfExpandLabel(hashFunc, secret, "quic key", nil, keyLen)
	iv := hkdfExpandLabel(hashFunc, secret, "quic iv", nil, 12)
	hpKey := hkdfExpandLabel(hashFunc, secret, "quic hp", nil, keyLen)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &packetKeys{aead: aead, iv: iv, hpKey: hpKey}, nil
}

func hkdfExpandLabel(hashFunc func() hash.Hash, secret []byte, label string, context []byte, length int) []byte {
	fullLabel := "tls13 " + label
	hkdfLabel := make([]byte, 2+1+len(fullLabel)+1+len(context))
	binary.BigEndian.PutUint16(hkdfLabel[0:2], uint16(length))
	hkdfLabel[2] = uint8(len(fullLabel))
	copy(hkdfLabel[3:], fullLabel)
	hkdfLabel[3+len(fullLabel)] = uint8(len(context))
	copy(hkdfLabel[3+len(fullLabel)+1:], context)

	out, err := hkdf.Expand(hashFunc, secret, string(hkdfLabel), length)
	if err != nil {
		panic(err)
	}
	return out
}

func (k *packetKeys) makeNonce(pktNum uint64) []byte {
	nonce := make([]byte, len(k.iv))
	copy(nonce, k.iv)
	var pnBytes [8]byte
	binary.BigEndian.PutUint64(pnBytes[:], pktNum)
	for i := 0; i < 8; i++ {
		nonce[len(nonce)-8+i] ^= pnBytes[i]
	}
	return nonce
}

func (k *packetKeys) headerProtectionMask(sample []byte) ([]byte, error) {
	if len(sample) < 16 {
		return nil, errors.New("sample too short")
	}
	block, err := aes.NewCipher(k.hpKey)
	if err != nil {
		return nil, err
	}
	mask := make([]byte, 16)
	block.Encrypt(mask, sample[:16])
	return mask[:5], nil
}

func (k *packetKeys) protectPacket(headerWithoutPN []byte, pktNum uint32, payload []byte) ([]byte, error) {
	var pnBytes [4]byte
	binary.BigEndian.PutUint32(pnBytes[:], pktNum)

	pnOffset := len(headerWithoutPN)
	header := append(append([]byte(nil), headerWithoutPN...), pnBytes[:]...)

	ciphertext := k.aead.Seal(nil, k.makeNonce(uint64(pktNum)), payload, header)
	pkt := append(header, ciphertext...)

	sample := pkt[pnOffset+4 : pnOffset+20]
	mask, err := k.headerProtectionMask(sample)
	if err != nil {
		return nil, err
	}
	if pkt[0]&0x80 != 0 {
		pkt[0] ^= mask[0] & 0x0f
	} else {
		pkt[0] ^= mask[0] & 0x1f
	}
	for i := 0; i < 4; i++ {
		pkt[pnOffset+i] ^= mask[1+i]
	}
	return pkt, nil
}

func (k *packetKeys) unprotectPacket(raw []byte, pnOffset int, expectedPN uint64) (pktNum uint64, plaintext []byte, err error) {
	if len(raw) < pnOffset+20 {
		return 0, nil, errors.New("packet too short")
	}
	pkt := append([]byte(nil), raw...)
	sample := pkt[pnOffset+4 : pnOffset+20]
	mask, err := k.headerProtectionMask(sample)
	if err != nil {
		return 0, nil, err
	}
	if pkt[0]&0x80 != 0 {
		pkt[0] ^= mask[0] & 0x0f
	} else {
		pkt[0] ^= mask[0] & 0x1f
	}
	pnLen := int(pkt[0]&0x03) + 1
	var truncatedPN uint64
	for i := 0; i < pnLen; i++ {
		pkt[pnOffset+i] ^= mask[1+i]
		truncatedPN = (truncatedPN << 8) | uint64(pkt[pnOffset+i])
	}
	pn := decodePacketNumber(expectedPN, truncatedPN, uint(pnLen*8))

	header := pkt[:pnOffset+pnLen]
	ciphertext := pkt[pnOffset+pnLen:]
	plaintext, err = k.aead.Open(nil, k.makeNonce(pn), ciphertext, header)
	if err != nil {
		return 0, nil, err
	}
	return pn, plaintext, nil
}

// decodePacketNumber reconstructs the full 62-bit packet number from a truncated
// wire packet number per RFC 9000 Appendix A.3.
func decodePacketNumber(expectedPN, truncatedPN uint64, pnNBits uint) uint64 {
	pnWin := uint64(1) << pnNBits
	pnHwin := pnWin / 2
	pnMask := pnWin - 1
	candidatePN := (expectedPN & ^pnMask) | truncatedPN
	if candidatePN+pnHwin <= expectedPN && candidatePN < (1<<62)-pnWin {
		return candidatePN + pnWin
	}
	if candidatePN > expectedPN+pnHwin && candidatePN >= pnWin {
		return candidatePN - pnWin
	}
	return candidatePN
}

func marshalTransportParams(initialSrcCID []byte) []byte {
	var b []byte
	appendParamVarint := func(id uint64, val uint64) {
		var v []byte
		v = AppendVarint(v, val)
		b = AppendVarint(b, id)
		b = AppendVarint(b, uint64(len(v)))
		b = append(b, v...)
	}
	appendParamVarint(0x04, 1048576) // initial_max_data (1 MB)
	appendParamVarint(0x05, 1048576) // initial_max_stream_data_bidi_local
	appendParamVarint(0x06, 1048576) // initial_max_stream_data_bidi_remote
	appendParamVarint(0x07, 1048576) // initial_max_stream_data_uni
	appendParamVarint(0x08, 100)     // initial_max_streams_bidi
	appendParamVarint(0x09, 100)     // initial_max_streams_uni

	// initial_source_connection_id (0x0f)
	b = AppendVarint(b, 0x0f)
	b = AppendVarint(b, uint64(len(initialSrcCID)))
	b = append(b, initialSrcCID...)
	return b
}
