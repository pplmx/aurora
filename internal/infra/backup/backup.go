package backup

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
)

type BackupService struct {
	dbPaths map[string]string
}

type BackupResult struct {
	File     string
	Size     int64
	Checksum string
}

type BackupData struct {
	Version   string                 `json:"version"`
	Timestamp string                 `json:"timestamp"`
	Checksum  string                 `json:"checksum"`
	Data      map[string]interface{} `json:"data"`
}

func NewBackupService(dbPaths map[string]string) *BackupService {
	return &BackupService{dbPaths: dbPaths}
}

func (s *BackupService) Create(ctx context.Context, output string) (*BackupResult, error) {
	data := BackupData{
		Version:   "1.1",
		Timestamp: "2026-04-26T00:00:00Z",
		Data:      make(map[string]interface{}),
	}

	for name := range s.dbPaths {
		if data.Data[name] == nil {
			data.Data[name] = []interface{}{}
		}
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, err
	}

	checksum := fmt.Sprintf("%x", sha256.Sum256(jsonData))
	data.Checksum = checksum

	finalData, _ := json.MarshalIndent(data, "", "  ")
	if err := os.WriteFile(output, finalData, 0644); err != nil {
		return nil, err
	}

	fi, _ := os.Stat(output)
	return &BackupResult{
		File:     output,
		Size:     fi.Size(),
		Checksum: checksum,
	}, nil
}

func (s *BackupService) Verify(ctx context.Context, file string) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("cannot read file: %w", err)
	}

	var backup BackupData
	if err := json.Unmarshal(data, &backup); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	storedChecksum := backup.Checksum
	backup.Checksum = ""
	recomputed, _ := json.MarshalIndent(backup, "", "  ")
	computed := fmt.Sprintf("%x", sha256.Sum256(recomputed))

	if storedChecksum != computed {
		return fmt.Errorf("checksum mismatch")
	}

	return nil
}

func (s *BackupService) Restore(ctx context.Context, file string) error {
	return fmt.Errorf("restore not implemented - requires schema migration")
}