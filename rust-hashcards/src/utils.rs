
use std::time::Duration;

use tokio::net::TcpStream;
use tokio::time::sleep;

use crate::error::Fallible;

// max-age is one week in seconds.
pub const CACHE_CONTROL_IMMUTABLE: &str = "public, max-age=604800, immutable";

pub async fn wait_for_server(host: &str, port: u16) -> Fallible<()> {
    loop {
        if let Ok(stream) = TcpStream::connect(format!("{host}:{port}")).await {
            drop(stream);
            break;
        }
        sleep(Duration::from_millis(1)).await;
    }
    Ok(())
}
