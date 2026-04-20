
use std::error::Error;
use std::fmt::Display;
use std::fmt::Formatter;
use std::path::StripPrefixError;
use std::string::FromUtf8Error;

use crate::parser::ParserError;

#[derive(Debug, PartialEq)]
pub struct ErrorReport {
    message: String,
}

impl ErrorReport {
    pub fn new(msg: impl Into<String>) -> Self {
        ErrorReport {
            message: msg.into(),
        }
    }
}

impl From<std::io::Error> for ErrorReport {
    fn from(value: std::io::Error) -> Self {
        ErrorReport {
            message: format!("I/O error: {value:#?}"),
        }
    }
}

impl From<StripPrefixError> for ErrorReport {
    fn from(value: StripPrefixError) -> Self {
        ErrorReport {
            message: format!("Strip prefix error: {value:#?}"),
        }
    }
}

impl From<walkdir::Error> for ErrorReport {
    fn from(value: walkdir::Error) -> Self {
        ErrorReport {
            message: format!("directory traversal error: {value:#?}"),
        }
    }
}

impl From<rusqlite::Error> for ErrorReport {
    fn from(value: rusqlite::Error) -> Self {
        ErrorReport {
            message: format!("rusqlite: {value:#?}"),
        }
    }
}

#[cfg(test)]
impl From<reqwest::Error> for ErrorReport {
    fn from(value: reqwest::Error) -> Self {
        ErrorReport {
            message: format!("reqwest: {value:#?}"),
        }
    }
}

impl From<FromUtf8Error> for ErrorReport {
    fn from(value: FromUtf8Error) -> Self {
        ErrorReport {
            message: format!("UTF-8 conversion error: {value:#?}"),
        }
    }
}

impl From<serde_json::Error> for ErrorReport {
    fn from(value: serde_json::Error) -> Self {
        ErrorReport {
            message: format!("JSON error: {value:#?}"),
        }
    }
}

impl From<ParserError> for ErrorReport {
    fn from(value: ParserError) -> Self {
        ErrorReport {
            message: format!("Parse error: {value}"),
        }
    }
}

impl Display for ErrorReport {
    fn fmt(&self, f: &mut Formatter) -> std::fmt::Result {
        write!(f, "error: {}", self.message)
    }
}

impl Error for ErrorReport {
    fn description(&self) -> &str {
        &self.message
    }
}

pub type Fallible<T> = Result<T, ErrorReport>;

pub fn fail<T>(msg: impl Into<String>) -> Fallible<T> {
    Err(ErrorReport {
        message: msg.into(),
    })
}
