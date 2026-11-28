#include "ob_varint.h"
#include "ob_encode.h"
#include "ob_decode.h"

uint64_t ob_zigzag_encode(int64_t n) {
    /* Avoid UB from left-shifting negative signed integers in C99. */
    uint64_t u = (uint64_t)n;
    uint64_t mask = (n < 0) ? UINT64_MAX : 0u;
    return (u << 1) ^ mask;
}

int64_t ob_zigzag_decode(uint64_t u) {
    /* Avoid UB/implementation-defined cast on INT64_MIN. */
    if ((u & 1u) != 0u) {
        return -(int64_t)(u >> 1) - 1;
    }
    return (int64_t)(u >> 1);
}

size_t ob_varint_size(uint64_t v) {
    if (v <= 127u) {
        return 1u;
    }
    if (v <= 16383u) {
        return 2u;
    }
    if (v <= 536870911UL) {
        return 4u;
    }
    return 9u;
}

bool ob_varint_encode(ob_ostream_t *stream, uint64_t v) {
    uint8_t buf[9];
    size_t len;

    if (stream->callback == NULL) {
        return ob_write(stream, NULL, ob_varint_size(v));
    }

    if (v <= 127u) {
        buf[0] = (uint8_t)((v << 1) | 1u);
        len = 1u;
    } else if (v <= 16383u) {
        uint16_t u = (uint16_t)((v << 2) | 2u);
        buf[0] = (uint8_t)(u & 0xFFu);
        buf[1] = (uint8_t)((u >> 8) & 0xFFu);
        len = 2u;
    } else if (v <= 536870911UL) {
        uint32_t u = (uint32_t)((v << 3) | 4u);
        buf[0] = (uint8_t)(u & 0xFFu);
        buf[1] = (uint8_t)((u >> 8) & 0xFFu);
        buf[2] = (uint8_t)((u >> 16) & 0xFFu);
        buf[3] = (uint8_t)((u >> 24) & 0xFFu);
        len = 4u;
    } else {
        buf[0] = 0x00u;
        buf[1] = (uint8_t)(v & 0xFFu);
        buf[2] = (uint8_t)((v >> 8) & 0xFFu);
        buf[3] = (uint8_t)((v >> 16) & 0xFFu);
        buf[4] = (uint8_t)((v >> 24) & 0xFFu);
        buf[5] = (uint8_t)((v >> 32) & 0xFFu);
        buf[6] = (uint8_t)((v >> 40) & 0xFFu);
        buf[7] = (uint8_t)((v >> 48) & 0xFFu);
        buf[8] = (uint8_t)((v >> 56) & 0xFFu);
        len = 9u;
    }

    return ob_write(stream, buf, len);
}

bool ob_varint_decode(ob_istream_t *stream, uint64_t *out) {
    uint8_t b0 = 0;
    if (!ob_read(stream, &b0, 1u)) {
        return false;
    }

    if ((b0 & 1u) != 0u) {
        *out = (uint64_t)(b0 >> 1);
        return true;
    }
    if ((b0 & 2u) != 0u) {
        uint8_t b1 = 0;
        if (!ob_read(stream, &b1, 1u)) {
            return false;
        }
        uint16_t u = (uint16_t)((uint16_t)b0 | ((uint16_t)b1 << 8));
        *out = (uint64_t)(u >> 2);
        return true;
    }
    if ((b0 & 4u) != 0u) {
        uint8_t rest[3];
        if (!ob_read(stream, rest, 3u)) {
            return false;
        }
        uint32_t u = (uint32_t)b0
                   | ((uint32_t)rest[0] << 8)
                   | ((uint32_t)rest[1] << 16)
                   | ((uint32_t)rest[2] << 24);
        *out = (uint64_t)(u >> 3);
        return true;
    }
    if (b0 == 0x00u) {
        uint8_t rest[8];
        if (!ob_read(stream, rest, 8u)) {
            return false;
        }
        uint64_t u = (uint64_t)rest[0]
                   | ((uint64_t)rest[1] << 8)
                   | ((uint64_t)rest[2] << 16)
                   | ((uint64_t)rest[3] << 24)
                   | ((uint64_t)rest[4] << 32)
                   | ((uint64_t)rest[5] << 40)
                   | ((uint64_t)rest[6] << 48)
                   | ((uint64_t)rest[7] << 56);
        *out = u;
        return true;
    }

    OB_RETURN_ERROR(stream, "reserved varint prefix");
}

bool ob_valid_utf8(const uint8_t *buf, size_t len) {
    size_t i = 0;
    while (i < len) {
        uint8_t c = buf[i];
        if (c < 0x80u) {
            i += 1u;
        } else if ((c & 0xE0u) == 0xC0u) {
            /* 2-byte sequence: reject overlong (c < 0xC2) */
            if (c < 0xC2u || i + 1u >= len) {
                return false;
            }
            if ((buf[i + 1u] & 0xC0u) != 0x80u) {
                return false;
            }
            i += 2u;
        } else if ((c & 0xF0u) == 0xE0u) {
            /* 3-byte sequence: reject overlong and UTF-16 surrogates (U+D800..U+DFFF) */
            if (i + 2u >= len) {
                return false;
            }
            uint8_t c1 = buf[i + 1u];
            uint8_t c2 = buf[i + 2u];
            if ((c1 & 0xC0u) != 0x80u || (c2 & 0xC0u) != 0x80u) {
                return false;
            }
            if (c == 0xE0u && c1 < 0xA0u) {
                return false;
            }
            if (c == 0xEDu && c1 >= 0xA0u) {
                return false;
            }
            i += 3u;
        } else if ((c & 0xF8u) == 0xF0u) {
            /* 4-byte sequence: reject overlong and > U+10FFFF */
            if (c > 0xF4u || i + 3u >= len) {
                return false;
            }
            uint8_t c1 = buf[i + 1u];
            uint8_t c2 = buf[i + 2u];
            uint8_t c3 = buf[i + 3u];
            if ((c1 & 0xC0u) != 0x80u || (c2 & 0xC0u) != 0x80u || (c3 & 0xC0u) != 0x80u) {
                return false;
            }
            if (c == 0xF0u && c1 < 0x90u) {
                return false;
            }
            if (c == 0xF4u && c1 >= 0x90u) {
                return false;
            }
            i += 4u;
        } else {
            return false;
        }
    }
    return true;
}
