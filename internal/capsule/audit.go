package capsule

import (
	"os"
	"path/filepath"
	"strings"
)

var auditNeedles = []string{"ghp_", "github_pat_", "glpat-", "AKIA", "-----BEGIN "}

var auditExtensions = map[string]bool{".log": true, ".txt": true, ".json": true, ".yml": true, ".yaml": true, ".out": true}

// AuditPath reports supported credential indicators without printing matched values.
func AuditPath(path string) ([]string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	files := []string{path}
	if info.IsDir() {
		files = nil
		err = filepath.WalkDir(path, func(current string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if !entry.IsDir() {
				files = append(files, current)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	var findings []string
	for _, file := range files {
		if info.IsDir() && !auditExtensions[filepath.Ext(file)] {
			continue
		}
		bytes, readErr := os.ReadFile(file)
		if readErr != nil {
			return nil, readErr
		}
		for _, needle := range auditNeedles {
			if strings.Contains(string(bytes), needle) {
				findings = append(findings, file+": supported credential indicator "+needle)
			}
		}
	}
	return findings, nil
}
