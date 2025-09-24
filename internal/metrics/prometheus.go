package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	FileCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "du_subfolder_file_count",
			Help: "Number of files in a subfolder",
		},
		[]string{"folder"},
	)

	TotalSize = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "du_subfolder_total_size_bytes",
			Help: "Total size of files in a subfolder (bytes)",
		},
		[]string{"folder"},
	)

	NewestMTime = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "du_subfolder_newest_mtime_seconds",
			Help: "Modification time of the newest file in a subfolder (epoch seconds)",
		},
		[]string{"folder"},
	)

	OldestMTime = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "du_subfolder_oldest_mtime_seconds",
			Help: "Modification time of the oldest file in a subfolder (epoch seconds)",
		},
		[]string{"folder"},
	)

	ScanDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "du_scan_duration_seconds",
			Help:    "Duration of the folder scan in seconds",
			Buckets: prometheus.DefBuckets, // [0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10]
		},
	)

	ScanCount = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "du_scan_total",
			Help: "Total number of folder scans performed",
		},
	)

	ScanErrors = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "du_scan_errors_total",
			Help: "Total number of folder scan errors",
		},
	)

	DiskTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "du_disk_total_bytes",
			Help: "Total bytes on the filesystem of a path",
		},
		[]string{"path"},
	)

	DiskUsed = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "du_disk_used_bytes",
			Help: "Used bytes on the filesystem of a path",
		},
		[]string{"path"},
	)

	DiskFreePercent = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "du_disk_free_percent",
			Help: "Percentage of free space on the filesystem of a path",
		},
		[]string{"path"},
	)
)

func init() {
	prometheus.MustRegister(
		FileCount, TotalSize, NewestMTime, OldestMTime,
		DiskTotal, DiskUsed, DiskFreePercent,
		ScanDuration, ScanCount, ScanErrors,
	)
}
