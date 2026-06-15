package middleware

import (
	"log"
	"os"
	"path/filepath"
	"time"
)

// StartCleanup goroutine periodically removes expired temp files.
func StartCleanup(tempDir string, intervalMin, expiryMin int) {
	go func() {
		ticker := time.NewTicker(time.Duration(intervalMin) * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			cleanup(tempDir, expiryMin)
		}
	}()
}

func cleanup(tempDir string, expiryMin int) {
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("[cleanup] read dir: %v", err)
		}
		return
	}

	cutoff := time.Now().Add(-time.Duration(expiryMin) * time.Minute)
	removed := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		path := filepath.Join(tempDir, e.Name())
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			if err := os.Remove(path); err == nil {
				removed++
			}
		}
	}
	if removed > 0 {
		log.Printf("[cleanup] removed %d expired files from %s", removed, tempDir)
	}
}
