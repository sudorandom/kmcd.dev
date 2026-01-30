use anyhow::Result;
use std::net::SocketAddr;
use tracing::info;
use wtransport::tls::Certificate;
use wtransport::{Endpoint, Server};

#[tokio::main]
async fn main() -> Result<()> {
    tracing_subscriber::fmt::init();

    let server_addr: SocketAddr = "0.0.0.0:4434".parse()?;

    let certificate = Certificate::load(
        "../go/localhost+1.pem",
        "../go/localhost+1-key.pem",
    )?;

    let server = Server::builder()
        .with_certificate(certificate)
        .build(Endpoint::server(server_addr)?)?;

    info!("WebTransport server listening on {}", server_addr);

    loop {
        let session_request = server.accept().await;
        tokio::spawn(async move {
            info!("Accepting incoming session...");

            let session = match session_request.await {
                Ok(session) => {
                    info!("Session accepted");
                    session
                }
                Err(err) => {
                    info!("Failed to accept session: {}", err);
                    return;
                }
            };

            loop {
                let stream_request = session.accept_bi_stream().await;

                match stream_request {
                    Ok(Some((mut send_stream, mut recv_stream))) => {
                        info!("Accepted a bidirectional stream");

                        let mut buffer = [0; 1024];
                        let bytes_read = match recv_stream.read(&mut buffer).await {
                            Ok(Some(n)) => n,
                            Ok(None) => {
                                info!("Stream closed by peer");
                                return;
                            }
                            Err(e) => {
                                info!("Error reading from stream: {}", e);
                                return;
                            }
                        };

                        let received_data = &buffer[..bytes_read];
                        info!("Received: {}", String::from_utf8_lossy(received_data));

                        if let Err(e) = send_stream.write_all(received_data).await {
                            info!("Error writing to stream: {}", e);
                        } else {
                            info!("Echoed back {} bytes", bytes_read);
                        }
                    }
                    Ok(None) => {
                        // No more streams from the client
                        break;
                    }
                    Err(e) => {
                        info!("Error accepting stream: {}", e);
                        break;
                    }
                }
            }
        });
    }
}
