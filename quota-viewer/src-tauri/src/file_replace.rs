use std::{io, path::Path};

#[cfg(any(not(windows), test))]
use std::fs;

pub(crate) fn replace_file(source: &Path, destination: &Path) -> io::Result<()> {
    replace_file_impl(source, destination)
}

#[cfg(not(windows))]
fn replace_file_impl(source: &Path, destination: &Path) -> io::Result<()> {
    fs::rename(source, destination)
}

#[cfg(windows)]
fn replace_file_impl(source: &Path, destination: &Path) -> io::Result<()> {
    use std::os::windows::ffi::OsStrExt;
    use windows_sys::Win32::Storage::FileSystem::{
        MoveFileExW, MOVEFILE_REPLACE_EXISTING, MOVEFILE_WRITE_THROUGH,
    };

    fn wide_path(path: &Path) -> io::Result<Vec<u16>> {
        let mut wide = path.as_os_str().encode_wide().collect::<Vec<_>>();
        if wide.contains(&0) {
            return Err(io::Error::new(
                io::ErrorKind::InvalidInput,
                "file path contains an embedded NUL",
            ));
        }
        wide.push(0);
        Ok(wide)
    }

    let source = wide_path(source)?;
    let destination = wide_path(destination)?;
    // SAFETY: both paths are valid, NUL-terminated UTF-16 buffers that remain
    // alive for the duration of the call. The temporary file is always created
    // beside its destination, so MoveFileExW stays on the same volume.
    let moved = unsafe {
        MoveFileExW(
            source.as_ptr(),
            destination.as_ptr(),
            MOVEFILE_REPLACE_EXISTING | MOVEFILE_WRITE_THROUGH,
        )
    };
    if moved == 0 {
        return Err(io::Error::last_os_error());
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn replaces_an_existing_destination() {
        let directory = tempfile::tempdir().unwrap();
        let source = directory.path().join("next.tmp");
        let destination = directory.path().join("current.json");
        fs::write(&source, b"next").unwrap();
        fs::write(&destination, b"current").unwrap();

        replace_file(&source, &destination).unwrap();

        assert_eq!(fs::read(destination).unwrap(), b"next");
        assert!(!source.exists());
    }
}
