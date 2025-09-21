package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
)

var (
	rootPath    string
	intervalSec int
	port        int

	fileCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "du_subfolder_file_count",
			Help: "Number of files in a subfolder",
		},
		[]string{"folder"},
	)

	totalSize = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "du_subfolder_total_size_bytes",
			Help: "Total size of files in a subfolder (bytes)",
		},
		[]string{"folder"},
	)

	newestMTime = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "du_subfolder_newest_mtime_seconds",
			Help: "Modification time of the newest file in a subfolder (epoch seconds)",
		},
		[]string{"folder"},
	)

	oldestMTime = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "du_subfolder_oldest_mtime_seconds",
			Help: "Modification time of the oldest file in a subfolder (epoch seconds)",
		},
		[]string{"folder"},
	)

	scanDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "du_scan_duration_seconds",
			Help:    "Duration of the folder scan in seconds",
			Buckets: prometheus.DefBuckets, // [0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10]
		},
	)

	scanCount = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "du_scan_total",
			Help: "Total number of folder scans performed",
		},
	)

	diskTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "du_disk_total_bytes",
			Help: "Total bytes on the filesystem of a path",
		},
		[]string{"path"},
	)

	diskUsed = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "du_disk_used_bytes",
			Help: "Used bytes on the filesystem of a path",
		},
		[]string{"path"},
	)

	diskFreePercent = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "du_disk_free_percent",
			Help: "Percentage of free space on the filesystem of a path",
		},
		[]string{"path"},
	)
)

func init() {
	prometheus.MustRegister(
		fileCount, totalSize, newestMTime, oldestMTime,
		diskTotal, diskUsed, diskFreePercent,
		scanDuration, scanCount,
	)
}

func scanFolder(root string) {
	start := time.Now() // for scan duration

	entries, err := os.ReadDir(root)
	if err != nil {
		log.Printf("Error reading root folder %s: %v", root, err)
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			subfolder := filepath.Join(root, entry.Name())

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
				log.Printf("Error scanning %s: %v", subfolder, err)
				continue
			}

			fileCount.WithLabelValues(entry.Name()).Set(float64(count))
			totalSize.WithLabelValues(entry.Name()).Set(float64(size))
			if count > 0 {
				newestMTime.WithLabelValues(entry.Name()).Set(float64(newest))
				oldestMTime.WithLabelValues(entry.Name()).Set(float64(oldest))
			}
		}
	}

	scanDuration.Observe(time.Since(start).Seconds())
	scanCount.Inc()
}

func updateDiskMetrics(path string) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		log.Printf("Error getting disk stats for %s: %v", path, err)
		return
	}

	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bavail * uint64(stat.Bsize) // available to non-root
	used := total - free
	freePercent := (float64(free) / float64(total)) * 100

	diskTotal.WithLabelValues(path).Set(float64(total))
	diskUsed.WithLabelValues(path).Set(float64(used))
	diskFreePercent.WithLabelValues(path).Set(freePercent)
}

func runService(cmd *cobra.Command, args []string) {
	log.Printf("Starting service, watching root: %s (scanning every %d seconds)", rootPath, intervalSec)

	go func() {
		for {
			scanFolder(rootPath)
			updateDiskMetrics(rootPath)
			time.Sleep(time.Duration(intervalSec) * time.Second)
		}
	}()

	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, "OK\n")
	})

	// Expose metrics
	http.Handle("/metrics", promhttp.Handler())
	log.Println("Started HTTP server on http://localhost:8080")
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}

func main() {
	var rootCmd = &cobra.Command{
		Use:   "du-exporter",
		Short: "Expose Prometheus metrics for files in subfolders",
		Run:   runService,
	}

	rootCmd.Flags().StringVar(&rootPath, "root", "./watched", "Root folder to watch for files")
	rootCmd.Flags().IntVar(&intervalSec, "interval", 300, "Scan interval in seconds")
	rootCmd.Flags().IntVar(&port, "port", 8080, "Port to start the server on")

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
