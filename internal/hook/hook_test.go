package hook

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func writeExecutable(t *testing.T, dir, name, body string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return path
}

func TestListMissingDir(t *testing.T) {
	names, err := List(filepath.Join(t.TempDir(), "does-not-exist"))
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(names) != 0 {
		t.Errorf("expected no hooks for a missing dir, got %v", names)
	}
}

func TestListOnlyExecutables(t *testing.T) {
	dir := t.TempDir()
	writeExecutable(t, dir, "convert", "#!/bin/sh\necho ok\n")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("not a hook"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "subdir"), 0o755); err != nil {
		t.Fatal(err)
	}

	names, err := List(dir)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(names) != 1 || names[0] != "convert" {
		t.Errorf("List = %v, want [convert]", names)
	}
}

func TestResolveEmptyName(t *testing.T) {
	path, err := Resolve(t.TempDir(), "")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if path != "" {
		t.Errorf("Resolve(\"\") = %q, want empty", path)
	}
}

func TestResolveRejectsPathTraversal(t *testing.T) {
	if _, err := Resolve(t.TempDir(), "../etc/passwd"); err == nil {
		t.Error("expected error for path-traversal hook name, got nil")
	}
}

func TestResolveRejectsMissingHook(t *testing.T) {
	if _, err := Resolve(t.TempDir(), "missing"); err == nil {
		t.Error("expected error for nonexistent hook, got nil")
	}
}

func TestResolveRejectsNonExecutable(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "notexec"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Resolve(dir, "notexec"); err == nil {
		t.Error("expected error for non-executable hook, got nil")
	}
}

func TestResolveValidHook(t *testing.T) {
	dir := t.TempDir()
	want := writeExecutable(t, dir, "convert", "#!/bin/sh\n")
	got, err := Resolve(dir, "convert")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got != want {
		t.Errorf("Resolve = %q, want %q", got, want)
	}
}

func TestRunWritesOutput(t *testing.T) {
	dir := t.TempDir()
	script := writeExecutable(t, dir, "gen", `#!/bin/sh
echo '[]' > "$HATCHCARDS_OUTPUT_DIR/deck.json"
`)
	sourceDir := t.TempDir()
	outputDir := filepath.Join(t.TempDir(), "generated")

	if _, err := Run(context.Background(), script, sourceDir, outputDir); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outputDir, "deck.json")); err != nil {
		t.Errorf("expected deck.json to be written: %v", err)
	}
}

// TestRunPassesDirsAsArguments verifies that $1/$2 receive the same
// directories as the HATCHCARDS_SOURCE_DIR / HATCHCARDS_OUTPUT_DIR
// environment variables, so scripts that expect positional arguments
// (e.g. "script.py <input_dir> <output_dir>") work without a wrapper.
func TestRunPassesDirsAsArguments(t *testing.T) {
	dir := t.TempDir()
	script := writeExecutable(t, dir, "gen", `#!/bin/sh
echo "$1" > "$2/args-input.txt"
echo "$2" > "$2/args-output.txt"
`)
	sourceDir := t.TempDir()
	outputDir := filepath.Join(t.TempDir(), "generated")

	if _, err := Run(context.Background(), script, sourceDir, outputDir); err != nil {
		t.Fatalf("Run: %v", err)
	}

	gotInput, err := os.ReadFile(filepath.Join(outputDir, "args-input.txt"))
	if err != nil {
		t.Fatalf("read args-input.txt: %v", err)
	}
	if got := string(gotInput); got != sourceDir+"\n" {
		t.Errorf("$1 = %q, want %q", got, sourceDir+"\n")
	}

	gotOutput, err := os.ReadFile(filepath.Join(outputDir, "args-output.txt"))
	if err != nil {
		t.Fatalf("read args-output.txt: %v", err)
	}
	if got := string(gotOutput); got != outputDir+"\n" {
		t.Errorf("$2 = %q, want %q", got, outputDir+"\n")
	}
}

func TestRunFailurePropagatesOutput(t *testing.T) {
	dir := t.TempDir()
	script := writeExecutable(t, dir, "fail", `#!/bin/sh
echo "boom" >&2
exit 1
`)
	_, err := Run(context.Background(), script, t.TempDir(), filepath.Join(t.TempDir(), "generated"))
	if err == nil {
		t.Fatal("expected error from failing hook, got nil")
	}
}
