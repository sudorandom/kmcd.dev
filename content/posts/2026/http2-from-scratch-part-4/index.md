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
series: ["HTTP from Scratch"]
---

Part 3 ended on a high note: we got a real `200 OK` from a server. But the code felt brittle. We were crafting requests by hand-picking hex codes from a spec sheet (`0x82` for `GET`, `0x84` for `/`) and our "client" couldn't have been less practical. It was a proof-of-concept held together with duct tape and hope.

Today, we're rebuilding it properly. We'll finish our HPACK implementation and, most importantly, refactor the raw socket logic into a clean Go client that speaks the language of the standard library: `http.Request` and `http.Response`.

Previously, we only implemented the **Static Table**, a read-only list of 61 common headers. But the real power of HTTP/2 comes from the **Dynamic Table**.

The Dynamic Table is essentially a short-term memory, or cache, that both the client and server build together for the duration of a single TCP connection.

1.  **Client** sends the header `x-user-id: 123` using the "Literal Header Field with Incremental Indexing" representation. This tells the server to use the value *and* add it to its Dynamic Table.
2.  **Server** acknowledges and adds it to its table at Index 62.
3.  **Client** sends a second request. Instead of sending the string `x-user-id: 123` again, it just sends the byte for **Index 62**.

To make this work, we need to handle a few new concepts in our `hpack.go`.

#### 1. Huffman Decoding

Next up is Huffman coding. HTTP/2 uses it to shrink header strings. The specification includes a canonical Huffman table for this, but it's massive. Implementing it would mean embedding a 400-line static lookup table and writing a tree-walker to encode/decode it. I briefly considered it, then imagined debugging a single bit-flip error in that tree.

That's why I'm using `golang.org/x/net/http2/hpack` for this one part. It's the perfect example of when building "from scratch" becomes counter-productive. We can pull in a battle-tested implementation for the tedious part and keep our focus on the protocol's state and frame logic.

```go
// We import the official helper just for the string decoding math
import "golang.org/x/net/http2/hpack"

// ... inside our decodeString method
if huffman {
    return hpack.HuffmanDecodeToString(data)
}
```

#### 2. The Decoder Loop

Our previous `Decode` method was incomplete. We need to handle four specific bit-patterns defined in [RFC 7541](https://datatracker.ietf.org/doc/html/rfc7541).

Aside from the standard **Indexed** fields (where we just look up a value), we now have to handle **Literals**. Some literals request **Incremental Indexing** (meaning the receiver saves them to its dynamic table for later), while others are **Non-Indexed** (one-offs for things like tokens that shouldn't be cached). Finally, we might see a **Table Size Update**, which tells us if the server resized its compression window.

Here is the updated loop using bitmasks to identify the type:

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

{{< aside >}}
See the full updated HPACK logic: {{< github-link file="go/hpack.go" >}}.
{{</ aside >}}

### Writing an Encoder

In the last part, we manually crafted our request bytes (`0x82`, `0x84`...). That was great for learning, but terrible for usability. We need an `HPACKEncoder`.

Our encoder will use a "naive" but compliant strategy to compress headers:

1.  **Perfect Match:** Is the full header (`:method: GET`) already in the Static Table? If yes, send the 1-byte index.
2.  **Name Match:** Is just the *name* (`:authority`) in the Static Table? If yes, send the index for the name, but write the value as a string literal and instruct the server to index it.
3.  **No Match:** Send both the name and value as string literals and instruct the server to index them.

This simple logic covers 90% of use cases without needing complex state tracking on the client side.

```go
func (e *HPACKEncoder) Encode(headers []HeaderField) []byte {
    var buf bytes.Buffer
    for _, hf := range headers {
        // 1. Try to find a full match (Name + Value)
        if index, ok := staticTableMap[hf]; ok {
            encodeInt(&buf, index, 7, patternIndexed)
            continue
        }

        // 2. Try to find a Name match
        if index, ok := staticTableNameMap[hf.Name]; ok {
            encodeInt(&buf, index, 6, patternLiteralIncremental) // Literal with Incremental Indexing
            encodeString(&buf, hf.Value)
            e.dynamicTable.Add(hf) // We must track this too!
            continue
        }

        // 3. Send as a full literal
        encodeInt(&buf, 0, 6, patternLiteralIncremental)
        encodeString(&buf, hf.Name)
        encodeString(&buf, hf.Value)
        e.dynamicTable.Add(hf)
    }
    return buf.Bytes()
}
```

### Building a Real Client

Now for the satisfying part. We are going to take all that raw socket code from `main.go` and wrap it in a clean struct that mimics the standard library. Our goal here is to integrate seamlessly with Go's `net/http` package, allowing us to leverage familiar types.

```go
// client.go

type Client struct {
    Timeout time.Duration
}

func NewClient() *Client {
    return &Client{
        Timeout: 30 * time.Second,
    }
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
    // ... Connection and Handshake logic ...

    // 1. Convert http.Request to HPACK HeaderFields
    authority := req.URL.Host
    if authority == "" {
        authority = req.Host // Fallback for robustness
    }
    headers := []HeaderField{
        {Name: ":method", Value: req.Method},
        {Name: ":scheme", Value: req.URL.Scheme}, 
        {Name: ":authority", Value: authority},
        {Name: ":path", Value: req.URL.Path},
    }
    // ... append req.Header ...

    // 2. Encode and send the HEADERS frame
    // ...

    // 3. Read Loop and a Subtle Bug
    // ...
    
    return httpResp, nil
}
```

After sending our request, we enter a loop to read the server's response. This is where I hit a classic Go concurrency bug. My first implementation had a `defer conn.Close()` right after dialing the connection.

```go
// The WRONG way
conn, err := tls.Dial(...)
if err != nil { ... }
defer conn.Close() // Problem!

// ... send request ...

return &http.Response{ Body: io.NopCloser(bytes.NewReader(bodyBytes)) }, nil
```

The problem? `defer` executes when the function (`Do`) returns. But the user of our client needs to read the response `Body` *after* `Do` has returned. The connection was being closed before they had a chance to read it!

The fix is to remove the `defer` and instead wrap the connection itself in a custom `io.ReadCloser` that we assign to the `http.Response.Body`. When the user calls `resp.Body.Close()`, it's now our responsibility to close the underlying network connection. This is the standard pattern used by Go's own `net/http` client.

Here's the read loop that replaces the `...` above. I spent a good hour debugging why it would sometimes hang, only to realize I was mishandling the `END_STREAM` flag.

```go
// The Read Loop
var respHeaders []HeaderField
var respBody []byte
for {
    frame, err := ReadFrame(conn)
    if err != nil {
        return nil, fmt.Errorf("connection closed: %w", err)
    }

    switch frame.Header.Type {
    case FrameData:
        respBody = append(respBody, frame.Payload...)
    case FrameHeaders:
        headers, err := hpackDec.Decode(frame.Payload)
        if err != nil {
            return nil, fmt.Errorf("hpack error: %w", err)
        }
        respHeaders = append(respHeaders, headers...)
    case FrameGoAway:
        // Handle server telling us to go away
    case FrameWindowUpdate:
        // Handle flow control
    }

    // The stream is finished when we receive a frame with the END_STREAM flag.
    // This can be on a HEADERS frame or the final DATA frame.
    isEndFrame := frame.Header.Type == FrameData || frame.Header.Type == FrameHeaders
    if isEndFrame && (frame.Header.Flags&FlagEndStream != 0) {
        break
    }
}

// ... build http.Response, then assign our custom Body ...
httpResp.Body = &responseBody{bytes.NewReader(respBody), conn}
return httpResp, nil

// And the helper struct that makes this possible:
type responseBody struct {
    *bytes.Reader
    conn io.Closer
}

func (rb *responseBody) Close() error {
    return rb.conn.Close()
}
```

### The Result

With this refactor, our `main.go` transforms from a mess of magical hex values into clean, idiomatic Go:

```go
func main() {
    client := NewClient()
    req, _ := http.NewRequest("GET", "https://kmcd.dev/", nil)

    resp, err := client.Do(req)
    if err != nil {
        log.Fatal(err)
    }
    // This now correctly closes our underlying TCP connection.
    defer resp.Body.Close()

    fmt.Printf("Status: %s\n", resp.Status)
    body, _ := io.ReadAll(resp.Body)
    fmt.Printf("Body Length: %d bytes\n", len(body))
}
```

### Limitations

While we have a working client, it is strictly a "happy path" implementation:

* **Concurrency:** Our client is synchronous. It sends a request and waits. Real HTTP/2 uses multiplexing to send many requests at once over a single connection.
* **Flow Control:** We are ignoring `WINDOW_UPDATE` frames. If we tried to download a large file, our connection would stall once the window fills up.
* **Connection Re-use:** We create a new TCP connection for every `Do` call. Because the HPACK Dynamic Table is tied to the connection, we aren't actually gaining any compression benefits across multiple requests.
* **Many, Many Other Features:** There is a lot of small details that this toy client completely glosses over. For example, trailer support. This is funny, because I actually added trailer support for [quic-go](https://github.com/quic-go/quic-go): [#4581](https://github.com/quic-go/quic-go/pull/4581), [#4630](https://github.com/quic-go/quic-go/pull/4630). You can read up more about this when writing my [gRPC over HTTP/3](/posts/grpc-over-http3/) series.

Getting this far, I have a renewed and massive respect for the `net/http` maintainers. We started this series with hex dumps and now have a client struct that respects `io.Closer`. The "happy path" alone is a journey through specifications, bit-masking, and subtle concurrency bugs. Handling real-world network conditions, multiplexing, and flow control is a monumental task that makes you appreciate the standard library on a new level.

For now, I'm happy with this victory. We've built a real, working HTTP/2 client.
