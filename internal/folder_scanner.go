package internal

import (
	"os"
	"path/filepath"
	"time"

	"github.com/rm-hull/du-exporter/internal/metrics"
	"go.uber.org/zap"
)

func ScanFolder(root string, logger *zap.Logger) {
	metrics.FileCount.Reset()
	metrics.TotalSize.Reset()
	metrics.NewestMTime.Reset()
	metrics.OldestMTime.Reset()

	start := time.Now() // for scan duration

	entries, err := os.ReadDir(root)
	if err != nil {
		logger.Error("Error reading root folder", zap.String("root", root), zap.Error(err))
		metrics.ScanErrors.Inc()
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			subfolder := filepath.Join(root, entry.Name())
			scanSubfolder(subfolder, entry, logger)
		}
	}

	metrics.ScanDuration.Observe(time.Since(start).Seconds())
	metrics.ScanCount.Inc()
}

func scanSubfolder(subfolder string, entry os.DirEntry, logger *zap.Logger) {
	var count int
	var size int64
	var newest, oldest int64

	err := filepath.Walk(subfolder, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			count++
			size += info.Size()
			mtime := info.ModTime().Unix()
			if newest == 0 || mtime > newest {
				newest = mtime
			}
			if oldest == 0 || mtime < oldest {
				oldest = mtime
			}
		}
		return nil
	})

	if err != nil {
		logger.Error("Error scanning subfolder", zap.String("subfolder", subfolder), zap.Error(err))
		metrics.ScanErrors.Inc()
		return
	}

	metrics.FileCount.WithLabelValues(entry.Name()).Set(float64(count))
	metrics.TotalSize.WithLabelValues(entry.Name()).Set(float64(size))
	if count > 0 {
		metrics.NewestMTime.WithLabelValues(entry.Name()).Set(float64(newest))
		metrics.OldestMTime.WithLabelValues(entry.Name()).Set(float64(oldest))
	}
}
