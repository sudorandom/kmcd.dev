#include "uart.h"

#define UART_DR_OFFSET 0x00UL
#define UART_FR_OFFSET 0x18UL
#define UART_FR_TXFF   (1UL << 5)

void uart_putc(uintptr_t base, uint8_t byte) {
    volatile uint32_t *fr = (volatile uint32_t *)(base + UART_FR_OFFSET);
    volatile uint32_t *dr = (volatile uint32_t *)(base + UART_DR_OFFSET);
    while ((*fr & UART_FR_TXFF) != 0UL) {
    }
    *dr = (uint32_t)byte;
}

void uart_write(uintptr_t base, const uint8_t *buf, size_t len) {
    for (size_t i = 0; i < len; i++) {
        uart_putc(base, buf[i]);
    }
}

void uart_puts(uintptr_t base, const char *s) {
    while (*s != '\0') {
        uart_putc(base, (uint8_t)*s++);
    }
}

void uart_put_u32(uintptr_t base, uint32_t v) {
    char tmp[12];
    size_t n = 0;
    if (v == 0u) {
        uart_putc(base, '0');
        return;
    }
    while (v > 0u) {
        tmp[n++] = (char)('0' + (v % 10u));
        v /= 10u;
    }
    while (n > 0u) {
        uart_putc(base, (uint8_t)tmp[--n]);
    }
}

void uart_put_i32(uintptr_t base, int32_t v) {
    if (v < 0) {
        uart_putc(base, '-');
        /* Avoid UB on INT32_MIN */
        uint32_t mag = (uint32_t)(-(v + 1)) + 1u;
        uart_put_u32(base, mag);
        return;
    }
    uart_put_u32(base, (uint32_t)v);
}

void uart_put_hex32(uintptr_t base, uint32_t v) {
    static const char hex[] = "0123456789abcdef";
    uart_puts(base, "0x");
    for (int shift = 28; shift >= 0; shift -= 4) {
        uart_putc(base, (uint8_t)hex[(v >> (unsigned)shift) & 0xFu]);
    }
}

void uart_put_hexdump(uintptr_t base, const uint8_t *buf, size_t len) {
    static const char hex[] = "0123456789abcdef";
    for (size_t i = 0; i < len; i++) {
        if (i > 0u) {
            uart_putc(base, ' ');
        }
        uart_putc(base, (uint8_t)hex[(buf[i] >> 4) & 0xFu]);
        uart_putc(base, (uint8_t)hex[buf[i] & 0xFu]);
    }
}
