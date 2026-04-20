
use std::path::PathBuf;
use std::sync::Arc;
use std::sync::Mutex;

use tokio::sync::oneshot::Sender;

use crate::cmd::drill::cache::Cache;
use crate::cmd::drill::server::AnswerControls;
use crate::db::Database;
use crate::db::ReviewRecord;
use crate::fsrs::Difficulty;
use crate::fsrs::Grade;
use crate::fsrs::Stability;
use crate::types::card::Card;
use crate::types::date::Date;
use crate::types::timestamp::Timestamp;

#[derive(Clone)]
pub struct ServerState {
    pub port: u16,
    pub directory: PathBuf,
    pub macros: Vec<(String, String)>,
    pub total_cards: usize,
    pub session_started_at: Timestamp,
    pub mutable: Arc<Mutex<MutableState>>,
    pub shutdown_tx: Arc<Mutex<Option<Sender<()>>>>,
    pub answer_controls: AnswerControls,
}

pub struct MutableState {
    pub reveal: bool,
    pub db: Database,
    pub cache: Cache,
    pub cards: Vec<Card>,
    pub reviews: Vec<Review>,
    pub finished_at: Option<Timestamp>,
}

#[derive(Clone)]
pub struct Review {
    pub card: Card,
    pub reviewed_at: Timestamp,
    pub grade: Grade,
    pub stability: Stability,
    pub difficulty: Difficulty,
    pub interval_raw: f64,
    pub interval_days: i64,
    pub due_date: Date,
}

impl Review {
    pub fn should_repeat(&self) -> bool {
        self.grade == Grade::Forgot || self.grade == Grade::Hard
    }

    pub fn into_record(self) -> ReviewRecord {
        ReviewRecord {
            card_hash: self.card.hash(),
            reviewed_at: self.reviewed_at,
            grade: self.grade,
            stability: self.stability,
            difficulty: self.difficulty,
            interval_raw: self.interval_raw,
            interval_days: self.interval_days,
            due_date: self.due_date,
        }
    }
}
