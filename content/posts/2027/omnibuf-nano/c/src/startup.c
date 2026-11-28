#include <stdint.h>

extern uint32_t _estack;
extern uint32_t _sidata;
extern uint32_t _sdata;
extern uint32_t _edata;
extern uint32_t _sbss;
extern uint32_t _ebss;

extern int main(void);

static void semihost_exit(int code) {
    /* ARM semihosting SYS_EXIT (0x18).
     * 0x20026 = ADP_Stopped_ApplicationExit (clean exit 0)
     * 0x20023 = ADP_Stopped_RunTimeErrorUnknown (non-zero exit)
     */
    register uint32_t r0 __asm__("r0") = 0x18u;
    register uint32_t r1 __asm__("r1") = (code == 0) ? 0x20026u : 0x20023u;
    __asm__ volatile("bkpt #0xAB" : : "r"(r0), "r"(r1) : "memory");
    for (;;) {
    }
}

void Reset_Handler(void) {
    uint32_t *src = &_sidata;
    uint32_t *dst = &_sdata;
    while (dst < &_edata) {
        *dst++ = *src++;
    }
    dst = &_sbss;
    while (dst < &_ebss) {
        *dst++ = 0u;
    }

    int rc = main();
    semihost_exit(rc);
}

static void Default_Handler(void) {
    semihost_exit(1);
}

__attribute__((section(".isr_vector"), used))
const uint32_t vector_table[16] = {
    (uint32_t)(uintptr_t)&_estack,
    (uint32_t)(uintptr_t)Reset_Handler,
    (uint32_t)(uintptr_t)Default_Handler, /* NMI */
    (uint32_t)(uintptr_t)Default_Handler, /* HardFault */
    (uint32_t)(uintptr_t)Default_Handler, /* MemManage */
    (uint32_t)(uintptr_t)Default_Handler, /* BusFault */
    (uint32_t)(uintptr_t)Default_Handler, /* UsageFault */
    0u, 0u, 0u, 0u,
    (uint32_t)(uintptr_t)Default_Handler, /* SVCall */
    (uint32_t)(uintptr_t)Default_Handler, /* DebugMon */
    0u,
    (uint32_t)(uintptr_t)Default_Handler, /* PendSV */
    (uint32_t)(uintptr_t)Default_Handler  /* SysTick */
};
