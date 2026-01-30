---
categories: ["article"]
tags: ["go", "http2", "protocols", "networking"]
date: "2026-02-20T10:00:00Z"
description: "Decoding HPACK and the evolution of the HTTP header"
cover: "cover.svg"
images: ["/posts/http2-from-scratch-part-3/cover.svg"]
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "HTTP/2 From Scratch: Part 3"
slug: "http2-from-scratch-part-3"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/http2-from-scratch-part-3/
draft: true
series: ["HTTP From Scratch"]
---

In the last two posts, we established a raw TCP connection, navigated the TLS handshake with ALPN to select "h2", and built a parser that can read the 9-byte frames of an HTTP/2 connection. We have a synchronized, acknowledged connection. Now it's time to do what we came for: request a web page.

This is where HTTP/2 departs dramatically from its predecessor. There is no `GET / HTTP/1.1`. Instead, we enter the world of compressed headers, pseudo-headers, and stateful tables. This is the world of HPACK: Header Compression for HTTP/2.

### What is HPACK?

In HTTP/1.1, headers are human-readable text. This is great for debugging but inefficient. The same headers (like `User-Agent`) are sent with every single request, wasting bandwidth. [HPACK (RFC 7541)](https://www.rfc-editor.org/rfc/rfc7541.html) solves this by using several compression strategies. Instead of sending full header names and values, it sends compact, indexed representations.

At its core, HPACK uses two tables to translate between full headers and small integer indices:

1.  **Static Table**: A predefined, read-only table containing 61 of the most common headers. For example, `{':method', 'GET'}` is entry #2. Every HTTP/2 client and server knows this table.
2.  **Dynamic Table**: A small, temporary table that is specific to a single connection. If you send a header that's not in the static table (like a custom `x-request-id`), it can be added to the dynamic table. On the next request, you can just send its index instead of the full header again.

Our journey into HPACK will start with the basics. We'll implement a decoder that understands the static table and indexed headers. We'll leave the dynamic table handling for the next article.

### Decoding Indexed Headers

The simplest form of header compression is the **Indexed Header Field**. When a header to be sent is present in one of the tables, it can be represented by a single integer.

An indexed header byte starts with a `1`. The remaining 7 bits are the start of a variable-length integer representing the index in the tables.

Let's look at our `hpack.go` file. The `Decode` method reads the payload from a `HEADERS` frame. If it sees a byte starting with `1`, it knows it's an indexed header.

```go
func (h *HPACKDecoder) Decode(payload []byte) error {
	fmt.Printf("Decoding %d bytes\n", len(payload))
	for len(payload) > 0 {
		b := payload[0]
		if b&128 == 128 { // Indexed Header Field (starts with 1)
			index, n := decodeInt(payload, 7)
			if n < 0 {
				return fmt.Errorf("failed to decode integer")
			}
			payload = payload[n:]
			header, ok := h.Header(index)
			if !ok {
				return fmt.Errorf("invalid header index: %d", index)
			}
			fmt.Printf("  [Header] %s: %s\n", header.Name, header.Value)
		} else {
			// Other header types (like literals) are not implemented yet.
			// We'll tackle this in the next part.
			return fmt.Errorf("not implemented: literal header field")
		}
	}
	return nil
}
```

The `decodeInt` function handles HPACK's specific integer encoding. It parses the first byte manually to account for the `n`-bit prefix. If the integer overflows that first byte, it uses the standard library's `binary.Uvarint` to efficiently parse the remaining continuation bytes. This hybrid approach correctly handles the HPACK format while leveraging Go's optimized, built-in varint decoder.

With this, our decoder can parse headers that are in the static table. The `hpack.go` file contains the full static table definition.

{{% render-code file="go/hpack.go" language="go" %}}
{{< aside >}}
See the full HPACK implementation: {{< github-link file="go/hpack.go" >}}.
{{</ aside >}}

### Manually Encoding Our First Request

Our current client doesn't have an HPACK *encoder*. To send our first request, we're going to manually craft the byte payload for our `HEADERS` frame. This is a great way to understand how encoding works.

We want to send the following headers for a `GET /` request:
- `:method: GET`
- `:path: /`
- `:scheme: https`
- `:authority: kmcd.dev`

Looking at the static table in RFC 7541, Appendix A:
- `{':method', 'GET'}` is at index 2.
- `{':path', '/'}` is at index 4.
- `{':scheme', 'https}` is at index 7.

The `:authority` header doesn't have a perfect match with both name and value. However, its *name* is at index 1. This means we have to send it as a **Literal Header Field**. This type of header representation has a prefix indicating how it should be handled. For now, we will use "Literal Header Field with Incremental Indexing" (prefix `0100`), which tells the server to use this header and add it to the dynamic table. We will do this manually for now, but in the future we will want create code to manage this for us.

Our `client.go` assembles this payload:

```go
// content/posts/2026/http2-from-scratch-part-3/go/client.go

// ...
	authority := "kmcd.dev"
	requestPayload := []byte{
		0x82, // Index 2: :method: GET
		0x84, // Index 4: :path: /
		0x87, // Index 7: :scheme: https
		0x41, // Index 1 for :authority, with literal value
		byte(len(authority)), // Length of "kmcd.dev"
	}
	requestPayload = append(requestPayload, []byte(authority)...)
// ...
```
Let's break down the bytes:
- `0x82`: `10000010`. Starts with `1`, so it's an indexed header. The integer value is 2. This is `:method: GET`.
- `0x84`: `10000100`. Indexed header, index 4. This is `:path: /`.
- `0x87`: `10000111`. Indexed header, index 7. This is `:scheme: https`.
- `0x41`: `01000001`. Starts with `01`, so it's a "Literal Header Field with Incremental Indexing". The remaining 6 bits are the index for the name, which is 1 (`:authority`). The value will follow as a literal string.
- `byte(len(authority))`: The length of the value "kmcd.dev", which is 8.
- `[]byte(authority)`: The string "kmcd.dev" itself.

We have now manually encoded a `HEADERS` payload!

### The Client's Structure

Here is a high-level overview of our client's logic so far:

1.  **Connect & Handshake**: Establish a TCP connection and perform the TLS handshake, using ALPN to negotiate "h2".
2.  **Send Preface**: Send the magic `PRI * ...` connection preface.
3.  **Exchange Settings**: Send our empty `SETTINGS` frame, wait for the server's `SETTINGS` frame, and then send an `ACK`.
4.  **Send Request**: Construct and send the `HEADERS` frame for `GET /` with our manually encoded HPACK payload. We set the `END_STREAM` flag to indicate this is our entire request.
5.  **Read Response**: Loop and read frames from the server.
    - If it's a `HEADERS` frame, use our `HPACKDecoder` to parse and print the response headers.
    - If it's a `DATA` frame, print the content.
    - If we see a frame with the `END_STREAM` flag, the response is complete, and we exit.

This gives us a working end-to-end client that makes a real HTTP/2 request and prints the response.

### Putting It All Together: The Final Client

Here is the full `client.go` script.

{{% render-code file="go/client.go" language="go" %}}
{{< aside >}}
See the full client implementation: {{< github-link file="go/client.go" >}}.
{{</ aside >}}

Running this client produces a full HTTP/2 interaction. We send our request and get back headers and data from the server, all parsed by our own code.

### What's Next?

We've made a huge leap. We can now make requests and parse simple responses. However, our HPACK decoder is incomplete. It only understands indexed header fields. What happens when the server sends a header that isn't in our static table, like a unique `date` or `etag`? Our client will return an error.

In the next part, we will implement the decoding of **Literal Header Fields**. This will involve parsing string lengths and reading string values. We'll also start to properly manage the dynamic table, bringing us even closer to a complete HTTP/2 client.
