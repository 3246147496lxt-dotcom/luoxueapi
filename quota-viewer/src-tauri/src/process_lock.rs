use std::{
    fs::{self, File, OpenOptions},
    io,
    path::Path,
};

use fs2::FileExt;

#[derive(Debug)]
pub(crate) struct ProcessLock {
    // The open handle owns the OS lock for the lifetime of the Tauri process.
    _file: File,
}

impl ProcessLock {
    pub(crate) fn acquire(path: &Path) -> io::Result<Self> {
        if let Some(parent) = path.parent() {
            fs::create_dir_all(parent)?;
        }
        let file = open_lock_file(path)?;
        if let Err(error) = FileExt::try_lock_exclusive(&file) {
            if error.kind() == fs2::lock_contended_error().kind() {
                return Err(io::Error::new(
                    io::ErrorKind::AlreadyExists,
                    "another Luoxue quota viewer process is already running",
                ));
            }
            return Err(error);
        }
        Ok(Self { _file: file })
    }
}

#[cfg(unix)]
fn open_lock_file(path: &Path) -> io::Result<File> {
    use std::os::unix::fs::OpenOptionsExt;

    OpenOptions::new()
        .create(true)
        .truncate(false)
        .read(true)
        .write(true)
        .mode(0o600)
        .open(path)
}

#[cfg(not(unix))]
fn open_lock_file(path: &Path) -> io::Result<File> {
    OpenOptions::new()
        .create(true)
        .truncate(false)
        .read(true)
        .write(true)
        .open(path)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn excludes_second_handle_and_releases_lock_on_drop() {
        let directory = tempfile::tempdir().unwrap();
        let path = directory.path().join("quota-viewer-process.lock");

        let first = ProcessLock::acquire(&path).unwrap();
        let error = ProcessLock::acquire(&path).unwrap_err();
        assert_eq!(error.kind(), io::ErrorKind::AlreadyExists);

        drop(first);
        let _replacement = ProcessLock::acquire(&path).unwrap();
    }
}
