package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"
)

type bodyReader struct {
	reader io.Reader
}

func (r *bodyReader) Read(p []byte) (n int, err error) {
	return r.reader.Read(p)
}

func (r *bodyReader) Close() error {
	_, err := io.Copy(io.Discard, r.reader)
	return err
}

type chunkedBodyReader struct {
	reader     *bufio.Reader
	chunkSize  int
	chunkBytes []byte
}

func (r *chunkedBodyReader) Read(p []byte) (n int, err error) {
	for n < len(p) && err == nil {
		// If we've exhausted the current chunk, read the next one
		if r.chunkSize == 0 && len(r.chunkBytes) == 0 {
			if err = r.readNextChunk(); err != nil {
				return
			}
			if r.chunkSize == 0 { // End of chunked encoding
				return
			}
		}

		// Copy from the current chunk into p
		copied := copy(p[n:], r.chunkBytes)
		n += copied
		r.chunkBytes = r.chunkBytes[copied:]
		r.chunkSize -= copied
	}
	return
}

func (r *chunkedBodyReader) readNextChunk() error {
	// Read the chunk size line
	var sizeLine []byte
	sizeLine, _, err := r.reader.ReadLine()
	if err != nil {
		return err
	}

	fmt.Println(string(sizeLine), sizeLine)

	if bytes.Equal(sizeLine, nlcf) {
		return io.EOF
	}

	// Parse the chunk size
	chunkSizeStr := string(sizeLine[:len(sizeLine)-2]) // Remove \r\n
	fmt.Println(chunkSizeStr)
	chunkSize, err := strconv.ParseInt(chunkSizeStr, 16, 64)
	if err != nil {
		return err
	}
	r.chunkSize = int(chunkSize)

	// If chunk size is 0, it's the end of chunked encoding
	if r.chunkSize == 0 {
		// Read the trailing \r\n
		_, err := io.ReadFull(r.reader, make([]byte, 2))
		return err
	}

	// Read the chunk data
	r.chunkBytes = make([]byte, r.chunkSize)
	_, err = io.ReadFull(r.reader, r.chunkBytes)
	if err != nil {
		return err
	}

	// Read the trailing \r\n
	_, err = io.ReadFull(r.reader, make([]byte, 2))
	return err
}

func (r *chunkedBodyReader) Close() error {
	// make sure we read until the end
	for {
		if _, err := r.Read(make([]byte, 1024)); err != nil {
			if err == io.EOF {
				break // Reached the end
			}
			return err
		}
	}
	return nil
}
