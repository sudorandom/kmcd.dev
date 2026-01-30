# WebTransport Go Samples

This repository contains a series of sample programs for client and servers that send and receive WebTransport messages using `github.com/quic-go/webtransport-go`.

## Generating Certificates

To run the server, you need a TLS certificate and a private key. You can generate a self-signed certificate for local testing using `openssl`.

```bash
openssl req -x509 -newkey rsa:2048 -keyout cert.key -out cert.pem -days 365 -nodes -subj "/C=US/ST=CA/L=San Francisco/O=WebTransport/OU=Development/CN=localhost"
```

This command will create two files: `cert.pem` (the certificate) and `cert.key` (the private key). Place these files in the `server` directory.
