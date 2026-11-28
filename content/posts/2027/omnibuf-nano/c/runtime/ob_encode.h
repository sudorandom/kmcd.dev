#ifndef OMNIBUF_NANO_ENCODE_H
#define OMNIBUF_NANO_ENCODE_H

#include "ob_common.h"

#ifdef __cplusplus
extern "C" {
#endif

/* Construct an output stream backed by a fixed byte buffer. */
ob_ostream_t ob_ostream_from_buffer(uint8_t *buf, size_t bufsize);

/* Write count bytes to stream (if callback == NULL, counts bytes only). */
bool ob_write(ob_ostream_t *stream, const uint8_t *buf, size_t count);

/* Write an OmniBuf field tag: Prefix-Varint((tag << 4) | wire_type). */
bool ob_write_tag(ob_ostream_t *stream, uint16_t tag, uint8_t wire_type);

/* Encode a message struct to stream using 2-pass streaming for containers. */
bool ob_encode(ob_ostream_t *stream, const ob_msgdesc_t *desc, const void *src_struct);

/* Compute the exact encoded byte size of a message without writing output. */
bool ob_get_encoded_size(size_t *size, const ob_msgdesc_t *desc, const void *src_struct);

#ifdef __cplusplus
}
#endif

#endif /* OMNIBUF_NANO_ENCODE_H */
