package internal

import (
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
)

func ScanFolder(roots []string, logger *zap.Logger) {
	start := time.Now()

	for _, root := range roots {
		scanRoot(root, logger)
	}

	scanDuration.Observe(time.Since(start).Seconds())
	scanCount.Inc()
}

func scanRoot(root string, logger *zap.Logger) {
	removeRootLabels(root)

	entries, err := os.ReadDir(root)
	if err != nil {
		logger.Error("Error reading root folder", zap.String("root", root), zap.Error(err))
		scanErrors.Inc()
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			subfolder := filepath.Join(root, entry.Name())
			scanSubfolder(root, subfolder, entry, logger)
		}
	}
}

func removeRootLabels(root string) {
	labels := map[string]string{"root": root}
	fileCount.DeletePartialMatch(labels)
	totalSize.DeletePartialMatch(labels)
	newestMTime.DeletePartialMatch(labels)
	oldestMTime.DeletePartialMatch(labels)
}

func scanSubfolder(root, subfolder string, entry os.DirEntry, logger *zap.Logger) {
	var count int
	var size int64
	var newest, oldest int64

	walkDir(subfolder, &count, &size, &newest, &oldest, logger)

	fileCount.WithLabelValues(root, entry.Name()).Set(float64(count))
	totalSize.WithLabelValues(root, entry.Name()).Set(float64(size))
	if count > 0 {
		newestMTime.WithLabelValues(root, entry.Name()).Set(float64(newest))
		oldestMTime.WithLabelValues(root, entry.Name()).Set(float64(oldest))
	}
}

func walkDir(dir string, count *int, size *int64, newest *int64, oldest *int64, logger *zap.Logger) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		logger.Error("Error reading directory", zap.String("dir", dir), zap.Error(err))
		scanErrors.Inc()
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			walkDir(filepath.Join(dir, entry.Name()), count, size, newest, oldest, logger)
			continue
		}

		info, err := entry.Info()
		if err != nil {
			logger.Error("Error reading file info", zap.String("path", filepath.Join(dir, entry.Name())), zap.Error(err))
			scanErrors.Inc()
			continue
		}

		*count++
		*size += info.Size()
		mtime := info.ModTime().Unix()
		if *newest == 0 || mtime > *newest {
			*newest = mtime
		}
		if *oldest == 0 || mtime < *oldest {
			*oldest = mtime
		}
	}
}
