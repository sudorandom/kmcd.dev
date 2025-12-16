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

The `whois` protocol is dead. For decades, it was a fundamental tool for network reconnaissance, but its time has passed. The protocol was officially sunset for all generic top-level domains in early 2025, replaced by a modern, secure, and structured successor: [RDAP](https://about.rdap.org/).

{{< image src="whois-dead.png" width="500px" class="center" >}}

So why talk about it now? To pay our respects. The WHOIS protocol, in all its simplistic, text-based glory, is a perfect case study for basic network programming and a window into an earlier era of the internet. To memorialize this piece of internet history, we will build a tiny implementation from scratch and, in the process, understand why its death was necessary.

#### What is WHOIS?

From a user's perspective, WHOIS is the "caller ID" of the internet. It answers the fundamental question: "Who owns this?"

When you run a query for `example.com`, you aren't just asking for an IP address (like DNS); you are asking for the administrative metadata behind that address. A typical response provides the name of the Registrar that sold the domain (e.g., Namecheap, GoDaddy), the Name Servers responsible for translating the domain into an IP, and critical Dates—specifically when the domain was created and when it will expire. Historically, this also included the full contact details of the owner, though privacy laws like GDPR have largely pushed that data behind redaction services.

### Who provides this data?

WHOIS isn't a single central database. It is a distributed requirement enforced by ICANN (Internet Corporation for Assigned Names and Numbers). ICANN sets the rules, Registries (like Verisign for .com) manage the master lists for their specific TLDs, and Registrars (like Google Domains) sell the names to you. These registrars are contractually obligated to maintain this registration data and make it available to the public.

### Why do we need it?

While often used by developers checking if a cool side project name is taken, WHOIS is critical infrastructure for maintaining the internet's health. Network Operators use it to contact admins when a specific domain is spamming or attacking a network. Security Researchers rely on it to identify "co-location" of malicious domains—for example, noticing that a malware site was registered by the same email address as 50 other suspicious domains.

It is also a powerful instrument for accountability. Investigative journalists have historically used WHOIS as a "digital paper trail," using contact information to map the infrastructure of state-sponsored disinformation campaigns or uncover the real-world actors behind criminal shell companies.

### How WHOIS Works

The WHOIS protocol, defined in [RFC 3912](https://www.rfc-editor.org/rfc/rfc3912), is a simple exchange over a TCP connection on port 43.

1.  A client opens a TCP socket to a WHOIS server on port 43.
2.  The client sends the query, a single line of text like `example.com`, terminated by a carriage return and line feed (`<CR><LF>`).
3.  The server sends back a stream of plain text containing the registration data.
4.  The server closes the connection.

There are no headers, no authentication, and no complex data formats. This simplicity makes it a good candidate for a small project to demonstrate basic networking concepts.

### Building a WHOIS Server

We can build a server in Go to respond to WHOIS queries. Standard tools like `telnet` or the `whois` command itself are sufficient for testing.

#### The Server's Job

A WHOIS server has a minimal set of responsibilities:
1.  Listen for incoming TCP connections on port 43.
2.  For each connection, read the single-line query from the client.
3.  Look up the requested information.
4.  Write the response back to the client.
5.  Close the connection.

Our implementation will keep a simple in-memory map to store domain records.

#### WHOIS Server Implementation

We'll add a `records` map to hold fake domain data and implement a `handleConnection` function to process queries and send back the corresponding record.

{{< details-md summary="whois-server/main.go" github_file="go/whois-server/main.go" >}}
{{% render-code file="go/whois-server/main.go" language="go" %}}
{{< /details-md >}}

To run the server, execute:
```bash
go run ./whois-server
```

With the server running, standard tools can now interact with it:

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

This system is brittle, relying on parsing unstructured text to find the referral server. Classic `whois` clients, such as the [`rfc1036/whois`](https://github.com/rfc1036/whois/blob/next/whois.c), handle this by scanning each line of text for known referral markers using functions like `find_referral_server_iana`. This approach works but is fragile, because every registry formats output differently. The brittleness of parsing free-form text was a key driver to replace it with a modern protocol that uses structured data like JSON.

### RDAP: The Modern Successor

The push to replace it began in earnest back in 2013, when an ICANN Expert Working Group recommended scrapping WHOIS entirely. They proposed a system that would keep information secret from most users, disclosing data only for specific "permissible purposes" like legal actions or trademark enforcement. Notably, **journalism was excluded** from this list, despite WHOIS historically being a key tool for investigative reporting.

After years of debate and voting, the transition became official. On January 28, 2025, WHOIS was officially sunset for generic Top-Level Domains (gTLDs). While the port 43 service didn't vanish overnight, registries are no longer required to support it, and the industry has shifted its focus to RDAP (Registration Data Access Protocol).

RDAP performs the same function as WHOIS but over HTTPS, returning structured JSON instead of raw text.

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

The response is a clean, predictable JSON object, which is far easier to parse than the free-form text of WHOIS.

### Why `.dev` domains don't work

This is not a theoretical transition. Many modern top-level domains (TLDs), like Google's `.dev`, have effectively abandoned WHOIS entirely since they are no longer required to support it. They **only** provide registration data via RDAP.

If you try to look up a `.dev` domain with a standard `whois` client, you get a generic response from IANA pointing you to the `.dev` registry's information, which is useless for finding the owner of a specific domain like `kmcd.dev`.

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

While `whois` doesn't show much for `kmcd.dev`, `rdap` seems to have everything that you can normally get with whois:

```shell
rdap kmcd.dev
```

{{< details-md summary="Output" >}}
{{% render-code file="kmcd-rdap.txt" language="text" %}}
{{< /details-md >}}

Even though `rdap` works, it isn't installed by default on most systems. I'm probably going to forget to install it on my new laptop, or just default to `whois` out of habit. Muscle memory—and availability—are hard to beat. So I figured that one way to get the old `whois` command working again is by making a proxy that speaks the WHOIS protocol to the client and will fetch the data using RDAP.

### Building a WHOIS-to-RDAP Proxy

Let's build a smarter WHOIS server that acts as a proxy. It will listen for traditional WHOIS queries on port 43. Upon receiving a query, it will make an HTTPS request to the appropriate RDAP server, parse the structured JSON response, format the key details into a human-readable text format, and send that text back to the original WHOIS client.

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

This approach makes RDAP-only domains accessible to legacy tools that only speak the classic WHOIS protocol. Although, let's be real, you should probably just use the existing `rdap` command for anything serious.

{{< image src="rdap-me-up.png" width="300px" class="center" >}}

Here is the implementation of our proxy server:

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

WHOIS is simple and approachable, but it belongs to a smaller and more trusting internet. Building a WHOIS server from scratch is useful because it makes those assumptions obvious. The protocol’s minimalism is what makes it easy to learn, and it is also what limits it.

As the internet grew, the cracks became impossible to ignore. WHOIS depends on unstructured text, inconsistent formatting, and informal conventions between registries and registrars. Clients are expected to scrape meaning out of free-form output and follow referrals that may or may not exist. That approach does not scale, and it does not age well.

RDAP is what replacing that system actually looks like. Queries move over HTTPS. Responses are structured and predictable. Discovery is standardized instead of implied. The fact that some modern TLDs never supported WHOIS at all says more than any deprecation notice.

A WHOIS-to-RDAP proxy makes the transition easier to see. Old tools still function, but only by leaning on a protocol that was designed for the current internet. At that point, WHOIS is no longer the system of record. It is just the interface.

### References

* [RFC 3912 - WHOIS Protocol Specification](https://www.rfc-editor.org/rfc/rfc3912)
* [WHOIS (Wikipedia)](https://en.wikipedia.org/wiki/WHOIS)
* [RDAP (Registration Data Access Protocol)](https://www.rfc-editor.org/rfc/rfc7480)
