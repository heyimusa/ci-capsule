package capsule

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAuditPathIgnoresSourceAndBinaryFilesByDefault(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "rules.go"), []byte(`var prefix = "ghp_"`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "bundle.json"), []byte(`{"ok":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	findings, err := AuditPath(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 {
		t.Fatalf("findings = %#v", findings)
	}
}

func TestAuditPathReportsIndicatorWithoutLeakingValue(t *testing.T) {
	path := filepath.Join(t.TempDir(), "log.txt")
	if err := os.WriteFile(path, []byte("token=ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"), 0o600); err != nil {
		t.Fatal(err)
	}
	findings, err := AuditPath(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 || findings[0] != path+": supported credential indicator ghp_" {
		t.Fatalf("findings = %#v", findings)
	}
}
