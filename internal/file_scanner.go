package internal

import (
	"os"
	"path/filepath"

	"github.com/bmatcuk/doublestar/v4"
	"go.uber.org/zap"
)

func ScanFiles(root string, globs []string, logger *zap.Logger) {
	if len(globs) == 0 {
		if logger != nil {
			logger.Warn("ScanFiles called with no globs; nothing to match")
		}
		return
	}

	err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			logger.Error("error walking path", zap.String("path", path), zap.Error(walkErr))
			scanErrors.Inc()
			return nil
		}

		if info.IsDir() {
			return nil
		}

		// match against path relative to root when possible
		rel, relErr := filepath.Rel(root, path)
		var matchTarget string
		if relErr == nil {
			matchTarget = filepath.ToSlash(rel)
		} else {
			matchTarget = filepath.ToSlash(path)
		}

		matched := false
		for _, pat := range globs {
			ok, matchErr := doublestar.PathMatch(pat, matchTarget)
			if matchErr != nil {
				logger.Error("glob match error", zap.String("pattern", pat), zap.String("path", matchTarget), zap.Error(matchErr))
				scanErrors.Inc()
				continue
			}
			if ok {
				matched = true
				break
			}
		}

		if matched {
			fileSize.WithLabelValues(path).Set(float64(info.Size()))
		}

		return nil
	})

	if err != nil {
		logger.Error("error walking root folder", zap.String("root", root), zap.Error(err))
		scanErrors.Inc()
	}
}
