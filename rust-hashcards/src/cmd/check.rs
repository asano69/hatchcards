
use crate::collection::Collection;
use crate::error::Fallible;

pub fn check_collection(directory: Option<String>) -> Fallible<()> {
    let _ = Collection::new(directory)?;
    println!("ok");
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::check_collection;
    use crate::error::Fallible;
    use crate::helper::create_tmp_copy_of_test_directory;

    #[test]
    fn test_non_existent_directory() {
        assert!(check_collection(Some("./derpherp".to_string())).is_err());
    }

    #[test]
    fn test_directory() -> Fallible<()> {
        let directory = create_tmp_copy_of_test_directory()?;
        assert!(check_collection(Some(directory)).is_ok());
        Ok(())
    }
}
