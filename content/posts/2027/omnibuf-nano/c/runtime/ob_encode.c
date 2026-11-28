#include "ob_encode.h"
#include "ob_varint.h"

static bool buf_write_callback(ob_ostream_t *stream, const uint8_t *buf, size_t count) {
    uint8_t *dest = (uint8_t *)stream->state;
    memcpy(dest, buf, count);
    stream->state = dest + count;
    return true;
}

ob_ostream_t ob_ostream_from_buffer(uint8_t *buf, size_t bufsize) {
    ob_ostream_t stream;
    stream.callback = buf_write_callback;
    stream.state = buf;
    stream.max_size = bufsize;
    stream.bytes_written = 0u;
    stream.errmsg = NULL;
    return stream;
}

bool ob_write(ob_ostream_t *stream, const uint8_t *buf, size_t count) {
    if (count == 0u) {
        return true;
    }
    if (count > (SIZE_MAX - stream->bytes_written)) {
        OB_RETURN_ERROR(stream, "size overflow");
    }
    if (stream->bytes_written > stream->max_size ||
        (stream->max_size - stream->bytes_written) < count) {
        OB_RETURN_ERROR(stream, "stream full");
    }
    if (stream->callback == buf_write_callback) {
        uint8_t *dest = (uint8_t *)stream->state;
        memcpy(dest, buf, count);
        stream->state = dest + count;
    } else if (stream->callback != NULL) {
        if (!stream->callback(stream, buf, count)) {
            if (stream->errmsg == NULL) {
                stream->errmsg = "write callback failed";
            }
            return false;
        }
    }
    stream->bytes_written += count;
    return true;
}

bool ob_write_tag(ob_ostream_t *stream, uint16_t tag, uint8_t wire_type) {
    uint64_t raw = ((uint64_t)tag << 4) | (uint64_t)(wire_type & 0x0Fu);
    return ob_varint_encode(stream, raw);
}

static bool write_le32(ob_ostream_t *stream, uint32_t v) {
    uint8_t b[4];
    b[0] = (uint8_t)(v & 0xFFu);
    b[1] = (uint8_t)((v >> 8) & 0xFFu);
    b[2] = (uint8_t)((v >> 16) & 0xFFu);
    b[3] = (uint8_t)((v >> 24) & 0xFFu);
    return ob_write(stream, b, 4u);
}

static bool write_le64(ob_ostream_t *stream, uint64_t v) {
    uint8_t b[8];
    b[0] = (uint8_t)(v & 0xFFu);
    b[1] = (uint8_t)((v >> 8) & 0xFFu);
    b[2] = (uint8_t)((v >> 16) & 0xFFu);
    b[3] = (uint8_t)((v >> 24) & 0xFFu);
    b[4] = (uint8_t)((v >> 32) & 0xFFu);
    b[5] = (uint8_t)((v >> 40) & 0xFFu);
    b[6] = (uint8_t)((v >> 48) & 0xFFu);
    b[7] = (uint8_t)((v >> 56) & 0xFFu);
    return ob_write(stream, b, 8u);
}

static uint64_t read_uint_field(const void *ptr, uint16_t data_size) {
    if (data_size == 1u) {
        uint8_t v = 0;
        memcpy(&v, ptr, sizeof(v));
        return (uint64_t)v;
    }
    if (data_size == 2u) {
        uint16_t v = 0;
        memcpy(&v, ptr, sizeof(v));
        return (uint64_t)v;
    }
    if (data_size == 4u) {
        uint32_t v = 0;
        memcpy(&v, ptr, sizeof(v));
        return (uint64_t)v;
    }
    uint64_t v = 0;
    memcpy(&v, ptr, sizeof(v));
    return v;
}

static int64_t read_sint_field(const void *ptr, uint16_t data_size) {
    if (data_size == 1u) {
        int8_t v = 0;
        memcpy(&v, ptr, sizeof(v));
        return (int64_t)v;
    }
    if (data_size == 2u) {
        int16_t v = 0;
        memcpy(&v, ptr, sizeof(v));
        return (int64_t)v;
    }
    if (data_size == 4u) {
        int32_t v = 0;
        memcpy(&v, ptr, sizeof(v));
        return (int64_t)v;
    }
    int64_t v = 0;
    memcpy(&v, ptr, sizeof(v));
    return v;
}

static size_t bounded_strlen(const char *s, size_t max_size) {
    size_t n = 0;
    while (n < max_size && s[n] != '\0') {
        n++;
    }
    return n;
}

static bool encode_msg_depth(ob_ostream_t *stream, const ob_msgdesc_t *desc,
                             const void *src_struct, unsigned int depth);

static bool buf_begin_container(ob_ostream_t *stream, uint8_t **len_ptr_out, size_t *start_out) {
    *len_ptr_out = (uint8_t *)stream->state;
    *start_out = stream->bytes_written;
    const uint8_t zeros[2] = {0u, 0u};
    return ob_write(stream, zeros, 2u);
}

static bool buf_end_container(ob_ostream_t *stream, uint8_t *len_ptr, size_t start_written) {
    size_t body_len = stream->bytes_written - (start_written + 2u);
    if (body_len <= 16383u) {
        uint16_t u = (uint16_t)(((uint16_t)body_len << 2) | 2u);
        len_ptr[0] = (uint8_t)(u & 0xFFu);
        len_ptr[1] = (uint8_t)((u >> 8) & 0xFFu);
        return true;
    }
#if SIZE_MAX > 536870911UL
    if (body_len <= 536870911UL) {
#endif
        if (stream->bytes_written > stream->max_size ||
            (stream->max_size - stream->bytes_written) < 2u) {
            OB_RETURN_ERROR(stream, "stream full");
        }
        memmove(len_ptr + 4u, len_ptr + 2u, body_len);
        uint32_t u = (uint32_t)(((uint32_t)body_len << 3) | 4u);
        len_ptr[0] = (uint8_t)(u & 0xFFu);
        len_ptr[1] = (uint8_t)((u >> 8) & 0xFFu);
        len_ptr[2] = (uint8_t)((u >> 16) & 0xFFu);
        len_ptr[3] = (uint8_t)((u >> 24) & 0xFFu);
        stream->state = (uint8_t *)stream->state + 2u;
        stream->bytes_written += 2u;
        return true;
#if SIZE_MAX > 536870911UL
    }
    if (stream->bytes_written > stream->max_size ||
        (stream->max_size - stream->bytes_written) < 7u) {
        OB_RETURN_ERROR(stream, "stream full");
    }
    memmove(len_ptr + 9u, len_ptr + 2u, body_len);
    uint64_t v = (uint64_t)body_len;
    len_ptr[0] = 0x00u;
    len_ptr[1] = (uint8_t)(v & 0xFFu);
    len_ptr[2] = (uint8_t)((v >> 8) & 0xFFu);
    len_ptr[3] = (uint8_t)((v >> 16) & 0xFFu);
    len_ptr[4] = (uint8_t)((v >> 24) & 0xFFu);
    len_ptr[5] = (uint8_t)((v >> 32) & 0xFFu);
    len_ptr[6] = (uint8_t)((v >> 40) & 0xFFu);
    len_ptr[7] = (uint8_t)((v >> 48) & 0xFFu);
    len_ptr[8] = (uint8_t)((v >> 56) & 0xFFu);
    stream->state = (uint8_t *)stream->state + 7u;
    stream->bytes_written += 7u;
    return true;
#endif
}

/* Encode a single value payload (without field tag).
 * Used for singular fields (after tag), WireArray elements, and WireMap keys/values.
 * Note: For bool in container context, writes 1 byte (0 or 1). */
static bool encode_payload(ob_ostream_t *stream, const ob_field_t *field,
                           const void *ptr, unsigned int depth) {
    switch (field->wire_type) {
    case OB_WIRE_UINT:
        return ob_varint_encode(stream, read_uint_field(ptr, field->data_size));

    case OB_WIRE_SINT:
        return ob_varint_encode(stream, ob_zigzag_encode(read_sint_field(ptr, field->data_size)));

    case OB_WIRE_BOOL_FALSE: {
        bool v = false;
        memcpy(&v, ptr, sizeof(v));
        uint8_t b = v ? 1u : 0u;
        return ob_write(stream, &b, 1u);
    }

    case OB_WIRE_FIXED32:
    case OB_WIRE_SFIXED32:
    case OB_WIRE_FLOAT32: {
        if (field->data_size != 4u) {
            OB_RETURN_ERROR(stream, "invalid 32-bit field size");
        }
        uint32_t bits = 0;
        memcpy(&bits, ptr, sizeof(bits));
        return write_le32(stream, bits);
    }

    case OB_WIRE_FIXED64:
    case OB_WIRE_SFIXED64:
    case OB_WIRE_FLOAT64: {
        if (field->data_size != 8u) {
            OB_RETURN_ERROR(stream, "invalid 64-bit field size");
        }
        uint64_t bits = 0;
        memcpy(&bits, ptr, sizeof(bits));
        return write_le64(stream, bits);
    }

    case OB_WIRE_STRING: {
        const char *s = (const char *)ptr;
        size_t slen = bounded_strlen(s, field->data_size);
        if (slen == field->data_size) {
            OB_RETURN_ERROR(stream, "unterminated string");
        }
        if (!ob_valid_utf8((const uint8_t *)s, slen)) {
            OB_RETURN_ERROR(stream, "invalid UTF-8 in string");
        }
        if (!ob_varint_encode(stream, (uint64_t)slen)) {
            return false;
        }
        return ob_write(stream, (const uint8_t *)s, slen);
    }

    case OB_WIRE_BYTES: {
        if ((field->flags & OB_FLAG_FIXED_LEN) != 0u) {
            if (!ob_varint_encode(stream, (uint64_t)field->data_size)) {
                return false;
            }
            return ob_write(stream, (const uint8_t *)ptr, field->data_size);
        } else {
            uint16_t blen = 0;
            memcpy(&blen, ptr, sizeof(blen));
            if (blen > field->data_size) {
                OB_RETURN_ERROR(stream, "bytes length exceeds max_size");
            }
            if (!ob_varint_encode(stream, (uint64_t)blen)) {
                return false;
            }
            const uint8_t *data = (const uint8_t *)ptr + sizeof(uint16_t);
            return ob_write(stream, data, blen);
        }
    }

    case OB_WIRE_SUBMSG: {
        if (field->submsg == NULL) {
            OB_RETURN_ERROR(stream, "missing submsg descriptor");
        }
        if (stream->callback == buf_write_callback) {
            uint8_t *len_ptr = NULL;
            size_t start_written = 0u;
            if (!buf_begin_container(stream, &len_ptr, &start_written)) {
                return false;
            }
            if (!encode_msg_depth(stream, field->submsg, ptr, depth + 1u)) {
                return false;
            }
            return buf_end_container(stream, len_ptr, start_written);
        }
        ob_ostream_t sizing = {NULL, NULL, SIZE_MAX, 0u, NULL};
        if (!encode_msg_depth(&sizing, field->submsg, ptr, depth + 1u)) {
            stream->errmsg = sizing.errmsg;
            return false;
        }
        if (!ob_varint_encode(stream, (uint64_t)sizing.bytes_written)) {
            return false;
        }
        if (stream->callback == NULL) {
            /* Sizing pass: skip redundant second traversal of submsg. */
            return ob_write(stream, NULL, sizing.bytes_written);
        }
        return encode_msg_depth(stream, field->submsg, ptr, depth + 1u);
    }

    default:
        OB_RETURN_ERROR(stream, "unsupported wire type");
    }
}

static size_t repeated_elem_stride(const ob_field_t *field) {
    if (field->wire_type == OB_WIRE_BYTES && (field->flags & OB_FLAG_FIXED_LEN) == 0u) {
        return OB_BYTES_STRIDE(field->data_size);
    }
    return (size_t)field->data_size;
}

#define OB_ELEM_SIZE_CACHE_LEN 8u

static bool encode_array_body(ob_ostream_t *stream, const ob_field_t *field,
                              const uint8_t *arr_ptr, uint16_t count,
                              size_t *cached_sizes, size_t cache_cap,
                              unsigned int depth) {
    uint8_t elem_wt = field->wire_type;
    if (!ob_write(stream, &elem_wt, 1u)) {
        return false;
    }
    if (!ob_varint_encode(stream, (uint64_t)count)) {
        return false;
    }
    size_t stride = repeated_elem_stride(field);
    if (elem_wt == OB_WIRE_SUBMSG) {
        if (field->submsg == NULL) {
            OB_RETURN_ERROR(stream, "missing submsg descriptor");
        }
        for (uint16_t i = 0; i < count; i++) {
            const uint8_t *elem_ptr = arr_ptr + ((size_t)i * stride);
            size_t sub_len = 0u;
            if (stream->callback == buf_write_callback) {
                uint8_t *len_ptr = NULL;
                size_t start_written = 0u;
                if (!buf_begin_container(stream, &len_ptr, &start_written)) {
                    return false;
                }
                if (!encode_msg_depth(stream, field->submsg, elem_ptr, depth + 1u)) {
                    return false;
                }
                if (!buf_end_container(stream, len_ptr, start_written)) {
                    return false;
                }
            } else if (stream->callback == NULL) {
                ob_ostream_t sizing = {NULL, NULL, SIZE_MAX, 0u, NULL};
                if (!encode_msg_depth(&sizing, field->submsg, elem_ptr, depth + 1u)) {
                    stream->errmsg = sizing.errmsg;
                    return false;
                }
                sub_len = sizing.bytes_written;
                if (cached_sizes != NULL && i < cache_cap) {
                    cached_sizes[i] = sub_len;
                }
                if (!ob_varint_encode(stream, (uint64_t)sub_len)) {
                    return false;
                }
                if (!ob_write(stream, NULL, sub_len)) {
                    return false;
                }
            } else {
                if (cached_sizes != NULL && i < cache_cap) {
                    sub_len = cached_sizes[i];
                } else {
                    ob_ostream_t sizing = {NULL, NULL, SIZE_MAX, 0u, NULL};
                    if (!encode_msg_depth(&sizing, field->submsg, elem_ptr, depth + 1u)) {
                        stream->errmsg = sizing.errmsg;
                        return false;
                    }
                    sub_len = sizing.bytes_written;
                }
                if (!ob_varint_encode(stream, (uint64_t)sub_len)) {
                    return false;
                }
                if (!encode_msg_depth(stream, field->submsg, elem_ptr, depth + 1u)) {
                    return false;
                }
            }
        }
        return true;
    }
    for (uint16_t i = 0; i < count; i++) {
        if (!encode_payload(stream, field, arr_ptr + ((size_t)i * stride), depth)) {
            return false;
        }
    }
    return true;
}

static bool encode_repeated_field(ob_ostream_t *stream, const ob_field_t *field,
                                  const uint8_t *base, const uint8_t *data_ptr,
                                  unsigned int depth) {
    if (depth + 1u > OB_MAX_DEPTH) {
        OB_RETURN_ERROR(stream, "max nesting depth exceeded");
    }
    uint16_t count = field->array_size;
    if ((field->flags & OB_FLAG_FIXED_LEN) == 0u) {
        memcpy(&count, base + field->size_offset, sizeof(count));
    }
    if (count == 0u) {
        return true;
    }
    if (count > field->array_size) {
        OB_RETURN_ERROR(stream, "array count exceeds max_count");
    }
    if (stream->callback == buf_write_callback) {
        if (!ob_write_tag(stream, field->tag, OB_WIRE_ARRAY)) {
            return false;
        }
        uint8_t *len_ptr = NULL;
        size_t start_written = 0u;
        if (!buf_begin_container(stream, &len_ptr, &start_written)) {
            return false;
        }
        if (!encode_array_body(stream, field, data_ptr, count, NULL, 0u, depth + 1u)) {
            return false;
        }
        return buf_end_container(stream, len_ptr, start_written);
    }
    size_t cached_sizes[OB_ELEM_SIZE_CACHE_LEN];
    size_t *cache_ptr = (field->wire_type == OB_WIRE_SUBMSG) ? cached_sizes : NULL;
    ob_ostream_t sizing = {NULL, NULL, SIZE_MAX, 0u, NULL};
    if (!encode_array_body(&sizing, field, data_ptr, count,
                           cache_ptr, OB_ELEM_SIZE_CACHE_LEN, depth + 1u)) {
        stream->errmsg = sizing.errmsg;
        return false;
    }
    if (!ob_write_tag(stream, field->tag, OB_WIRE_ARRAY)) {
        return false;
    }
    if (!ob_varint_encode(stream, (uint64_t)sizing.bytes_written)) {
        return false;
    }
    if (stream->callback == NULL) {
        return ob_write(stream, NULL, sizing.bytes_written);
    }
    return encode_array_body(stream, field, data_ptr, count,
                             cache_ptr, OB_ELEM_SIZE_CACHE_LEN, depth + 1u);
}

static bool encode_map_body(ob_ostream_t *stream, const ob_field_t *field,
                            const uint8_t *arr_ptr, uint16_t count,
                            unsigned int depth) {
    const ob_msgdesc_t *entry_desc = field->submsg;
    if (entry_desc == NULL || entry_desc->field_count != 2u) {
        OB_RETURN_ERROR(stream, "invalid map entry descriptor");
    }
    const ob_field_t *kfield = &entry_desc->fields[0];
    const ob_field_t *vfield = &entry_desc->fields[1];
    uint8_t header[2];
    header[0] = kfield->wire_type;
    header[1] = vfield->wire_type;
    if (!ob_write(stream, header, 2u)) {
        return false;
    }
    if (!ob_varint_encode(stream, (uint64_t)count)) {
        return false;
    }
    size_t stride = (size_t)field->data_size;
    for (uint16_t i = 0; i < count; i++) {
        const uint8_t *entry = arr_ptr + ((size_t)i * stride);
        if (!encode_payload(stream, kfield, entry + kfield->data_offset, depth)) {
            return false;
        }
        if (!encode_payload(stream, vfield, entry + vfield->data_offset, depth)) {
            return false;
        }
    }
    return true;
}

static bool is_singular_nonzero(const ob_field_t *field, const void *ptr) {
    switch (field->wire_type) {
    case OB_WIRE_UINT:
        return read_uint_field(ptr, field->data_size) != 0u;
    case OB_WIRE_SINT:
        return read_sint_field(ptr, field->data_size) != 0;
    case OB_WIRE_BOOL_FALSE: {
        bool v = false;
        memcpy(&v, ptr, sizeof(v));
        return v;
    }
    case OB_WIRE_FIXED32:
    case OB_WIRE_SFIXED32:
    case OB_WIRE_FLOAT32: {
        uint32_t bits = 0;
        memcpy(&bits, ptr, sizeof(bits));
        return bits != 0u;
    }
    case OB_WIRE_FIXED64:
    case OB_WIRE_SFIXED64:
    case OB_WIRE_FLOAT64: {
        uint64_t bits = 0;
        memcpy(&bits, ptr, sizeof(bits));
        return bits != 0u;
    }
    case OB_WIRE_STRING:
        return ((const char *)ptr)[0] != '\0';
    case OB_WIRE_BYTES: {
        if ((field->flags & OB_FLAG_FIXED_LEN) != 0u) {
            return true;
        }
        uint16_t blen = 0;
        memcpy(&blen, ptr, sizeof(blen));
        return blen != 0u;
    }
    case OB_WIRE_SUBMSG:
        return true;
    default:
        return false;
    }
}

static bool encode_field(ob_ostream_t *stream, const ob_field_t *field,
                         const uint8_t *base, unsigned int depth) {
    const uint8_t *data_ptr = base + field->data_offset;

    if ((field->flags & OB_FLAG_MAP) != 0u) {
        if (depth + 1u > OB_MAX_DEPTH) {
            OB_RETURN_ERROR(stream, "max nesting depth exceeded");
        }
        uint16_t count = 0;
        memcpy(&count, base + field->size_offset, sizeof(count));
        if (count == 0u) {
            return true;
        }
        if (count > field->array_size) {
            OB_RETURN_ERROR(stream, "map count exceeds max_count");
        }
        if (stream->callback == buf_write_callback) {
            if (!ob_write_tag(stream, field->tag, OB_WIRE_MAP)) {
                return false;
            }
            uint8_t *len_ptr = NULL;
            size_t start_written = 0u;
            if (!buf_begin_container(stream, &len_ptr, &start_written)) {
                return false;
            }
            if (!encode_map_body(stream, field, data_ptr, count, depth + 1u)) {
                return false;
            }
            return buf_end_container(stream, len_ptr, start_written);
        }
        ob_ostream_t sizing = {NULL, NULL, SIZE_MAX, 0u, NULL};
        if (!encode_map_body(&sizing, field, data_ptr, count, depth + 1u)) {
            stream->errmsg = sizing.errmsg;
            return false;
        }
        if (!ob_write_tag(stream, field->tag, OB_WIRE_MAP)) {
            return false;
        }
        if (!ob_varint_encode(stream, (uint64_t)sizing.bytes_written)) {
            return false;
        }
        if (stream->callback == NULL) {
            return ob_write(stream, NULL, sizing.bytes_written);
        }
        return encode_map_body(stream, field, data_ptr, count, depth + 1u);
    }

    if ((field->flags & OB_FLAG_REPEATED) != 0u) {
        return encode_repeated_field(stream, field, base, data_ptr, depth);
    }

    /* Singular / Optional / Required / Oneof field */
    if ((field->flags & OB_FLAG_ONEOF) != 0u) {
        uint16_t which = 0;
        memcpy(&which, base + field->size_offset, sizeof(which));
        if (which != field->tag) {
            return true;
        }
    } else if ((field->flags & OB_FLAG_OPTIONAL) != 0u) {
        bool has_field = false;
        memcpy(&has_field, base + field->size_offset, sizeof(has_field));
        if (!has_field) {
            return true;
        }
    } else if ((field->flags & OB_FLAG_REQUIRED) == 0u) {
        /* Proto3 implicit presence: omit default/zero values. */
        if (!is_singular_nonzero(field, data_ptr)) {
            return true;
        }
    }

    /* Bool encodes its value directly in the 4-bit wire type (0 payload bytes). */
    if (field->wire_type == OB_WIRE_BOOL_FALSE) {
        bool v = false;
        memcpy(&v, data_ptr, sizeof(v));
        return ob_write_tag(stream, field->tag, v ? OB_WIRE_BOOL_TRUE : OB_WIRE_BOOL_FALSE);
    }

    if (!ob_write_tag(stream, field->tag, field->wire_type)) {
        return false;
    }
    return encode_payload(stream, field, data_ptr, depth);
}

static bool encode_msg_depth(ob_ostream_t *stream, const ob_msgdesc_t *desc,
                             const void *src_struct, unsigned int depth) {
    if (depth > OB_MAX_DEPTH) {
        OB_RETURN_ERROR(stream, "max nesting depth exceeded");
    }
    const uint8_t *base = (const uint8_t *)src_struct;
    for (uint16_t i = 0; i < desc->field_count; i++) {
        if (!encode_field(stream, &desc->fields[i], base, depth)) {
            return false;
        }
    }
    return true;
}

bool ob_encode(ob_ostream_t *stream, const ob_msgdesc_t *desc, const void *src_struct) {
    if (stream == NULL || desc == NULL || src_struct == NULL) {
        if (stream != NULL) {
            stream->errmsg = "null argument";
        }
        return false;
    }
    return encode_msg_depth(stream, desc, src_struct, 0u);
}

bool ob_get_encoded_size(size_t *size, const ob_msgdesc_t *desc, const void *src_struct) {
    if (size == NULL || desc == NULL || src_struct == NULL) {
        return false;
    }
    ob_ostream_t sizing = {NULL, NULL, SIZE_MAX, 0u, NULL};
    if (!encode_msg_depth(&sizing, desc, src_struct, 0u)) {
        return false;
    }
    *size = sizing.bytes_written;
    return true;
}
