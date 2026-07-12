# WebTransport Go Samples

This repository contains a series of sample programs for client and servers that send and receive WebTransport messages using `github.com/quic-go/webtransport-go`.

## Generating Certificates

To run the server in a browser, you need a locally trusted TLS certificate. This sample uses `mkcert`, installed through `mise`.

```bash
mkcert -install
mkcert localhost
```

Run these commands from this `go` directory. `mkcert` creates the conventional
`localhost.pem` and `localhost-key.pem` files loaded by both sample servers.

## Testing WebTransport in Browsers

Installing the mkcert CA makes the certificate valid for normal HTTPS, but
Firefox and Chromium-based browsers apply additional policies to HTTP/3 and
QUIC. Configure the browser before opening `http://localhost:8080`.

### Firefox

Firefox trusts the mkcert CA for HTTPS, but disables HTTP/3 when it detects a
user-installed root CA. WebTransport therefore fails with `WebTransport
connection rejected` until local HTTP/3 is enabled:

1. Open `about:config`.
2. Set `network.http.http3.disable_when_third_party_roots_found` to `false`.
3. Restart Firefox.

This preference affects the entire Firefox profile and should only be changed
in a development profile.

### Chrome, Edge, and Brave

Chrome uses the system trust store, where `mkcert -install` installs the local
CA. WebTransport over HTTP/3 applies an additional requirement that the
certificate be issued by a publicly known root. For local development, enable
the built-in WebTransport developer mode to relax that additional requirement:

1. Open `chrome://flags/#webtransport-developer-mode`.
2. Set **WebTransport Developer Mode** to **Enabled**.
3. Relaunch the browser.

The browser still validates the certificate against the system trust store;
this flag allows that trusted root to be a locally installed CA such as
mkcert. It is intended only for development.

## Browser Test

This sample includes a Playwright test that starts the Go server and launches
installed Chrome in WebTransport developer mode. Firefox is tested
manually because Playwright creates an isolated Firefox profile that does not
inherit the mkcert CA installed in your normal Firefox profile.

```bash
npm install
npm run test:e2e:chrome
```
