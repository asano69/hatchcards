// Package hook resolves and runs pre-installed post-sync scripts. Scripts
// are never uploaded or authored through the API; an operator places them
// on disk ahead of time, so the only thing a connection stores is a name
// that gets resolved against this fixed, read-only directory. Commands are
// executed directly (no shell), so a hook name can never be used to inject
// arbitrary shell syntax.
package hook

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"

	"github.com/asano69/hatchcards/internal/errs"
)

// nameRe rejects anything but a bare identifier, so a name can never
// escape hooksDir via "../" or an absolute path.
var nameRe = regexp.MustCompile(`^[a-zA-Z0-9_][a-zA-Z0-9_.-]*$`)

// List returns the names of every executable regular file in hooksDir, i.e.
// every hook a connection is allowed to reference. It returns an empty list
// (not an error) when hooksDir doesn't exist, so installations that never
// configured a hooks directory behave exactly as before this feature existed.
func List(hooksDir string) ([]string, error) {
	entries, err := os.ReadDir(hooksDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, errs.Newf("read hooks dir: %v", err)
	}

	var names []string
	for _, e := range entries {
		if e.IsDir() || !nameRe.MatchString(e.Name()) {
			continue
		}
		info, err := e.Info()
		if err != nil || !info.Mode().IsRegular() || info.Mode()&0111 == 0 {
			continue
		}
		names = append(names, e.Name())
	}
	return names, nil
}

func Resolve(hooksDir, name string) (string, error) {
	if name == "" {
		return "", nil
	}
	if !nameRe.MatchString(name) {
		return "", errs.Newf("invalid hook name: %q", name)
	}
	path := filepath.Join(hooksDir, name)
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", errs.Newf("resolve hook path: %v", err)
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return "", errs.Newf("hook not found: %s", name)
	}
	if !info.Mode().IsRegular() || info.Mode()&0111 == 0 {
		return "", errs.Newf("hook is not executable: %s", name)
	}
	return absPath, nil
}

// Run executes the resolved script directly (no shell), passing the source
// and output directories both as positional arguments and as environment
// variables, so a hook script can use whichever convention is easiest —
// e.g. an existing "script.py <input_dir> <output_dir>" tool can be reused
// as-is. sourceDir is the connection's git working tree; outputDir is where
// generated JSON decks should be written, and is created if it doesn't
// already exist.
//
// Run always returns the script's combined stdout/stderr output, even on
// success, so the caller can surface it in the server log. This is what
// lets a script's own logging (e.g. Python's logging module) show up
// alongside the Go server's logs.
func Run(ctx context.Context, scriptPath, sourceDir, outputDir string) (string, error) {
	absSource, err := filepath.Abs(sourceDir)
	if err != nil {
		return "", errs.Newf("resolve source dir: %v", err)
	}
	absOutput, err := filepath.Abs(outputDir)
	if err != nil {
		return "", errs.Newf("resolve output dir: %v", err)
	}

	if err := os.MkdirAll(absOutput, 0o755); err != nil {
		return "", errs.Newf("create hook output dir: %v", err)
	}

	cmd := exec.CommandContext(ctx, scriptPath, absSource, absOutput)
	cmd.Env = append(os.Environ(),
		"HATCHCARDS_SOURCE_DIR="+absSource,
		"HATCHCARDS_OUTPUT_DIR="+absOutput,
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), errs.Newf("hook %q failed: %v\n%s", filepath.Base(scriptPath), err, out)
	}
	return string(out), nil
}
