use std::{
    fs,
    path::{Path, PathBuf},
};

use chrono::{Duration, Utc};
use rusqlite::{params, Connection, OptionalExtension, TransactionBehavior};

use crate::{
    error::AppResult,
    models::{RequestMetadata, TodaySummary},
};

#[derive(Clone)]
pub struct LocalStorage {
    path: PathBuf,
}

impl LocalStorage {
    pub fn open(path: impl AsRef<Path>) -> AppResult<Self> {
        let path = path.as_ref().to_path_buf();
        if let Some(parent) = path.parent() {
            fs::create_dir_all(parent)?;
        }
        let storage = Self { path };
        storage.with_connection(|connection| {
            connection.execute_batch(
                "PRAGMA journal_mode=WAL;
                 PRAGMA foreign_keys=ON;
                 CREATE TABLE IF NOT EXISTS session_routes (
                   session_hash TEXT PRIMARY KEY,
                   group_id INTEGER NOT NULL,
                   expires_at INTEGER NOT NULL
                 );
                 CREATE TABLE IF NOT EXISTS requests (
                   id TEXT PRIMARY KEY,
                   occurred_at TEXT NOT NULL,
                   model TEXT NOT NULL,
                   status_code INTEGER NOT NULL,
                   input_tokens INTEGER NOT NULL,
                   output_tokens INTEGER NOT NULL,
                   duration_ms INTEGER NOT NULL,
                   first_token_ms INTEGER,
                   request_id TEXT NOT NULL
                 );
                 CREATE INDEX IF NOT EXISTS requests_occurred_at_idx ON requests(occurred_at DESC);"
            )?;
            Ok(())
        })?;
        Ok(storage)
    }

    fn with_connection<T>(
        &self,
        operation: impl FnOnce(&mut Connection) -> AppResult<T>,
    ) -> AppResult<T> {
        let mut connection = Connection::open(&self.path)?;
        connection.busy_timeout(std::time::Duration::from_secs(2))?;
        operation(&mut connection)
    }

    pub fn resolve_session_group(
        &self,
        session_hash: &str,
        current_group_id: i64,
    ) -> AppResult<i64> {
        self.with_connection(|connection| {
            let transaction = connection.transaction_with_behavior(TransactionBehavior::Immediate)?;
            let now = Utc::now().timestamp();
            transaction.execute("DELETE FROM session_routes WHERE expires_at <= ?1", [now])?;
            let existing = transaction
                .query_row(
                    "SELECT group_id FROM session_routes WHERE session_hash = ?1",
                    [session_hash],
                    |row| row.get::<_, i64>(0),
                )
                .optional()?;
            let group_id = existing.unwrap_or(current_group_id);
            if existing.is_none() {
                transaction.execute(
                    "INSERT INTO session_routes(session_hash, group_id, expires_at) VALUES (?1, ?2, ?3)",
                    params![session_hash, group_id, (Utc::now() + Duration::days(30)).timestamp()],
                )?;
            }
            transaction.commit()?;
            Ok(group_id)
        })
    }

    pub fn insert_request(&self, item: &RequestMetadata) -> AppResult<()> {
        self.with_connection(|connection| {
            connection.execute(
                "INSERT OR REPLACE INTO requests(
                   id, occurred_at, model, status_code, input_tokens, output_tokens,
                   duration_ms, first_token_ms, request_id
                 ) VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9)",
                params![
                    item.id,
                    item.occurred_at,
                    item.model,
                    item.status_code,
                    item.input_tokens,
                    item.output_tokens,
                    item.duration_ms,
                    item.first_token_ms,
                    item.request_id,
                ],
            )?;
            Ok(())
        })
    }

    pub fn list_requests(
        &self,
        limit: usize,
        retention_days: u32,
    ) -> AppResult<Vec<RequestMetadata>> {
        self.cleanup(retention_days)?;
        self.with_connection(|connection| {
            let mut statement = connection.prepare(
                "SELECT id, occurred_at, model, status_code, input_tokens, output_tokens,
                        duration_ms, first_token_ms, request_id
                 FROM requests ORDER BY occurred_at DESC LIMIT ?1",
            )?;
            let rows = statement.query_map([limit as i64], |row| {
                Ok(RequestMetadata {
                    id: row.get(0)?,
                    occurred_at: row.get(1)?,
                    model: row.get(2)?,
                    status_code: row.get(3)?,
                    input_tokens: row.get(4)?,
                    output_tokens: row.get(5)?,
                    duration_ms: row.get(6)?,
                    first_token_ms: row.get(7)?,
                    request_id: row.get(8)?,
                })
            })?;
            Ok(rows.collect::<Result<Vec<_>, _>>()?)
        })
    }

    pub fn today_summary(&self) -> AppResult<TodaySummary> {
        self.with_connection(|connection| {
            let today = Utc::now()
                .date_naive()
                .and_hms_opt(0, 0, 0)
                .expect("valid midnight")
                .and_utc()
                .to_rfc3339();
            let summary = connection.query_row(
                "SELECT COUNT(*), COALESCE(SUM(input_tokens + output_tokens), 0),
                        CAST(AVG(first_token_ms) AS INTEGER)
                 FROM requests WHERE occurred_at >= ?1",
                [today],
                |row| {
                    Ok(TodaySummary {
                        requests: row.get(0)?,
                        tokens: row.get(1)?,
                        cost: 0.0,
                        balance: 0.0,
                        average_first_token_ms: row.get(2)?,
                    })
                },
            )?;
            Ok(summary)
        })
    }

    pub fn cleanup(&self, retention_days: u32) -> AppResult<()> {
        self.with_connection(|connection| {
            let cutoff = (Utc::now() - Duration::days(i64::from(retention_days))).to_rfc3339();
            connection.execute("DELETE FROM requests WHERE occurred_at < ?1", [cutoff])?;
            Ok(())
        })
    }

    pub fn clear_private_state(&self) -> AppResult<()> {
        self.with_connection(|connection| {
            connection.execute_batch("DELETE FROM session_routes; DELETE FROM requests;")?;
            Ok(())
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn pins_existing_session_to_original_group() {
        let directory = tempfile::tempdir().unwrap();
        let storage = LocalStorage::open(directory.path().join("state.db")).unwrap();
        assert_eq!(storage.resolve_session_group("hash", 12).unwrap(), 12);
        assert_eq!(storage.resolve_session_group("hash", 18).unwrap(), 12);
    }
}
