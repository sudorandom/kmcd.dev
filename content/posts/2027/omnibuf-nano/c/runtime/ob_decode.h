#ifndef OMNIBUF_NANO_DECODE_H
#define OMNIBUF_NANO_DECODE_H

#include "ob_common.h"

#ifdef __cplusplus
extern "C" {
#endif

/* Construct an input stream backed by a read-only byte buffer. */
ob_istream_t ob_istream_from_buffer(const uint8_t *buf, size_t msglen);

/* Read exactly count bytes from stream. */
bool ob_read(ob_istream_t *stream, uint8_t *buf, size_t count);

/* Zero-initialize dst_struct and decode an OmniBuf message from stream. */
bool ob_decode(ob_istream_t *stream, const ob_msgdesc_t *desc, void *dst_struct);

/* Decode an OmniBuf message into dst_struct without zeroing it first (merges fields). */
bool ob_decode_noinit(ob_istream_t *stream, const ob_msgdesc_t *desc, void *dst_struct);

#ifdef __cplusplus
}
#endif

#endif /* OMNIBUF_NANO_DECODE_H */
