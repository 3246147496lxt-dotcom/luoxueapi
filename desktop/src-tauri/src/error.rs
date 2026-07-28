use std::io;

#[derive(Debug, thiserror::Error)]
pub enum AppError {
    #[error("{0}")]
    Message(String),
    #[error("I/O error: {0}")]
    Io(#[from] io::Error),
    #[error("JSON error: {0}")]
    Json(#[from] serde_json::Error),
    #[error("TOML error: {0}")]
    Toml(#[from] toml_edit::TomlError),
    #[error("database error: {0}")]
    Database(#[from] rusqlite::Error),
    #[error("cloud request failed: {0}")]
    Cloud(#[from] reqwest::Error),
    #[error("cloud request rejected ({status}, {reason})")]
    CloudRejected { status: u16, reason: String },
    #[error("device was revoked and local recovery failed: {0}")]
    RemoteRevocation(String),
    #[error("credential store error: {0}")]
    Keyring(#[from] keyring::Error),
    #[error("configuration conflict: {0}")]
    ConfigConflict(String),
    #[error("configuration is unsafe to modify: {0}")]
    UnsafeConfig(String),
}

impl AppError {
    pub fn is_desktop_session_revoked(&self) -> bool {
        matches!(
            self,
            Self::CloudRejected { reason, .. }
                if matches!(
                    reason.as_str(),
                    "DESKTOP_DEVICE_FORBIDDEN"
                        | "DESKTOP_TOKEN_INVALID"
                        | "DESKTOP_REFRESH_INVALID"
                        | "DESKTOP_REFRESH_REPLAY"
                )
        )
    }

    pub fn is_remote_revocation(&self) -> bool {
        matches!(self, Self::RemoteRevocation(_))
    }
}

pub type AppResult<T> = Result<T, AppError>;
