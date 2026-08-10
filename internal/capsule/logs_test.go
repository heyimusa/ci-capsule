package capsule

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestReadLocalLogRejectsOverLimitEvidence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "large.log")
	if err := os.WriteFile(path, bytes.Repeat([]byte("x"), maxExpandedLogBytes+1), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadLocalLog(path); err == nil {
		t.Fatal("ReadLocalLog accepted oversized input")
	}
}

func TestNormalizeJobLogCountsSeparatorsInOutputLimit(t *testing.T) {
	var payload bytes.Buffer
	writer := zip.NewWriter(&payload)
	for index, size := range []int{maxExpandedLogBytes - 1, 1} {
		entry, err := writer.Create(fmt.Sprintf("%d.txt", index))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write(bytes.Repeat([]byte("x"), size)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := NormalizeJobLog(payload.Bytes()); err == nil {
		t.Fatal("NormalizeJobLog excluded separator from output limit")
	}
}

func TestNormalizeJobLogRejectsUnsafeZipEntry(t *testing.T) {
	var payload bytes.Buffer
	writer := zip.NewWriter(&payload)
	entry, err := writer.Create("../log.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte("FAILED")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := NormalizeJobLog(payload.Bytes()); err == nil {
		t.Fatal("NormalizeJobLog accepted unsafe ZIP entry")
	}
}

func TestNormalizeJobLogRejectsAggregateOverflow(t *testing.T) {
	var payload bytes.Buffer
	writer := zip.NewWriter(&payload)
	for _, name := range []string{"one.txt", "two.txt"} {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write(bytes.Repeat([]byte("x"), maxExpandedLogBytes/2+1)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := NormalizeJobLog(payload.Bytes()); err == nil {
		t.Fatal("NormalizeJobLog accepted aggregate overflow")
	}
}

func TestNormalizeJobLogAcceptsPlainText(t *testing.T) {
	got, err := NormalizeJobLog([]byte("FAILED"))
	if err != nil || got != "FAILED" {
		t.Fatalf("got=%q err=%v", got, err)
	}
}

func TestNormalizeJobLogExtractsTextFromZip(t *testing.T) {
	var archive bytes.Buffer
	writer := zip.NewWriter(&archive)
	file, err := writer.Create("test/9_Test.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte("FAILED")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	got, err := NormalizeJobLog(archive.Bytes())
	if err != nil || got != "FAILED" {
		t.Fatalf("got=%q err=%v", got, err)
	}
}
