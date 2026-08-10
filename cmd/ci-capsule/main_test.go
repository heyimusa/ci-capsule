package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunVersion(t *testing.T) {
	if err := run([]string{"--version"}); err != nil {
		t.Fatal(err)
	}
}

func TestRunCreateAndInspect(t *testing.T) {
	root := t.TempDir()
	workflow := filepath.Join(root, "workflow.yml")
	log := filepath.Join(root, "failed.log")
	if err := os.WriteFile(workflow, []byte("jobs:\n  test:\n    steps:\n      - name: Test\n        run: go test ./...\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(log, []byte("FAILED"), 0o600); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(root, "capsule")
	if err := run([]string{"create", "--workflow", workflow, "--log", log, "--job", "test", "--step", "Test", "--out", out}); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"inspect", "--bundle", out}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(out, "bundle.json")); err != nil {
		t.Fatal(err)
	}
}

func TestRunRejectsUnknownCommand(t *testing.T) {
	if err := run([]string{"destroy-world"}); err == nil || !strings.Contains(err.Error(), "pull|create|inspect|audit") {
		t.Fatalf("err = %v", err)
	}
}
