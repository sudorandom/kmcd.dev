---
categories: ["article"]
tags: ["go", "http2", "protocols", "networking"]
date: "2026-03-04T10:00:00Z"
description: "More HPACK and using http.Request and http.Response"
cover: "cover.svg"
images: ["/posts/http2-from-scratch-part-4/cover.svg"]
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "HTTP/2 From Scratch: Part 4"
slug: "http2-from-scratch-part-4"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/http2-from-scratch-part-4/
draft: true
series: ["HTTP From Scratch"]
---

TODO: Intro here. Outline the goals for this article.

In Part 3, we only implemented the **Static Table**, a read-only list of 61 common headers. But the real power of HTTP/2 comes from the **Dynamic Table**.

Think of the Dynamic Table as a rolling cache that exists for the duration of the connection.

1. **Client** sends a custom header `x-user-id: 123`.
2. **Server** sees this is a new literal. It uses the value and adds it to the Dynamic Table at Index 62.
3. **Client** sends a second request. Instead of sending the string `x-user-id: 123` again, it just sends the byte for **Index 62**.

To make this work, we need to handle a few new concepts in our `hpack.go`.

#### 1. Huffman Decoding

HTTP/2 allows headers to be compressed using Huffman coding. This saves bytes by using shorter bit sequences for common characters (like `e` or `a`) and longer ones for rare characters.

Implementing a full Huffman tree from scratch is a fun exercise, but it is out of scope for this series for the same reason we are not implementing TLS from scratch: it is a complex topic in itself. For those interested in learning more, here are some great resources:
*   [Huffman Coding Explained](https://www.youtube.com/watch?v=0kNXhUcr3qQ) (YouTube video)
*   [A detailed article on Huffman Coding](https://www.geeksforgeeks.org/huffman-coding-greedy-algo-3/)

To keep our focus on HTTP/2, we will borrow the standard library's internal logic for this specific math problem.

```go
// We import the official helper just for the string decoding math
import "golang.org/x/net/http2/hpack"

// ... inside our decodeString method
if huffman {
    return hpack.HuffmanDecodeToString(data)
}
```

#### 2. The Decoder Loop

Our previous `Decode` method was not complete. We need to expand it to handle the four main types of header representations defined in [RFC 7541](https://datatracker.ietf.org/doc/html/rfc7541).

TODO: enumerate header representations

To keep the code readable, we’ve defined constants for the bitmasks (like `0x80` for indexed fields). Here is the updated loop:

```go
func (h *HPACKDecoder) Decode(payload []byte) ([]HeaderField, error) {
    var headers []HeaderField
    r := bytes.NewReader(payload)
    
    for r.Len() > 0 {
        b, _ := r.ReadByte()
        
        // 1. Indexed Header Field (starts with 1xxxxxxx)
        if b&maskIndexed == patternIndexed { 
            // ... fetch from Static or Dynamic table ...
        
        // 2. Literal with Incremental Indexing (starts with 01xxxxxx)
        //    (Read the header, then ADD it to the Dynamic Table)
        } else if b&maskLiteralIncremental == patternLiteralIncremental { 
            header, _ := h.decodeLiteralHeader(r, 6, true)
            headers = append(headers, header)

        // 3. Literal without Indexing (starts with 0000xxxx or 0001xxxx)
        //    (Read the header, do NOT add to Dynamic Table)
        } else if b&maskLiteral == patternLiteral { 
             header, _ := h.decodeLiteralHeader(r, 4, false)
             headers = append(headers, header)

        // 4. Dynamic Table Size Update (starts with 001xxxxx)
        } else if b&maskDynamicTableSize == patternDynamicTableSize { 
            // ... update max size ...
        }
    }
    return headers, nil
}
```

TODO: Add note about dynamically adjusting the table size

You'll notice we refactored the heavy lifting into a helper called `decodeLiteralHeader`. This function handles reading the name (which might be an index or a string) and the value (which is always a string), and optionally inserting it into our `dynamicTable`.

{{< aside >}}
See the full updated HPACK logic: {{< github-link file="go/hpack.go" >}}.
{{</ aside >}}

### Writing an Encoder

In the last part, we manually crafted our request bytes (`0x82`, `0x84`...). That was great for learning, but terrible for usability. We need an `HPACKEncoder`.

Our encoder will use a "naive" but compliant strategy to compress headers:

1. **Perfect Match:** Is the full header (`:method: GET`) already in the Static Table? If yes, send the 1-byte index.
2. **Name Match:** Is just the *name* (`:authority`) in the Static Table? If yes, send the index for the name, but write the value as a string literal.
3. **No Match:** Send both the name and value as string literals.

This simple logic covers 90% of use cases without needing complex state tracking on the client side.

```go
func (e *HPACKEncoder) Encode(headers []HeaderField) []byte {
    var buf bytes.Buffer
    for _, hf := range headers {
        // 1. Try to find a full match (Name + Value)
        if index, ok := staticTableMap[hf]; ok {
            encodeInt(&buf, index, 7, 0x80)
            continue
        }

        // 2. Try to find a Name match
        if index, ok := staticTableNameMap[hf.Name]; ok {
            encodeInt(&buf, index, 6, 0x40) // Literal with Incremental Indexing
            encodeString(&buf, hf.Value)
            e.dynamicTable.Add(hf) // We must track this too!
            continue
        }

        // 3. Send as a full literal
        encodeInt(&buf, 0, 6, 0x40)
        encodeString(&buf, hf.Name)
        encodeString(&buf, hf.Value)
        e.dynamicTable.Add(hf)
    }
    return buf.Bytes()
}
```

### Building a Real Client

Now for the satisfying part. We are going to take all that raw socket code from `main.go` and wrap it in a clean struct that mimics the standard library.

TODO: Talk about and reference http.Client, http.Request, http.Response

We define a `Client` that holds our HPACK context. The connection itself will be managed by the `Do` method.

```go
type Client struct {
	addr     string
	hpackDec *HPACKDecoder
	hpackEnc *HPACKEncoder
}

func NewClient(addr string) *Client {
	return &Client{
		addr:     addr,
		hpackDec: NewHPACKDecoder(4096),
		hpackEnc: NewHPACKEncoder(4096),
	}
}
```

And finally, the `Do` method. This is where we translate a standard Go `http.Request` into HTTP/2 frames. It now handles the entire connection lifecycle for a single request.

```go
func (c *Client) Do(req *http.Request) (*http.Response, error) {
    // 1. Connect and perform handshake
	conn, err := tls.Dial("tcp", c.addr, config)
    // ... handle preface and settings frames ...
    
    // 2. Convert http.Request headers to HPACK HeaderFields
    //    Don't forget the pseudo-headers!
    headers := []HeaderField{
        {Name: ":method", Value: req.Method},
        // ... and so on
    }
    
    // ... append req.Header ...

    // 3. Encode headers to bytes
    requestPayload := c.hpackEnc.Encode(headers)

    // 4. Send HEADERS frame (Stream ID 1)
    // ... writeFrame ...

    // 5. Read Loop: Wait for response frames
    // ... readFrame ...
}
```

### The Result

With this refactor, our `main.go` transforms from a mess of hex dumps into clean, idiomatic Go:

```go
func main() {
    // 1. Create a client
	client := NewClient("kmcd.dev:443")

    // 2. Create a standard Request
    req, _ := http.NewRequest("GET", "https://kmcd.dev/", nil)

    // 3. Execute!
    resp, err := client.Do(req)
    if err != nil {
        log.Fatal(err)
    }
    defer resp.Body.Close()

    // 4. Enjoy the output
    fmt.Printf("Protocol: %s\n", resp.Proto)
    fmt.Printf("Status: %s\n", resp.Status)

    body, _ := io.ReadAll(resp.Body)
    fmt.Printf("Body Length: %d bytes\n", len(body))
}
```

When we run this, we no longer fail to parse all of the headers. We get a full 200 OK response, fully parsed, with the body ready to read. We have successfully built a working HTTP/2 client from scratch.

### Limitations

While we have a working client, it is strictly a "happy path" implementation. If you compare this to `net/http`, we are missing some very important features:

* **Concurrency:** Our client is synchronous. It sends a request and waits. Real HTTP/2 uses multiplexing to send many requests at once over a single connection. It also splits up "sending" and "receiving" into two goroutines and maintains synchronization primitives to coordinate between the two.
* **Flow Control:** We are ignoring `WINDOW_UPDATE` frames. If we tried to download a large file (larger than the default 65KB window), our connection would stall.
* **Connection Re-use:** We create a new TCP connection for every `Do` call. A real client maintains a pool of connections.

However, we have achieved our goal: we de-mystified the binary protocol, decoded HPACK, and successfully talked to a real server using code we wrote ourselves.

That wraps up HTTP/2! I hope this gave you a deeper appreciation for what happens every time you type a URL into your browser. The natural next step here is to do the same exercise with QUIC and HTTP/3... although I don't promise do that that soon.
