package internal

import (
	"syscall"

	"go.uber.org/zap"
)

func UpdateDiskMetrics(path string, logger *zap.Logger) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		logger.Error("Error getting disk stats", zap.String("path", path), zap.Error(err))
		scanErrors.Inc()
		return
	}

	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bavail * uint64(stat.Bsize) // available to non-root
	used := total - free
	freePercent := (float64(free) / float64(total)) * 100

	diskTotal.WithLabelValues(path).Set(float64(total))
	diskUsed.WithLabelValues(path).Set(float64(used))
	diskFreePercent.WithLabelValues(path).Set(freePercent)
	diskUsedPercent.WithLabelValues(path).Set(100.0 - freePercent)
}
