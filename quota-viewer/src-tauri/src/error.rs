use serde::Serialize;

pub type AppResult<T> = Result<T, AppError>;
pub type CommandResult<T> = Result<T, CommandError>;

#[derive(Debug, thiserror::Error)]
pub enum AppError {
    #[error("network request failed")]
    Network(#[from] reqwest::Error),
    #[error("invalid URL")]
    Url(#[from] url::ParseError),
    #[error("secure storage is unavailable")]
    Keyring(#[from] keyring::Error),
    #[error("local cache is unavailable")]
    Io(#[from] std::io::Error),
    #[error("local cache is invalid")]
    Json(#[from] serde_json::Error),
    #[error("request rejected: {reason}")]
    CloudRejected { status: u16, reason: String },
    #[error("quota refresh state is inconsistent; reconnect the account")]
    RefreshStateInvalid,
    #[error("{0}")]
    Message(String),
}

impl AppError {
    pub fn is_authorization_invalid(&self) -> bool {
        match self {
            Self::RefreshStateInvalid => true,
            Self::CloudRejected { status, reason } => {
                *status == 401
                    || *status == 403
                    || matches!(
                        reason.as_str(),
                        "QUOTA_REFRESH_TOKEN_INVALID"
                            | "QUOTA_REFRESH_TOKEN_REPLAY"
                            | "QUOTA_REFRESH_ROTATION_SUPERSEDED"
                            | "QUOTA_REFRESH_RECOVERY_EXPIRED"
                            | "QUOTA_AUTH_DEVICE_UNAUTHORIZED"
                            | "QUOTA_ACCESS_TOKEN_INVALID"
                            | "QUOTA_ACCESS_TOKEN_EXPIRED"
                            | "QUOTA_ACCESS_TOKEN_AUDIENCE_FORBIDDEN"
                            | "QUOTA_ACCESS_TOKEN_CLIENT_FORBIDDEN"
                            | "QUOTA_ACCESS_TOKEN_SCOPE_FORBIDDEN"
                    )
            }
            _ => false,
        }
    }

    pub fn is_access_token_recoverable(&self) -> bool {
        matches!(
            self,
            Self::CloudRejected { status: 401, reason }
                if matches!(
                    reason.as_str(),
                    "QUOTA_ACCESS_TOKEN_INVALID"
                        | "QUOTA_ACCESS_TOKEN_EXPIRED"
                        | "QUOTA_AUTH_REQUIRED"
                )
        )
    }

    pub fn code(&self) -> String {
        match self {
            Self::Network(error) if error.is_timeout() => "NETWORK_TIMEOUT".into(),
            Self::Network(_) => "NETWORK_UNAVAILABLE".into(),
            Self::Url(_) => "INVALID_URL".into(),
            Self::Keyring(_) => "SECURE_STORAGE_UNAVAILABLE".into(),
            Self::Io(_) | Self::Json(_) => "LOCAL_CACHE_UNAVAILABLE".into(),
            Self::CloudRejected { reason, .. } if !reason.is_empty() => reason.clone(),
            Self::CloudRejected { status, .. } => format!("HTTP_{status}"),
            Self::RefreshStateInvalid => "QUOTA_REFRESH_PROTOCOL_INVALID".into(),
            Self::Message(_) => "QUOTA_VIEWER_ERROR".into(),
        }
    }

    pub fn retryable(&self) -> bool {
        match self {
            Self::Network(_) | Self::Io(_) | Self::Json(_) => true,
            Self::CloudRejected { status, .. } => {
                *status == 408 || *status == 429 || *status >= 500
            }
            _ => false,
        }
    }
}

#[derive(Clone, Debug, Serialize)]
pub struct CommandError {
    pub code: String,
    pub message: String,
    pub retryable: bool,
}

impl From<AppError> for CommandError {
    fn from(error: AppError) -> Self {
        Self {
            code: error.code(),
            message: error.to_string(),
            retryable: error.retryable(),
        }
    }
}

impl From<String> for CommandError {
    fn from(message: String) -> Self {
        AppError::Message(message).into()
    }
}

impl From<&str> for CommandError {
    fn from(message: &str) -> Self {
        AppError::Message(message.to_string()).into()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn terminal_refresh_rotation_reasons_invalidate_even_if_transport_status_is_lost() {
        for reason in [
            "QUOTA_REFRESH_TOKEN_INVALID",
            "QUOTA_REFRESH_ROTATION_SUPERSEDED",
            "QUOTA_REFRESH_RECOVERY_EXPIRED",
        ] {
            assert!(AppError::CloudRejected {
                status: 200,
                reason: reason.into(),
            }
            .is_authorization_invalid());
        }
    }
}
