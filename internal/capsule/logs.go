package capsule

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
)

const maxExpandedLogBytes = 32 << 20

// ReadLocalLog reads local log evidence using the same bounded policy as GitHub log collection.
func ReadLocalLog(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return readBounded(file)
}

func readBounded(reader io.Reader) ([]byte, error) {
	content, err := io.ReadAll(io.LimitReader(reader, maxExpandedLogBytes+1))
	if err != nil {
		return nil, err
	}
	if len(content) > maxExpandedLogBytes {
		return nil, fmt.Errorf("job log exceeds %d bytes", maxExpandedLogBytes)
	}
	return content, nil
}

// NormalizeJobLog accepts the plain-text and ZIP shapes observed from GitHub's job-log endpoint.
// ZIP entries are bounded, path-neutralized, and concatenated only as text evidence.
func NormalizeJobLog(payload []byte) (string, error) {
	if !bytes.HasPrefix(payload, []byte("PK\x03\x04")) {
		return string(payload), nil
	}
	reader, err := zip.NewReader(bytes.NewReader(payload), int64(len(payload)))
	if err != nil {
		return "", fmt.Errorf("read job log zip: %w", err)
	}
	var output strings.Builder
	var written int64
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}
		if file.UncompressedSize64 > maxExpandedLogBytes {
			return "", fmt.Errorf("job log ZIP entry exceeds %d bytes", maxExpandedLogBytes)
		}
		if strings.Contains(file.Name, "..") {
			return "", fmt.Errorf("job log ZIP contains unsafe entry name")
		}
		entry, err := file.Open()
		if err != nil {
			return "", err
		}
		separator := int64(0)
		if output.Len() > 0 {
			separator = 1
		}
		remaining := maxExpandedLogBytes - written - separator
		if remaining < 0 {
			entry.Close()
			return "", fmt.Errorf("expanded job logs exceed %d bytes", maxExpandedLogBytes)
		}
		content, readErr := io.ReadAll(io.LimitReader(entry, remaining+1))
		closeErr := entry.Close()
		if readErr != nil {
			return "", readErr
		}
		if closeErr != nil {
			return "", closeErr
		}
		if int64(len(content)) > remaining {
			return "", fmt.Errorf("expanded job logs exceed %d bytes", maxExpandedLogBytes)
		}
		if output.Len() > 0 {
			output.WriteByte('\n')
		}
		output.Write(content)
		written += separator + int64(len(content))
	}
	return output.String(), nil
}
