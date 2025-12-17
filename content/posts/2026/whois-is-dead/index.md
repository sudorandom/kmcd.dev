---
categories: ["article"]
tags: ["go", "networking", "whois", "protocol"]
date: "2026-01-13T10:00:00Z"
description: "WHOIS is dead. To memorialize this piece of internet history, let's build a tiny implementation from scratch."
cover: "cover.svg"
images: ["/posts/whois-from-scratch/cover.svg"]
featuredalt: "A diagram showing a client and server communicating via WHOIS"
featuredpath: "date"
linktitle: ""
title: "WHOIS is dead, long live RDAP"
slug: "whois-from-scratch"
type: "posts"
devtoSkip: true
canonical_url: https://kmcd.dev/posts/whois-from-scratch/
draft: true
---

The `whois` protocol is dead. For decades, it was a fundamental tool for network reconnaissance, but its time has passed. The protocol was officially sunset for all generic top-level domains in early 2025, replaced by a new kid in town: [RDAP](https://about.rdap.org/).

{{< image src="whois-dead.png" width="500px" class="center" >}}

So why talk about WHOIS now? To pay our respects. Because the WHOIS protocol is so simple, it makes a perfect case study for basic network programming and a window into an earlier era of the internet. To help memorialize this piece of internet history, we will build a tiny implementation from scratch and understand why its death was necessary in the process.

#### What is WHOIS?

WHOIS is the "caller ID" of the internet. It answers the fundamental question: "Who owns this?"

When you run a WHOIS query for `example.com`, you are asking for the administrative metadata behind that address. A typical response provides the name of the Registrar that sold the domain (e.g., Namecheap, GoDaddy), the Name Servers responsible for translating the domain into an IP, and critical Dates. Specifically, you receive when the domain was created and when it will expire. Historically, this also included the full contact details of the owner, though privacy laws like GDPR have largely pushed that data behind redaction services.

### Who provides this data?

WHOIS data is a requirement enforced by ICANN (Internet Corporation for Assigned Names and Numbers). ICANN sets the rules, Registries (like Verisign for .com) manage the master lists for their TLDs, and Registrars (like Google Domains or Namecheap) sell the names to you. Domain registrars are contractually obligated to maintain this registration data and make it available to the public.

### Why do we need it?

While often used by developers to check if a cool side project name is taken, WHOIS is critical infrastructure for maintaining the internet's health.

However, its utility has changed. Since the introduction of GDPR and widespread privacy redaction services, you rarely see the actual owner's name or email address anymore. The "Who" in WHOIS is often hidden. So why is it still useful?

Even with privacy redaction, ICANN requires a valid "Abuse Contact Email" and phone number to be visible. Network operators rely on this field to alert registrars when a domain is hosting malware or spam.

Researchers have adapted to the privacy changes. They no longer look for a name; they look for patterns. If 500 suspicious domains were all registered at the exact same second, using the same obscure Name Server, and the same Registrar, they are likely part of the same botnet. We don't need to know who they are to know they are connected.

Investigative journalists use historical WHOIS data to map state-sponsored disinformation campaigns. For modern investigations, RDAP introduces "tiered access," theoretically allowing vetted professionals to request unredacted data for legitimate purposes, though this process is still maturing. Also, a new initiative called [RDRS](https://www.icann.org/rdrs-en) aims to standardize access to nonpublic registration data.

### How WHOIS Works

The WHOIS protocol, defined in [RFC 3912](https://www.rfc-editor.org/rfc/rfc3912), is a simple exchange over a TCP connection on port 43.

1. The client opens a TCP socket to a WHOIS server on port 43.
2. The client sends the query: a single line of text like `example.com`, terminated by a carriage return and line feed (`<CR><LF>`).
3. The server sends back a stream of plain text containing the registration data.
4. The server closes the connection.

There are no headers, no authentication, and no complex data formats (more on that later). It is quite literally one of the simplest protocols imaginable. This simplicity makes it a good candidate for a small project to demonstrate basic networking concepts.

### Building a WHOIS Server

Let's put that theory into code. Because the protocol is so trivial, we can implement a functional server in Go using a few lines of code and the Go standard library. We can then verify that it works using the tools already installed on your machine, like telnet or the whois command itself.

#### WHOIS Server Implementation

We'll add a `records` map to hold fake domain data and implement a `handleConnection` function to process queries and send back the corresponding record.

{{< details-md summary="whois-server/main.go" github_file="go/whois-server/main.go" >}}
{{% render-code file="go/whois-server/main.go" language="go" %}}
{{< /details-md >}}

To run the server, execute:
```bash
go run ./whois-server
```

With the server running, you can now test it with `telnet` and `whois`:

```bash
telnet localhost 43
# Trying ::1...
# Connected to localhost.
# Escape character is '^]'.
# example.com
# Domain Name: example.com
# Registrar: My Go Server
# Creation Date: 2025-12-15T00:00:00Z
# Connection closed by foreign host.
```

```bash
whois -h localhost google.com
# Domain Name: google.com
# Registrar: My Go Server
# Creation Date: 2025-12-15T00:00:00Z
```

### Real-World Complications

Our server works for the domains stored in its local `records` map, but the real `WHOIS` system is a distributed, federated system of registries and registrars, not a single database.

This leads to concepts like **"thin" and "thick" lookups**. A "thick" registry (like `.org`) holds all the data, and one query is enough. A "thin" registry (like `.com`) only knows which registrar manages a domain (e.g., GoDaddy, Namecheap). A `whois` client querying a "thin" registry gets a referral and must make a second query to the correct registrar's WHOIS server to get the full details.

This system is brittle, relying on parsing unstructured text to find the referral server. Classic `whois` clients, such as the [`rfc1036/whois`](https://github.com/rfc1036/whois/blob/next/whois.c), handle this by scanning each line of text for known referral markers using functions like `find_referral_server_iana`. This approach works, but it is fragile because every registry formats output differently. The brittleness of parsing free-form text was a key driver to replace it with a modern protocol that uses structured data like JSON.

### RDAP: The Modern Successor

The push to replace it began back in 2013, when an ICANN Expert Working Group recommended that the WHOIS protocol should be tossed out. They proposed a system that would keep information secret from most users, disclosing data only for specific "permissible purposes" like legal actions or trademark enforcement. Notably, **journalism was excluded** from this list, despite WHOIS historically being a key tool for investigative reporting.

After years of debate and voting, the transition became official. On January 28, 2025, WHOIS was officially sunset for generic Top-Level Domains (gTLDs). Registries are no longer required to support it, and the industry has shifted its focus to RDAP (Registration Data Access Protocol).

RDAP performs the same function as WHOIS but it uses HTTPS and returns structured JSON instead of raw text.

| Feature    | WHOIS                               | RDAP                                    |
|------------|-------------------------------------|-----------------------------------------|
| Transport  | TCP Port 43                         | HTTP/HTTPS (Port 80/443)                |
| Format     | Unstructured Plain Text             | Structured JSON (machine-readable)      |
| Security   | None                                | Standard Web Security (TLS, Auth)       |
| Discovery  | Brittle, text-based referrals       | Standardized discovery.                 |

You can try it with `curl`:
```bash
curl -L https://rdap.verisign.com/com/v1/domain/google.com
```
{{< details-md summary="Output" >}}
{{% render-code file="google-rdap.json" language="json" %}}
{{< /details-md >}}

The response is a structured JSON object that is far easier to parse than the free-form text of WHOIS.

### Why `.dev` domains don't work

Many modern top-level domains (TLDs), like Google's `.dev`, have effectively abandoned WHOIS since they are no longer required to support it. They **only** provide registration data via RDAP.

If you try to look up a `.dev` domain with a standard `whois` client, you get a generic response from IANA pointing you to the `.dev` registry's information, which is useless for finding details about a specific domain like `kmcd.dev`.

```shell
$ whois kmcd.dev
# % IANA WHOIS server
# % for more information on IANA, visit http://www.iana.org
# % This query returned 1 object

# domain:       DEV
# organisation: Charleston Road Registry Inc.
# ...
# remarks:      Registration information: https://www.registry.google
# source:       IANA
```

While `whois` doesn't show much for `kmcd.dev`, `rdap` seems to have everything that you can normally get with WHOIS:

```shell
rdap kmcd.dev
```

{{< details-md summary="Output" >}}
{{% render-code file="kmcd-rdap.txt" language="text" %}}
{{< /details-md >}}

Even though `rdap` works, it isn't installed by default on most systems. Many people are probably going to forget to install `rdap` or will just default to `whois` out of habit. Muscle memory—and availability—are hard to beat. So I figured that one way to get the old `whois` command working again is by making a proxy that speaks the WHOIS protocol to the client and will fetch the data using RDAP.

### Building a WHOIS-to-RDAP Proxy

Let's build a smarter WHOIS server that acts as a proxy. It will listen for WHOIS queries on port 43. When it receives a query, it will make an HTTPS request to the appropriate RDAP server, parse the structured JSON response, format the key details into a human-readable text format, and send that text back to the original WHOIS client. Simple, no?

{{< d2 width="500px" >}}

style {
  stroke-width: 2
  font-size: 14
}

Client -> Proxy: WHOIS query (domain)
Proxy -> RDAP Server: HTTPS GET (domain)
RDAP Server -> Proxy: JSON response
Proxy -> Client: Formatted Text Response

{{< /d2 >}}

This approach makes RDAP-only domains accessible to legacy tools that only speak the classic WHOIS protocol. Although, let's be honest, you should probably just use the existing `rdap` command for anything serious. This is just a toy. But it was fun to make.

{{< image src="rdap-me-up.png" width="300px" class="center" >}}

Here is the implementation of our new WHOIS->RDAP proxy server:

{{< details-md summary="whois-server-proxy/main.go" github_file="go/whois-server-proxy/main.go" >}}
{{% render-code file="go/whois-server-proxy/main.go" language="go" %}}
{{< /details-md >}}

The output is formatted using a Go template to create a classic WHOIS-style report from the RDAP JSON data:
{{< details-md summary="whois-server-proxy/rdap.template" github_file="go/whois-server-proxy/rdap.template" >}}
{{% render-code file="go/whois-server-proxy/rdap.template" language="text/template" %}}
{{< /details-md >}}

With the proxy running, we can query it for `kmcd.dev` and get a complete and useful response using a standard `whois` client:
```bash
# Run the proxy in one terminal
go run ./whois-server-proxy

# Query it from another
whois -h localhost kmcd.dev
```

{{< details-md summary="Output" >}}
{{% render-code file="whois-proxy-output.txt" language="text" %}}
{{< /details-md >}}

### Conclusion

WHOIS is simple and approachable, but it belongs to a smaller and more trusting internet. It relies on unstructured text, inconsistent formatting, and informal conventions, an approach that simply does not scale in the modern era.

RDAP is the inevitable successor because it fixes the exact problems that made WHOIS brittle: transport, structure, discovery and security.

By wrapping the new standard in the old interface, we bridged the gap between the past and present. We get the structured power of RDAP without losing the muscle-memory efficiency of the command line.

This was a small, intentionally simple project, but it is a useful lens on how internet infrastructure evolves. Protocols rarely disappear overnight. They get replaced, translated, and slowly pushed out of the critical path.

### References

* [RFC 3912 - WHOIS Protocol Specification](https://www.rfc-editor.org/rfc/rfc3912)
* [WHOIS (Wikipedia)](https://en.wikipedia.org/wiki/WHOIS)
* [RDAP (Registration Data Access Protocol)](https://www.rfc-editor.org/rfc/rfc7480)
