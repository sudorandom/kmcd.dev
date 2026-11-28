#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>
#include <string.h>

#include "ob_decode.h"
#include "ob_encode.h"
#include "sensor.ob.h"
#include "test_vectors.h"
#include "uart.h"

/* Helper to copy a C string safely into a fixed-size char array. */
static void copy_str(char *dst, size_t cap, const char *src) {
    size_t n = strlen(src);
    if (n >= cap) {
        n = cap - 1u;
    }
    memcpy(dst, src, n);
    dst[n] = '\0';
}

// start: uart-callback
typedef struct {
    uintptr_t uart_base;
    size_t tx_calls;
} uart_sink_t;

static bool uart_write_cb(ob_ostream_t *stream, const uint8_t *buf, size_t count) {
    uart_sink_t *sink = (uart_sink_t *)stream->state;
    uart_write(sink->uart_base, buf, count);
    sink->tx_calls++;
    return true;
}
// end: uart-callback

/* Simulates a byte-at-a-time streaming RX peripheral (such as a UART FIFO). */
typedef struct {
    const uint8_t *data;
    size_t pos;
    size_t len;
    size_t read_calls;
} rx_fifo_t;

static bool rx_fifo_read_cb(ob_istream_t *stream, uint8_t *buf, size_t count) {
    rx_fifo_t *fifo = (rx_fifo_t *)stream->state;
    if (fifo->pos + count > fifo->len) {
        return false;
    }
    memcpy(buf, fifo->data + fifo->pos, count);
    fifo->pos += count;
    fifo->read_calls++;
    return true;
}

static sensor_TelemetryFrame build_sample_frame(void) {
    // start: build-frame
    sensor_TelemetryFrame frame = sensor_TelemetryFrame_init_zero;
    frame.device_id = 42u;
    copy_str(frame.node_name, sizeof(frame.node_name), "roof-north-01");
    frame.battery_mv = 3740u;
    frame.alert = sensor_AlertLevel_ALERT_LEVEL_WARN;

    frame.readings_count = 2u;
    frame.readings[0].celsius_x100 = -425; /* -4.25 C */
    frame.readings[0].humidity_pct = 81u;
    frame.readings[0].pressure_pa = 101325u;

    frame.readings[1].celsius_x100 = -410; /* -4.10 C */
    frame.readings[1].humidity_pct = 80u;
    frame.readings[1].pressure_pa = 101318u;

    static const uint8_t sha[8] = {0xdeu, 0xadu, 0xbeu, 0xefu, 0x01u, 0x23u, 0x45u, 0x67u};
    memcpy(frame.firmware_sha, sha, sizeof(frame.firmware_sha));

    frame.calibration_count = 2u;
    copy_str(frame.calibration[0].key, sizeof(frame.calibration[0].key), "temp_off");
    frame.calibration[0].value = -12;
    copy_str(frame.calibration[1].key, sizeof(frame.calibration[1].key), "hum_off");
    frame.calibration[1].value = 3;
    // end: build-frame
    return frame;
}

int main(void) {
    /* Verify descriptor sits in Flash (.rodata), not SRAM (>= 0x20000000). */
    if ((uintptr_t)&sensor_TelemetryFrame_msg >= 0x20000000UL) {
        uart_puts(UART0_BASE, "err: descriptor not in flash\n");
        return 1;
    }

    uart_puts(UART0_BASE, "sizeof(TelemetryFrame)=");
    uart_put_u32(UART0_BASE, (uint32_t)sizeof(sensor_TelemetryFrame));
    uart_puts(UART0_BASE, " sizeof(Reading)=");
    uart_put_u32(UART0_BASE, (uint32_t)sizeof(sensor_Reading));
    uart_puts(UART0_BASE, "\n");

    /* Buffer encode + round-trip decode */
    sensor_TelemetryFrame out = build_sample_frame();

    // start: buffer-roundtrip
    uint8_t wire[160];
    ob_ostream_t os = ob_ostream_from_buffer(wire, sizeof(wire));
    if (!ob_encode(&os, &sensor_TelemetryFrame_msg, &out)) {
        uart_puts(UART0_BASE, "encode err: ");
        uart_puts(UART0_BASE, os.errmsg ? os.errmsg : "unknown");
        uart_puts(UART0_BASE, "\n");
        return 1;
    }

    sensor_TelemetryFrame in;
    ob_istream_t is = ob_istream_from_buffer(wire, os.bytes_written);
    if (!ob_decode(&is, &sensor_TelemetryFrame_msg, &in)) {
        uart_puts(UART0_BASE, "decode err: ");
        uart_puts(UART0_BASE, is.errmsg ? is.errmsg : "unknown");
        uart_puts(UART0_BASE, "\n");
        return 1;
    }
    // end: buffer-roundtrip

    if (in.device_id != 42u ||
        strcmp(in.node_name, "roof-north-01") != 0 ||
        in.battery_mv != 3740u ||
        in.alert != sensor_AlertLevel_ALERT_LEVEL_WARN ||
        in.readings_count != 2u ||
        in.readings[0].celsius_x100 != -425 ||
        in.readings[1].pressure_pa != 101318u ||
        memcmp(in.firmware_sha, out.firmware_sha, 8u) != 0 ||
        in.calibration_count != 2u ||
        strcmp(in.calibration[0].key, "temp_off") != 0 ||
        in.calibration[0].value != -12) {
        uart_puts(UART0_BASE, "err: roundtrip mismatch\n");
        return 1;
    }

    uart_puts(UART0_BASE, "buffer: ");
    uart_put_u32(UART0_BASE, (uint32_t)os.bytes_written);
    uart_puts(UART0_BASE, " bytes (node=");
    uart_puts(UART0_BASE, in.node_name);
    uart_puts(UART0_BASE, " temp=");
    uart_put_i32(UART0_BASE, in.readings[0].celsius_x100);
    uart_puts(UART0_BASE, ")\n");

    /* Zero-buffer hardware UART1 streaming */
    // start: uart-stream
    uart_sink_t sink = { .uart_base = UART1_BASE, .tx_calls = 0u };
    ob_ostream_t uart_os = {
        .callback = uart_write_cb,
        .state = &sink,
        .max_size = SIZE_MAX,
        .bytes_written = 0u,
        .errmsg = NULL
    };
    if (!ob_encode(&uart_os, &sensor_TelemetryFrame_msg, &out)) {
        uart_puts(UART0_BASE, "uart encode err: ");
        uart_puts(UART0_BASE, uart_os.errmsg ? uart_os.errmsg : "unknown");
        uart_puts(UART0_BASE, "\n");
        return 1;
    }
    // end: uart-stream

    if (uart_os.bytes_written == 0u || uart_os.bytes_written > os.bytes_written) {
        uart_puts(UART0_BASE, "err: unexpected uart stream length\n");
        return 1;
    }

    uart_puts(UART0_BASE, "uart1: ");
    uart_put_u32(UART0_BASE, (uint32_t)uart_os.bytes_written);
    uart_puts(UART0_BASE, " bytes (");
    uart_put_u32(UART0_BASE, (uint32_t)sink.tx_calls);
    uart_puts(UART0_BASE, " writes)\n");

    /* Streaming decode of v2 server packet (Field #0 + unknown field #99) */
    // start: forward-compat
    rx_fifo_t rx = {
        .data = v2_server_packet,
        .pos = 0u,
        .len = sizeof(v2_server_packet),
        .read_calls = 0u
    };
    ob_istream_t rx_is = {
        .callback = rx_fifo_read_cb,
        .state = &rx,
        .bytes_left = sizeof(v2_server_packet),
        .errmsg = NULL
    };

    sensor_TelemetryFrame v2;
    if (!ob_decode(&rx_is, &sensor_TelemetryFrame_msg, &v2)) {
        uart_puts(UART0_BASE, "v2 decode err: ");
        uart_puts(UART0_BASE, rx_is.errmsg ? rx_is.errmsg : "unknown");
        uart_puts(UART0_BASE, "\n");
        return 1;
    }
    // end: forward-compat

    if (v2.device_id != 7u ||
        strcmp(v2.node_name, "field-west-04") != 0 ||
        v2.battery_mv != 3910u ||
        v2.alert != sensor_AlertLevel_ALERT_LEVEL_CRITICAL) {
        uart_puts(UART0_BASE, "err: v2 mismatch\n");
        return 1;
    }

    uart_puts(UART0_BASE, "v2 decode: ");
    uart_put_u32(UART0_BASE, (uint32_t)sizeof(v2_server_packet));
    uart_puts(UART0_BASE, " bytes -> device_id=");
    uart_put_u32(UART0_BASE, v2.device_id);
    uart_puts(UART0_BASE, " node=");
    uart_puts(UART0_BASE, v2.node_name);
    uart_puts(UART0_BASE, " battery_mv=");
    uart_put_u32(UART0_BASE, v2.battery_mv);
    uart_puts(UART0_BASE, "\n");

    /* Bounds rejection on oversized string (> max_size:24) */
    sensor_TelemetryFrame bad;
    ob_istream_t bad_is = ob_istream_from_buffer(oversized_name_packet, sizeof(oversized_name_packet));
    if (ob_decode(&bad_is, &sensor_TelemetryFrame_msg, &bad)) {
        uart_puts(UART0_BASE, "err: oversized string accepted\n");
        return 1;
    }
    uart_puts(UART0_BASE, "bounds check: err=\"");
    uart_puts(UART0_BASE, bad_is.errmsg ? bad_is.errmsg : "none");
    uart_puts(UART0_BASE, "\"\n");

    return 0;
}
