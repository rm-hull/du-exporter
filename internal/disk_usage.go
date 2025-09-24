package internal

import (
	"syscall"

	"github.com/rm-hull/du-exporter/internal/metrics"
	"go.uber.org/zap"
)

func UpdateDiskMetrics(path string, logger *zap.Logger) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		logger.Error("Error getting disk stats", zap.String("path", path), zap.Error(err))
		metrics.ScanErrors.Inc()
		return
	}

	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bavail * uint64(stat.Bsize) // available to non-root
	used := total - free
	freePercent := (float64(free) / float64(total)) * 100

	metrics.DiskTotal.WithLabelValues(path).Set(float64(total))
	metrics.DiskUsed.WithLabelValues(path).Set(float64(used))
	metrics.DiskFreePercent.WithLabelValues(path).Set(freePercent)
}
