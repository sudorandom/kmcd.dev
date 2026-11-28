CC      := arm-none-eabi-gcc
SIZE    := arm-none-eabi-size
OBJDUMP := arm-none-eabi-objdump
NM      := arm-none-eabi-nm
QEMU    := qemu-system-arm

CFLAGS := -mcpu=cortex-m3 -mthumb -Os --specs=nano.specs \
          -ffunction-sections -fdata-sections -fshort-enums \
          -Wall -Wextra -Werror -std=c99 -pedantic \
          -Wshadow -Wundef -Wcast-align -Wstrict-aliasing=2 \
          -Iruntime -Igen -Isrc

LDFLAGS := -T src/linker.ld -Wl,--gc-sections -nostartfiles

RUNTIME_SRCS := runtime/ob_varint.c runtime/ob_encode.c runtime/ob_decode.c
GEN_SRCS     := gen/sensor.ob.c
APP_SRCS     := src/startup.c src/uart.c src/main.c

BUILD_DIR := out
ELF       := $(BUILD_DIR)/firmware.elf
UART_OUT  := $(BUILD_DIR)/telemetry.obuf

.PHONY: all generate build size run decode-uart decode-uart-schema clean

all: build size run

generate:
	buf generate

$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

$(BUILD_DIR)/ob_varint.o: runtime/ob_varint.c | $(BUILD_DIR)
	$(CC) $(CFLAGS) -c $< -o $@

$(BUILD_DIR)/ob_encode.o: runtime/ob_encode.c | $(BUILD_DIR)
	$(CC) $(CFLAGS) -c $< -o $@

$(BUILD_DIR)/ob_decode.o: runtime/ob_decode.c | $(BUILD_DIR)
	$(CC) $(CFLAGS) -c $< -o $@

$(BUILD_DIR)/sensor.ob.o: gen/sensor.ob.c | $(BUILD_DIR)
	$(CC) $(CFLAGS) -c $< -o $@

$(BUILD_DIR)/startup.o: src/startup.c | $(BUILD_DIR)
	$(CC) $(CFLAGS) -c $< -o $@

$(BUILD_DIR)/uart.o: src/uart.c | $(BUILD_DIR)
	$(CC) $(CFLAGS) -c $< -o $@

$(BUILD_DIR)/main.o: src/main.c gen/sensor.ob.h src/test_vectors.h | $(BUILD_DIR)
	$(CC) $(CFLAGS) -c $< -o $@

OBJS := $(BUILD_DIR)/startup.o \
        $(BUILD_DIR)/uart.o \
        $(BUILD_DIR)/ob_varint.o \
        $(BUILD_DIR)/ob_encode.o \
        $(BUILD_DIR)/ob_decode.o \
        $(BUILD_DIR)/sensor.ob.o \
        $(BUILD_DIR)/main.o

$(ELF): $(OBJS) src/linker.ld
	$(CC) $(CFLAGS) $(OBJS) $(LDFLAGS) -o $@

build: $(ELF)

size: $(ELF)
	$(SIZE) -A $(ELF)
	@echo ""
	$(SIZE) $(BUILD_DIR)/ob_varint.o $(BUILD_DIR)/ob_encode.o $(BUILD_DIR)/ob_decode.o $(BUILD_DIR)/sensor.ob.o $(ELF)

run: $(ELF)
	rm -f $(UART_OUT)
	$(QEMU) -M lm3s6965evb -nographic -monitor none \
		-semihosting-config enable=on,target=native \
		-serial stdio \
		-serial file:$(UART_OUT) \
		-kernel $(ELF)

decode-uart: $(UART_OUT)
	omnibuf $(UART_OUT)

decode-uart-schema: $(UART_OUT)
	buf build -o $(BUILD_DIR)/schema.pb
	omnibuf -s $(BUILD_DIR)/schema.pb -t sensor.TelemetryFrame $(UART_OUT)

clean:
	rm -rf $(BUILD_DIR)
