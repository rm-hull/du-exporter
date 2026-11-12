package internal

import (
	"os"
	"path/filepath"
	"strconv"
	"time"

	"go.uber.org/zap"
)

func isExcluded(name string, excludeList []string) bool {
	for _, excluded := range excludeList {
		if name == excluded {
			return true
		}
	}
	return false
}

func ScanFolder(roots []string, maxDepth int, excludeList []string, logger *zap.Logger) {
	start := time.Now()

	for _, root := range roots {
		scanRoot(root, maxDepth, excludeList, logger)
	}

	scanDuration.Observe(time.Since(start).Seconds())
	scanCount.Inc()
}

func scanRoot(root string, maxDepth int, excludeList []string, logger *zap.Logger) {
	removeRootLabels(root)

	entries, err := os.ReadDir(root)
	if err != nil {
		logger.Error("Error reading root folder", zap.String("root", root), zap.Error(err))
		scanErrors.Inc()
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			if isExcluded(entry.Name(), excludeList) {
				continue
			}
			subfolder := filepath.Join(root, entry.Name())
			scanSubfolder(root, subfolder, entry.Name(), 1, maxDepth, excludeList, logger)
		}
	}
}

func removeRootLabels(root string) {
	labels := map[string]string{"root": root}
	folderFileCount.DeletePartialMatch(labels)
	folderTotalSize.DeletePartialMatch(labels)
	folderNewestMTime.DeletePartialMatch(labels)
	folderOldestMTime.DeletePartialMatch(labels)
	folderSize.DeletePartialMatch(map[string]string{"parent_path": root})
}

func scanSubfolder(root, fullPath, relPath string, depth int, maxDepth int, excludeList []string, logger *zap.Logger) (int, int64, int64, int64) {
	var count int
	var size int64
	var newest, oldest int64

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		logger.Error("Error reading directory", zap.String("dir", fullPath), zap.Error(err))
		scanErrors.Inc()
		return 0, 0, 0, 0
	}

	for _, entry := range entries {
		if entry.IsDir() {
			if isExcluded(entry.Name(), excludeList) {
				continue
			}
			subRelPath := filepath.Join(relPath, entry.Name())
			subFullPath := filepath.Join(fullPath, entry.Name())
			subCount, subSize, subNewest, subOldest := scanSubfolder(root, subFullPath, subRelPath, depth+1, maxDepth, excludeList, logger)
			count += subCount
			size += subSize
			if subNewest > 0 && (newest == 0 || subNewest > newest) {
				newest = subNewest
			}
			if subOldest > 0 && (oldest == 0 || subOldest < oldest) {
				oldest = subOldest
			}
			continue
		}

		info, err := entry.Info()
		if err != nil {
			logger.Error("Error reading file info", zap.String("path", filepath.Join(fullPath, entry.Name())), zap.Error(err))
			scanErrors.Inc()
			continue
		}

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

	if depth <= maxDepth {
		depthStr := strconv.Itoa(depth)
		folderFileCount.WithLabelValues(root, relPath, depthStr).Set(float64(count))
		folderTotalSize.WithLabelValues(root, relPath, depthStr).Set(float64(size))
		if count > 0 {
			folderNewestMTime.WithLabelValues(root, relPath, depthStr).Set(float64(newest))
			folderOldestMTime.WithLabelValues(root, relPath, depthStr).Set(float64(oldest))
		}
		folderSize.WithLabelValues(relPath, root, depthStr).Set(float64(size))
	}

	return count, size, newest, oldest
}
