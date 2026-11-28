#ifndef UART_H
#define UART_H

#include <stddef.h>
#include <stdint.h>

/* Stellaris LM3S6965EVB PL011 UART base addresses */
#define UART0_BASE 0x4000C000UL
#define UART1_BASE 0x4000D000UL

void uart_putc(uintptr_t base, uint8_t byte);
void uart_write(uintptr_t base, const uint8_t *buf, size_t len);
void uart_puts(uintptr_t base, const char *s);
void uart_put_u32(uintptr_t base, uint32_t v);
void uart_put_i32(uintptr_t base, int32_t v);
void uart_put_hex32(uintptr_t base, uint32_t v);
void uart_put_hexdump(uintptr_t base, const uint8_t *buf, size_t len);

#endif /* UART_H */
