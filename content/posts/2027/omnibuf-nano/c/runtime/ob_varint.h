#ifndef OMNIBUF_NANO_VARINT_H
#define OMNIBUF_NANO_VARINT_H

#include "ob_common.h"

#ifdef __cplusplus
extern "C" {
#endif

/* ZigZag encode/decode for signed integers (WireSint). */
uint64_t ob_zigzag_encode(int64_t n);
int64_t  ob_zigzag_decode(uint64_t u);

/* Canonical encoded byte length of an OmniBuf Prefix-Varint (1, 2, 4, or 9). */
size_t ob_varint_size(uint64_t v);

/* Encode a Prefix-Varint to stream. */
bool ob_varint_encode(ob_ostream_t *stream, uint64_t v);

/* Decode a Prefix-Varint from stream (accepts overlong encodings, rejects reserved prefixes). */
bool ob_varint_decode(ob_istream_t *stream, uint64_t *out);

/* Strict UTF-8 validation (required for WireString). */
bool ob_valid_utf8(const uint8_t *buf, size_t len);

#ifdef __cplusplus
}
#endif

#endif /* OMNIBUF_NANO_VARINT_H */
