package media

import (
	"os"

	"github.com/asano69/hashcards/internal/errs"
)

const (
	// MaxFileSizeBytes is the maximum accepted media file size (10 MiB).
	MaxFileSizeBytes = 10 * 1024 * 1024
)

// ValidationError describes a single media validation problem.
type ValidationError struct {
	// Ref is the original reference string from the Markdown source.
	Ref string
	// Err is the underlying error.
	Err error
}

func (v ValidationError) Error() string {
	return v.Err.Error()
}

// Validate checks that a resolved media file exists on disk, is a regular
// file, and does not exceed MaxFileSizeBytes. It returns a non-nil
// ValidationError when any check fails.
func Validate(resolved ResolveResult) *ValidationError {
	info, err := os.Stat(resolved.AbsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &ValidationError{
				Ref: resolved.AbsPath,
				Err: errs.Newf("media file does not exist: %s", resolved.AbsPath),
			}
		}
		return &ValidationError{
			Ref: resolved.AbsPath,
			Err: errs.Newf("stat media file %s: %v", resolved.AbsPath, err),
		}
	}

	if !info.Mode().IsRegular() {
		return &ValidationError{
			Ref: resolved.AbsPath,
			Err: errs.Newf("media path is not a regular file: %s", resolved.AbsPath),
		}
	}

	if info.Size() > MaxFileSizeBytes {
		return &ValidationError{
			Ref: resolved.AbsPath,
			Err: errs.Newf(
				"media file %s exceeds maximum size of %d bytes (got %d)",
				resolved.AbsPath, MaxFileSizeBytes, info.Size(),
			),
		}
	}

	return nil
}

// ValidateAll resolves and validates every ref in refs. All validation errors
// are collected and returned together so the user can fix all problems at once
// rather than discovering them one at a time.
func ValidateAll(deckFilePath string, refs []string) []ValidationError {
	var errs []ValidationError
	for _, ref := range refs {
		resolved, err := Resolve(deckFilePath, ref)
		if err != nil {
			errs = append(errs, ValidationError{Ref: ref, Err: err})
			continue
		}
		if ve := Validate(resolved); ve != nil {
			errs = append(errs, *ve)
		}
	}
	return errs
}
