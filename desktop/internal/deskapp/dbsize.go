package deskapp

import (
	"os"

	"github.com/pplmx/aurora/internal/utils"
)

// DatabaseSizeText reports the ledger's disk usage the way `aurora lottery
// db-info` does: the database plus the -wal/-shm siblings that hold committed
// pages until a checkpoint. An empty path (a shell built before the database
// opened) reports nothing rather than a misleading zero.
func DatabaseSizeText(dbPath string) string {
	if dbPath == "" {
		return ""
	}
	var size int64
	for _, file := range append([]string{dbPath}, utils.SQLiteSidecarPaths(dbPath)...) {
		if info, err := os.Stat(file); err == nil {
			size += info.Size()
		}
	}
	return utils.FormatBytes(size)
}
