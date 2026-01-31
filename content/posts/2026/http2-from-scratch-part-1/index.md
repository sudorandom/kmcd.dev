---
categories: ["article"]
tags: ["go", "http2", "protocols", "networking"]
date: "2026-02-11T10:00:00Z"
description: "Re-building the web in Go to learn more about it"
cover: "cover.svg"
images: ["/posts/http2-from-scratch-part-1/cover.svg"]
featuredalt: ""
featuredpath: "date"
linktitle: ""
title: "HTTP/2 From Scratch: Part 1"
slug: "http2-from-scratch-part-1"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/http2-from-scratch-part-1/
draft: false
series: ["HTTP From Scratch"]
---

If you have ever opened a terminal and manually typed an HTTP/1.1 request, you know there is a certain beauty in its simplicity. You send a few lines of plain text and the server responds with more text. It is human-readable and easy to debug, but it is also remarkably inefficient for the modern web. If you haven't done that, I'm hoping I can get you to use telnet this way for the first time today.

As applications have grown more complex and require many disparate streams of data, we seem to have reached the limit to what a text-based protocol can do. In this series, we are going to stop using the high-level abstractions provided by the Go standard library (mostly) and build an HTTP/2 implementation from the ground up. This is the way that I've learned best, so maybe you can join me on this journey.

### The Simplicity of the Past

To understand why we need HTTP/2, we first have to look at how simple things used to be. You can still see the inner workings of the old web by using `telnet` or `nc` (netcat) to talk to a server. If you connect to port 80 and manually type a request, the interaction looks like this:

```http
$ telnet kmcd.dev 80
Connected to kmcd.dev.
Escape character is '^]'.

GET / HTTP/1.1
Host: kmcd.dev

```

After hitting `enter` twice, the server sends back a text response (Some of the CloudFlare headers removed for brevity):

```http
HTTP/1.1 301 Moved Permanently
Date: Wed, 28 Jan 2026 14:17:33 GMT
Content-Length: 0
Connection: keep-alive
Location: https://kmcd.dev/
Server: cloudflare
```

This response is a [301 status code](https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Status/301), telling you to use the HTTPS version of the website, because plain-text HTTP is for heathens. But regardless, this is extremely insightful and cuts out many of the complexities that tooling can add. You can read every byte, understand every header, and even type the requests by hand. However, this transparency comes at a cost.

For completeness, here is a similar setup using HTTPS via the openssl CLI client:
```http
$ openssl s_client -connect kmcd.dev:443 -servername kmcd.dev
CONNECTED(00000006)
depth=2 C = US, O = Google Trust Services LLC, CN = GTS Root R4
verify return:1
depth=1 C = US, O = Google Trust Services, CN = WE1
verify return:1
depth=0 CN = kmcd.dev
verify return:1
write W BLOCK
---
Certificate chain
 0 s:/CN=kmcd.dev
   i:/C=US/O=Google Trust Services/CN=WE1
 1 s:/C=US/O=Google Trust Services/CN=WE1
   i:/C=US/O=Google Trust Services LLC/CN=GTS Root R4
 2 s:/C=US/O=Google Trust Services LLC/CN=GTS Root R4
   i:/C=BE/O=GlobalSign nv-sa/OU=Root CA/CN=GlobalSign Root CA
---
Server certificate
-----BEGIN CERTIFICATE-----
MIIDkDCCAzWgAwIBAgIRANuMVwoYX/O/E75qT9V5T9AwCgYIKoZIzj0EAwIwOzEL
MAkGA1UEBhMCVVMxHjAcBgNVBAoTFUdvb2dsZSBUcnVzdCBTZXJ2aWNlczEMMAoG
A1UEAxMDV0UxMB4XDTI2MDEwMTA1MjA0OVoXDTI2MDQwMTA2MjA0M1owEzERMA8G
A1UEAxMIa21jZC5kZXYwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAATxkKGuge8n
FcjibwoBq1QKBeq/KDwkorBy7MUuyYMcvhfdB9QGMalqF8wDtkruPStPe6rMAUjZ
NAoDSKmJwrdco4ICQDCCAjwwDgYDVR0PAQH/BAQDAgeAMBMGA1UdJQQMMAoGCCsG
AQUFBwMBMAwGA1UdEwEB/wQCMAAwHQYDVR0OBBYEFKVQVjLBxQ3apOR8KM1a/BjG
BNyzMB8GA1UdIwQYMBaAFJB3kjVnxP+ozKnme9mAeXvMk/k4MF4GCCsGAQUFBwEB
BFIwUDAnBggrBgEFBQcwAYYbaHR0cDovL28ucGtpLmdvb2cvcy93ZTEvMjR3MCUG
CCsGAQUFBzAChhlodHRwOi8vaS5wa2kuZ29vZy93ZTEuY3J0MBMGA1UdEQQMMAqC
CGttY2QuZGV2MBMGA1UdIAQMMAowCAYGZ4EMAQIBMDYGA1UdHwQvMC0wK6ApoCeG
JWh0dHA6Ly9jLnBraS5nb29nL3dlMS9mLUxxQnNPOTBnWS5jcmwwggEDBgorBgEE
AdZ5AgQCBIH0BIHxAO8AdQDRbqmlaAd+ZjWgPzel3bwDpTxBEhTUiBj16TGzI8uV
BAAAAZt4N1MJAAAEAwBGMEQCIFpnx0F4+HFZiAAZp/S1OXTqUGE2XXEJreljtBiX
52ezAiA8IqlGlvYzT9to8aWkfCmvd5/zxpktLVxjPAZhTtvO/QB2AA5XlLzzrqk+
MxssmQez95Dfm8I9cTIl3SGpJaxhxU4hAAABm3g3Ui8AAAQDAEcwRQIgYpHRsDzn
0EyvJeasmiFJHnVAgNjO9nVLhHUkgMMluQ0CIQCLxOjYLJUGwBMAF4RG2L7J+P/A
4g2ojvjXSOqUhIpJmTAKBggqhkjOPQQDAgNJADBGAiEAgfuMNfYBAbPoYqIJ7TyY
y1uNp7kzpQShTsBpjgQrqTsCIQC22/6iBI+qiTSgWIrF7tKFil8+XyoMdmf8CTeI
GBAV/w==
-----END CERTIFICATE-----
subject=/CN=kmcd.dev
issuer=/C=US/O=Google Trust Services/CN=WE1
---
No client certificate CA names sent
Server Temp Key: ECDH, X25519, 253 bits
---
SSL handshake has read 2789 bytes and written 368 bytes
---
New, TLSv1/SSLv3, Cipher is AEAD-CHACHA20-POLY1305-SHA256
Server public key is 256 bit
Secure Renegotiation IS NOT supported
Compression: NONE
Expansion: NONE
No ALPN negotiated
SSL-Session:
    Protocol  : TLSv1.3
    Cipher    : AEAD-CHACHA20-POLY1305-SHA256
    Session-ID:
    Session-ID-ctx:
    Master-Key:
    Start Time: 1769674847
    Timeout   : 7200 (sec)
    Verify return code: 0 (ok)
---
GET / HTTP/1.1
Host: kmcd.dev

HTTP/1.1 200 OK
Date: Thu, 29 Jan 2026 08:20:53 GMT
Content-Type: text/html; charset=utf-8
Transfer-Encoding: chunked
Connection: keep-alive
Access-Control-Allow-Origin: *
Cache-Control: max-age=31536000, public
cf-cache-status: DYNAMIC
Link: <https://fonts.googleapis.com>; rel="preconnect"
referrer-policy: no-referrer
x-content-type-options: nosniff
x-frame-options: deny
x-xss-protection: 1; mode=block
Vary: accept-encoding
Server: cloudflare
alt-svc: h3=":443"; ma=86400

3d8d
<!doctype html><html lang=en><head><meta name=generator content="Hugo 0.152.2">...[snip]...</head></html>
0

```

The response has been trimmed for brevity, but this shows that you can use a TLS-wrapped socket to access HTTPS websites using a plain-text interface similar to telnet. This shows how the "S" in "HTTPS" is simply HTTP wrapped with TLS.

### Head-of-line blocking

The primary issue with HTTP/1.1 is **head-of-line blocking**. Even if you have a high-speed fiber connection, a single TCP stream can only process one request at a time. If your browser needs a tiny CSS file but it is stuck behind a massive high-resolution image, that CSS file simply waits. In the early days of the web, when pages were just a few kilobytes of text and a couple of images, this was a minor inconvenience. Today, the average webpage loads hundreds of individual assets.

Web developers and browser makers are clever individuals. They fixed this by opening multiple TCP connections to the same server. Browsers usually allow up to six parallel connections per domain name. If developers need more than that, they would host assets on different domain names using a technique called domain sharding. This set of techniques did make loading web assets faster, but this is a heavy-handed solution that comes as a cost. Each new connection requires a full TCP handshake and a TLS negotiation which means needing more powerful servers and clients to deal with this extra crypto negotiation.

Beyond the connection limits, HTTP/1.1 is incredibly verbose. Every single request sends the same headers over and over again. Your user agent and cookie headers are almost identical for every image on a page, yet the protocol forces you to upload that redundant text hundreds of times. On mobile networks with limited upload bandwidth, this overhead becomes a measurable performance tax.

### Persistent Connections and Pipelining

Before we dive into the binary world of HTTP/2, it’s worth noting that HTTP/1.1 didn’t just sit idly by while performance suffered. It introduced two key concepts to mitigate the overhead of opening new connections: Persistent Connections (Keep-Alive) and Pipelining.

In the original HTTP/1.0 days, every single request required a brand-new TCP handshake. HTTP/1.1 changed the default behavior to "Keep-Alive," allowing a single TCP connection to stay open for multiple subsequent requests. This saved the "tax" of the initial handshake for every image or script.

Then came Pipelining. The idea was brilliant on paper: instead of waiting for a response before sending the next request, a client could fire off three or four requests in a row. While you could send requests in a batch, the server was strictly required to send the responses back in the exact same order they were received. If the first request in the "pipe" was a slow database query and the second was a tiny static file, the static file was still stuck waiting for the database; the classic "Head-of-Line Blocking" problem. Because of buggy implementations in middleware and browsers, pipelining never saw widespread adoption and is disabled by default in almost every modern browser for `HTTP/1`. It was a valiant attempt at concurrency that ultimately proved the need for a more radical shift in how we frame data.

### Entering the Binary World

HTTP/2 solves these problems by moving away from text entirely. As defined in [RFC 9113, Section 4](https://www.rfc-editor.org/rfc/rfc9113.html#name-http-frames), it introduces a binary framing layer that allows us to interleave multiple requests and responses over a single connection. This technique is called multiplexing. By breaking every message into small, independent frames, a slow request no longer blocks the ones behind it. The protocol can send a piece of an image, then a piece of a script, then another piece of the image, all within the same stream.

In this binary world, we stop thinking about lines of text and start thinking about frames. Every piece of data is wrapped in a header that tells the receiver exactly what it is, how big it is, and which request it belongs to. This makes parsing significantly faster for machines. In HTTP/1.1, the server has to read the text and look for newlines to know where a header ends. In HTTP/2, the server knows exactly how many bytes to read next.

### Negotiating the Connection

Since the web is built on backward compatibility, a client needs a way to tell a server it wants to use HTTP/2 without breaking things for older servers. We do this using ALPN (Application-Layer Protocol Negotiation) during the TLS handshake. This allows the client and server to agree on a protocol before they ever start sending application data.

Once the encrypted tunnel is established, the client must send a connection preface. This is a 24-byte magic string that serves as a final confirmation. The string looks like this, as documented in [Section 3.5 in RFC-9113](https://www.rfc-editor.org/rfc/rfc9113.html#name-http-2-connection-preface):

```http
PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n
```

If the server receives this string successfully, it will speak HTTP/2 from then on. It will respond with its own settings. If it does not, it will likely drop the connection. This prevents a modern client from sending binary data to an old server that is still expecting plain text.

### The Implementation in Go

The following script is the starting point for our project. It uses the `crypto/tls` package to handle the encryption but stops right at the moment the HTTP/2 state machine should take over. We define the mandatory preface and configure our TLS dialer to negotiate for the "h2" protocol.

{{% render-code file="go/client.go" language="go" %}}
{{< aside >}}
See the full source at Github: {{< github-link file="go/client.go" >}}.
{{</ aside >}}

This is actually quite amazing. A lot of hard work is being done to give us a TLS-wrapped connection and we've also used TLS to negotiate the HTTP/2 protocol for us. TLS is a complex protocol in its own right, so we’ll rely on Go’s `crypto/tls` package and focus entirely on `HTTP/2` from this point forward.

### What Happens Next

At this point, the connection is technically alive, but it is silent. The server is now expecting us to handle binary frames. If you were to run this code and try to read from the connection, you would see a stream of bytes representing the server's initial configuration.

In the next post, we leave the comfort of strings behind and dive into the `encoding/binary` package. We’ll build a parser for the mandatory 9-byte frame header, learn how to mask bits, and start interpreting the hex constants that define the language of the modern web.
