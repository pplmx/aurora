package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

func NormalizePath(path string) (string, error) {
	// 1. transform to an absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to transform(%s) to an absolute path: %w", path, err)
	}

	// 2. check if the path is valid
	cleanPath := filepath.Clean(absPath)
	if _, err := os.Stat(cleanPath); err != nil {
		return "", fmt.Errorf("failed to get file info(%s): %w", cleanPath, err)
	}

	return cleanPath, nil
}
