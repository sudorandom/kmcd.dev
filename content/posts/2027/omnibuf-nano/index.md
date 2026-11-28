---
title: "OmniBuf on embedded devices"
date: "2027-02-09T10:00:00Z"
categories: ["article", "engineering"]
tags: ["omnibuf", "protobuf", "c", "embedded", "arm", "qemu", "mise", "serialization"]
keywords: ["omnibuf-nano", "nanopb alternative", "c99 serialization", "cortex-m3 qemu", "embedded protobuf", "zero malloc c99", "mise embedded toolchain"]
description: "Testing omnibuf-nano (a zero-malloc C99 OmniBuf library) on an emulated ARM Cortex-M3 with mise and QEMU."
cover: "cover.svg"
images: ["/posts/omnibuf-nano/cover.svg"]
slug: "omnibuf-nano"
type: "posts"
canonical_url: https://kmcd.dev/posts/omnibuf-nano/
devtoSkip: true
draft: true
---

I'm going to cover some of the [OmniBuf](/posts/omnibuf/) libraries in their own posts. Today, we're starting with the weirdest one: [omnibuf-nano](https://github.com/omnibuf/omnibuf-nano).

The other eleven OmniBuf libraries target normal computers and servers. `omnibuf-nano` is written in C99 for microcontrollers where you don't have an OS or even `malloc` and you have kilobytes of RAM instead of gigabytes.

{{< github-repo repo="omnibuf/omnibuf-nano" description="Zero-malloc C99 runtime and protoc plugin for embedded OmniBuf targets." >}}

I don't do embedded work professionally, even though I've always wanted to mess around with it. Most of the design here borrows from [Nanopb](https://jpa.kapsi.fi/nanopb/), which is the go-to Protobuf library for microcontrollers. I don't even own a microcontroller, but apparently that isn't much of a problem with QEMU.

Let's make a little weather station. It'll collect temperature, humidity, and pressure readings and spit them out over a serial port. I also want to see how much RAM and Flash OmniBuf needs compared to Nanopb.

## Emulating a chip

I've included a [mise.toml](https://github.com/sudorandom/kmcd.dev/blob/main/content/posts/2027/omnibuf-nano/c/mise.toml) with the [demo](https://github.com/sudorandom/kmcd.dev/tree/main/content/posts/2027/omnibuf-nano/c) that installs the ARM compiler, QEMU, `buf`, the `omnibuf-nano` code generator, and the `omnibuf` CLI. Run `mise install` to grab everything.

A lot of the bare-metal QEMU examples I found use the [`lm3s6965evb`](https://www.qemu.org/docs/master/system/arm/stellaris.html), a board with an ARM Cortex-M3, so that's what I'm using too.

There are two compiler flags worth pointing out here. `--specs=nano.specs` gets us the smaller version of newlib that comes with `arm-none-eabi-gcc`. With `-nostartfiles`, we get to write our own startup code.

{{< figure src="cortex-m3.jpg" width="520px" class="center" alt="Close-up photo of an STMicroelectronics STM32F107RBT6 ARM Cortex-M3 microcontroller mounted on a green PCB" caption="An ARM Cortex-M3 microcontroller (STM32F107RBT6) on a MikroTik RouterBoard; [photo by Raimond Spekking, CC BY-SA 4.0](https://commons.wikimedia.org/w/index.php?curid=185379194)." loading="lazy" >}}

I'm using two UARTs. `UART0` is effectively my `printf` debugger and `UART1` is where the weather station sends its OmniBuf messages. QEMU lets me redirect that second UART straight into a file on my laptop (`out/telemetry.obuf`).

The startup code and serial driver are in `src/`.

{{< diagram caption="Cortex-M3 QEMU setup: Flash holds the code and field tables, SRAM holds the message struct, and UART1 writes raw OmniBuf bytes to disk." >}}
{{< image src="cortex-m3-layout.svg" class="center" inline="true" >}}
{{< /diagram >}}

## Working without malloc

Normally I'd be able to put as many readings as I want into a Protobuf repeated field. Here I need to decide how many readings the struct has room for before I even compile the program.

Nanopb solves this with a separate `.options` file, and I've borrowed that idea for `omnibuf-nano`. Here are [proto/sensor.proto](https://github.com/sudorandom/kmcd.dev/blob/main/content/posts/2027/omnibuf-nano/c/proto/sensor.proto) and [proto/sensor.options](https://github.com/sudorandom/kmcd.dev/blob/main/content/posts/2027/omnibuf-nano/c/proto/sensor.options):

{{% render-code file="c/proto/sensor.proto" language="protobuf" %}}

{{% render-code file="c/proto/sensor.options" language="text" %}}

None of these options affect the generated Go code. My server can happily encode a message with five readings using OmniBuf, but this poor little microcontroller only has room for four.

`buf generate` turns those into fixed-size C structs in `gen/sensor.ob.h`:

```c
typedef int32_t sensor_AlertLevel;
enum {
    sensor_AlertLevel_ALERT_LEVEL_OK = 0,
    sensor_AlertLevel_ALERT_LEVEL_WARN = 1,
    sensor_AlertLevel_ALERT_LEVEL_CRITICAL = 2
};

typedef struct {
    uint32_t device_id;
    char node_name[24u];
    uint32_t battery_mv;
    sensor_AlertLevel alert;
    uint16_t readings_count;
    sensor_Reading readings[4u];
    uint8_t firmware_sha[8u];
    uint16_t calibration_count;
    sensor_TelemetryFrame_CalibrationEntry calibration[4u];
} sensor_TelemetryFrame;
```

No pointers! There's room for four readings in there, and it's up to me to check `readings_count` before adding another one. The map is an array of key/value pairs, which feels particularly odd coming from Go. I also used `fixed_length:true` to make `firmware_sha` a plain 8-byte array.

`sizeof(sensor_TelemetryFrame)` is 164 bytes. If I want room for eight readings instead of four, it'll be 212 bytes. That's another 48 bytes of RAM gone before I've taken a single temperature reading.

## Running the demo

In [src/main.c](https://github.com/sudorandom/kmcd.dev/blob/main/content/posts/2027/omnibuf-nano/c/src/main.c), encoding to a stack buffer and decoding back looks like this:

{{% render-code file="c/src/main.c" language="c" start="// start: buffer-roundtrip" end="// end: buffer-roundtrip" %}}

`make run` boots the firmware in QEMU and prints five lines to `UART0`:

```text
sizeof(TelemetryFrame)=164 sizeof(Reading)=12
buffer: 86 bytes (node=roof-north-01 temp=-425)
uart1: 82 bytes (40 writes)
v2 decode: 240 bytes -> device_id=7 node=field-west-04 battery_mv=3910
bounds check: err="string exceeds max_size"
```

Why did we get 86 bytes in the buffer and only 82 bytes out of the UART?

When I write to a buffer, I can leave two bytes for a container's length, encode the contents, and then go back and fill in the length. I can't go back and change bytes I've already sent over a serial port, so the callback encoder calculates the lengths first. In this case, it only needs one byte for each length.

The extra work means I can get rid of the `uint8_t wire[160]` from the earlier example entirely. I'd need to add some sort of message framing if I wanted to keep sending readings over the UART, but for this demo we're just writing one message to a file.

Now for my favorite part. Those 82 bytes are sitting in a file on my laptop and I can just do this:

```bash
$ omnibuf out/telemetry.obuf
{
  "field_1": 42,
  "field_2": "roof-north-01",
  "field_3": 3740,
  "field_4": 1,
  "field_5": [
    {
      "field_1": -425,
      "field_2": 81,
      "field_3": 101325
    },
    {
      "field_1": -410,
      "field_2": 80,
      "field_3": 101318
    }
  ],
  "field_6": "3q2+7wEjRWc=",
  "field_7": {
    "hum_off": 3,
    "temp_off": -12
  }
}
```

Notice that I didn't give the CLI a `.proto` file. Even the map comes out looking like a map! I can clean up those ugly `field_1` names by passing in the output of `buf build`:

```bash
$ buf build -o out/schema.pb
$ omnibuf -s out/schema.pb -t sensor.TelemetryFrame out/telemetry.obuf
{
  "device_id": 42,
  "node_name": "roof-north-01",
  "battery_mv": 3740,
  "alert": "ALERT_LEVEL_WARN",
  "readings": [
    {
      "celsius_x100": -425,
      "humidity_pct": 81,
      "pressure_pa": 101325
    },
    {
      "celsius_x100": -410,
      "humidity_pct": 80,
      "pressure_pa": 101318
    }
  ],
  "firmware_sha": "3q2+7wEjRWc=",
  "calibration": {
    "hum_off": 3,
    "temp_off": -12
  }
}
```

Back in the QEMU output, there's a test called `v2 decode`. I wanted to see what would happen if a newer version of my server started sending fields that the firmware didn't know about. The test packet is 240 bytes long and includes a metadata block (field `#0`) and an extra field (`#99`):

{{% render-code file="c/src/main.c" language="c" start="// start: forward-compat" end="// end: forward-compat" %}}

For this test I'm simulating a receive FIFO. `ob_decode` reads from it as needed, and when it gets to field `#99`, it reads and discards the contents 32 bytes at a time. There's no need to allocate a buffer big enough for whatever some future version of the server might send.

What happens if I send a `node_name` that won't fit into `char node_name[24]`? That's what the last line of the QEMU output is testing. I get a `string exceeds max_size` error, and the decoder refuses to write past the end of the array.

## Size and speed

Since I've borrowed so much from Nanopb, I figured I should compile a Nanopb version of the weather station too. I also want to know how much Flash I can save if my little weather station never needs to decode a message.

Both builds use `arm-none-eabi-gcc` 15.2.1 with these flags:

```text
-mcpu=cortex-m3 -mthumb -Os --specs=nano.specs
-ffunction-sections -fdata-sections -fshort-enums
-Wl,--gc-sections
```

| Cortex-M3 Metric (`TelemetryFrame`) | `omnibuf-nano` 0.1.3 | `nanopb` 0.4.9.3 |
| :--- | ---: | ---: |
| Runtime `.o` text (encode + decode) | 7,876 B | **6,536 B** |
| Runtime `.o` text (encode-only) | 3,327 B | **2,773 B** |
| Generated schema `.o` text (`sensor.*.o`) | 216 B | **172 B** |
| Linked firmware (encode + decode) | 8,706 B | **7,472 B** |
| Linked firmware (encode-only) | 4,030 B | **3,528 B** |
| Static RAM (`.data` + `.bss`) | 0 B | 0 B |
| Struct size (`sizeof`) | 164 B | **160 B** |
| Wire size (`int32`, original schema) | **82 B** | 106 B |
| Wire size (`sint32`, modified schema) | 82 B | **81 B** |
| Peak stack estimate (encode / decode) | ~470 B / ~1.1 KB | **~440 B / ~640 B** |

A few notes on how I got these numbers:

- For the `.o` files, I measured the `.text` sizes before linking. The encode-only numbers exclude the decoder object file.
- The linked firmware includes the startup code and UART driver, with unused sections removed by the linker.
- I used GCC's `-fstack-usage` to estimate stack usage across two levels of message nesting. This doesn't include the message structs or buffers allocated by the caller.

The firmware from the earlier demo comes out to 10 KB because I've also stuffed those two extra test messages into it.

Nanopb has me beat on both Flash and RAM here. I'm particularly unhappy with the amount of stack space the OmniBuf decoder needs. One of the functions in its decode path takes up 496 bytes for every level of nesting. Nanopb gets away with 144 bytes in its corresponding function. I'll need to spend some time on that.

I was a little surprised by the wire sizes. There are three negative `int32` values in the weather station's sample message, and Protobuf uses 10 bytes for each value before you even count the field tags. That's an easy fix: change the temperature and calibration values from `int32` to `sint32`. With that change, Nanopb produces 81 bytes instead of 106. OmniBuf produces 82 bytes regardless, since it uses zigzag encoding for signed integers by default.

The only performance numbers I have right now are from my M1 Pro, unfortunately. I've been using a `PeerUpdate` message to benchmark both libraries in `benchmarks/peer_bench.c`. The generated structs have equivalent field limits and are both 976 bytes:

| Library | Encode | Decode | Wire Size | Struct Size |
| :--- | ---: | ---: | ---: | ---: |
| **omnibuf-nano (C99)** | **1,345 ns** | **1,004 ns** | 358 B | 976 B |
| **nanopb (C99)** | 1,481 ns | 1,227 ns | **341 B** | 976 B |

If I were shipping an embedded product today, I'd probably use Nanopb. For one thing, I'm comparing a library that's been running on actual microcontrollers for over a decade with one that I've only run in QEMU. Nanopb also has some nice tricks for decoding repeated fields that I haven't implemented yet.

I'm still pretty excited that `omnibuf out/telemetry.obuf` works.

Anyway, does anyone want to try this on actual hardware? `omnibuf-nano` is portable C99, so the QEMU-specific code shouldn't be needed. I'd love to hear from someone with an STM32, RP2040, or ESP32.

You can try the QEMU demo yourself with the [code from GitHub](https://github.com/sudorandom/kmcd.dev/tree/main/content/posts/2027/omnibuf-nano/c):

```bash
mise install
make all
make decode-uart
make decode-uart-schema
```

Docs and options reference live at [omnibuf.com/nano](https://omnibuf.com/nano).
