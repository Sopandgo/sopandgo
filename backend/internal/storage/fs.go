package storage

import (
	"fmt"
	"path/filepath"
	"strings"
)

func SafeJoin(baseDir, relPath string) (string, error) {
	cleanBase, err := filepath.Abs(baseDir)
	if err != nil {
		return "", err
	}

	full := filepath.Join(cleanBase, relPath)
	cleanFull, err := filepath.Abs(full)
	if err != nil {
		return "", err
	}

	rel, err := filepath.Rel(cleanBase, cleanFull)
	if err != nil {
		return "", err
	}

	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes base directory")
	}

	return cleanFull, nil
}
