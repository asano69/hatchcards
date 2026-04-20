
mod cli;
mod cmd;
mod collection;
mod db;
mod error;
mod fsrs;
#[cfg(test)]
mod helper;
mod markdown;
mod media;
mod parser;
mod rng;
mod types;
mod utils;

use std::process::ExitCode;

use crate::cli::entrypoint;

#[tokio::main]
async fn main() -> ExitCode {
    env_logger::init();
    match entrypoint().await {
        Ok(_) => ExitCode::SUCCESS,
        Err(e) => {
            eprintln!("hashcards: {e}");
            ExitCode::FAILURE
        }
    }
}
