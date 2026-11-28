#ifndef OMNIBUF_NANO_COMMON_H
#define OMNIBUF_NANO_COMMON_H

#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>
#include <string.h>

#ifdef __cplusplus
extern "C" {
#endif

/* Maximum container nesting depth (matches Go OmniBuf MaxNestingDepth = 32). */
#ifndef OB_MAX_DEPTH
#define OB_MAX_DEPTH 32u
#endif

/* Maximum valid Protobuf field number ((1 << 29) - 1). */
#define OB_MAX_FIELD_NUM 536870911UL

/* OmniBuf explicit 4-bit wire types (0..15). */
#define OB_WIRE_UINT       0u
#define OB_WIRE_SINT       1u
#define OB_WIRE_BOOL_FALSE 2u
#define OB_WIRE_BOOL_TRUE  3u
#define OB_WIRE_FIXED32    4u
#define OB_WIRE_FIXED64    5u
#define OB_WIRE_SFIXED32   6u
#define OB_WIRE_SFIXED64   7u
#define OB_WIRE_FLOAT32    8u
#define OB_WIRE_FLOAT64    9u
#define OB_WIRE_STRING     10u
#define OB_WIRE_BYTES      11u
#define OB_WIRE_SUBMSG     12u
#define OB_WIRE_ARRAY      13u
#define OB_WIRE_MAP        14u
#define OB_WIRE_EXTENDED   15u

/* Field descriptor flags (bitmask in ob_field_t.flags). */
#define OB_FLAG_SINGULAR   0x00u /* Implicit presence (proto3 scalar/string/bytes) */
#define OB_FLAG_OPTIONAL   0x01u /* Explicit bool has_<name> at size_offset */
#define OB_FLAG_REQUIRED   0x02u /* Always encoded; decode fails if missing */
#define OB_FLAG_REPEATED   0x04u /* WireArray; uint16_t count at size_offset */
#define OB_FLAG_ONEOF      0x08u /* uint16_t which_<oneof> at size_offset */
#define OB_FLAG_FIXED_LEN  0x10u /* Fixed-capacity bytes or repeated (no size/count) */
#define OB_FLAG_MAP        0x20u /* WireMap; submsg holds key (idx 0) and value (idx 1) */

/* Helper macro for variable-length bytes fields in generated structs. */
#define OB_BYTES_ARRAY_T(n) struct { uint16_t size; uint8_t bytes[n]; }

/* Portable alignment and stride calculation for OB_BYTES_ARRAY_T(cap). */
typedef struct {
    char c;
    uint16_t u;
} ob_u16_align_probe_t;

#define OB_U16_ALIGN (sizeof(ob_u16_align_probe_t) - sizeof(uint16_t))
#define OB_BYTES_STRIDE(cap) \
    (((sizeof(uint16_t) + (size_t)(cap)) + (OB_U16_ALIGN - 1u)) & ~(OB_U16_ALIGN - 1u))

/* Convenience error-return macro for stream functions. */
#define OB_RETURN_ERROR(stream, msg) \
    do {                             \
        (stream)->errmsg = (msg);    \
        return false;                \
    } while (0)

typedef struct ob_msgdesc ob_msgdesc_t;

typedef struct {
    uint16_t tag;               /* Protobuf field number */
    uint8_t  wire_type;         /* OmniBuf 4-bit wire type (element type for repeated) */
    uint8_t  flags;             /* OB_FLAG_* bitmask */
    uint16_t data_offset;       /* offsetof(Msg, field) */
    uint16_t size_offset;       /* offsetof(Msg, has_field / field_count / which_oneof) */
    uint16_t data_size;         /* sizeof(element) or max string/bytes capacity */
    uint16_t array_size;        /* max_count for repeated/map (1 for singular) */
    const ob_msgdesc_t *submsg; /* Non-NULL for nested messages and map entries */
} ob_field_t;

struct ob_msgdesc {
    const ob_field_t *fields;   /* Sorted by tag ascending */
    uint16_t field_count;
    uint16_t struct_size;
};

typedef struct ob_ostream ob_ostream_t;
typedef struct ob_istream ob_istream_t;

struct ob_ostream {
    /* If callback == NULL, acts as a sizing stream (increments bytes_written only). */
    bool (*callback)(ob_ostream_t *stream, const uint8_t *buf, size_t count);
    void *state;
    size_t max_size;       /* Maximum bytes to write (SIZE_MAX = unlimited) */
    size_t bytes_written;
    const char *errmsg;
};

struct ob_istream {
    bool (*callback)(ob_istream_t *stream, uint8_t *buf, size_t count);
    void *state;
    size_t bytes_left;
    const char *errmsg;
};

#ifdef __cplusplus
}
#endif

#endif /* OMNIBUF_NANO_COMMON_H */
