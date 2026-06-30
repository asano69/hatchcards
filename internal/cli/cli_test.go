package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunBindsOutputWriter(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	collectionRoot := t.TempDir()
	err := Run([]string{"check", collectionRoot}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("Run() error = %v; stderr = %q", err, stderr.String())
	}

	if got := stdout.String(); !strings.Contains(got, "OK: 0 card(s) checked.") {
		t.Fatalf("stdout = %q, want check success message", got)
	}
}
