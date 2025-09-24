package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	rootPath    string
	intervalSec int
	port        int

	logger *zap.Logger
)

func updateDiskMetrics(path string) {
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
}

func runService(cmd *cobra.Command, args []string) {
	logger.Info("Starting service",
		zap.String("root", rootPath),
		zap.Int("interval", intervalSec),
	)

	go func() {
		scanFolder(rootPath)
		updateDiskMetrics(rootPath)

		ticker := time.NewTicker(time.Duration(intervalSec) * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			scanFolder(rootPath)
			updateDiskMetrics(rootPath)
		}
	}()

	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, "OK\n")
	})

	// Expose metrics
	http.Handle("/metrics", promhttp.Handler())
	logger.Info("Started HTTP server", zap.Int("port", port))
	logger.Fatal("HTTP server failed", zap.Error(http.ListenAndServe(fmt.Sprintf(":%d", port), nil)))
}

func main() {
	var err error
	logger, err = zap.NewProduction()
	if err != nil {
		fmt.Printf("can't initialize zap logger: %v", err)
		os.Exit(1)
	}
	defer func() {
		if err := logger.Sync(); err != nil {
			log.Print("Unable to sync zap logger")
		}
	}()

	var rootCmd = &cobra.Command{
		Use:   "du-exporter",
		Short: "Expose Prometheus metrics for files in subfolders",
		Run:   runService,
	}

	rootCmd.Flags().StringVar(&rootPath, "root", "./watched", "Root folder to watch for files")
	rootCmd.Flags().IntVar(&intervalSec, "interval", 300, "Scan interval in seconds")
	rootCmd.Flags().IntVar(&port, "port", 8080, "Port to start the server on")

	if err := rootCmd.Execute(); err != nil {
		logger.Fatal("Failed to execute command", zap.Error(err))
	}
}
