
use std::fs::copy;
use std::fs::create_dir_all;
use std::path::PathBuf;

use tempfile::tempdir;

use crate::error::Fallible;

pub fn create_tmp_directory() -> Fallible<PathBuf> {
    let dir: PathBuf = tempdir()?.path().to_path_buf().canonicalize()?;
    create_dir_all(&dir)?;
    Ok(dir)
}

pub fn create_tmp_copy_of_test_directory() -> Fallible<String> {
    let source: PathBuf = PathBuf::from("./test").canonicalize()?;
    let target: PathBuf = create_tmp_directory()?;
    let files = ["Deck.md", "foo.jpg", "macros.tex"];
    for file in files {
        copy(source.join(file), target.join(file))?;
    }
    Ok(target.display().to_string())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_create_tmp_copy_of_test_directory() -> Fallible<()> {
        let result = create_tmp_copy_of_test_directory();
        assert!(result.is_ok());
        Ok(())
    }
}
