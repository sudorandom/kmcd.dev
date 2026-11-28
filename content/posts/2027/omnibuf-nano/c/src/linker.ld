/* Linker script for TI Stellaris LM3S6965EVB (ARM Cortex-M3) */
MEMORY {
    FLASH (rx)  : ORIGIN = 0x00000000, LENGTH = 256K
    SRAM  (rwx) : ORIGIN = 0x20000000, LENGTH = 64K
}

_estack = ORIGIN(SRAM) + LENGTH(SRAM);

SECTIONS {
    .isr_vector : {
        KEEP(*(.isr_vector))
    } > FLASH

    .text : {
        *(.text*)
    } > FLASH

    .rodata : {
        . = ALIGN(4);
        *(.rodata*)
        . = ALIGN(4);
    } > FLASH

    _sidata = LOADADDR(.data);
    .data : {
        . = ALIGN(4);
        _sdata = .;
        *(.data*)
        . = ALIGN(4);
        _edata = .;
    } > SRAM AT > FLASH

    .bss : {
        . = ALIGN(4);
        _sbss = .;
        *(.bss*)
        *(COMMON)
        . = ALIGN(4);
        _ebss = .;
    } > SRAM
}
