package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rm-hull/du-exporter/internal"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func runService(rootPath string, fileGlobs []string, intervalSec int, port int, logger *zap.Logger) {
	logger.Info("Starting service",
		zap.String("root", rootPath),
		zap.Int("interval", intervalSec),
	)

	go func() {
		internal.ScanFolder(rootPath, logger)
		internal.UpdateDiskMetrics(rootPath, logger)
		internal.ScanFiles(rootPath, fileGlobs, logger)

		ticker := time.NewTicker(time.Duration(intervalSec) * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			internal.ScanFolder(rootPath, logger)
			internal.UpdateDiskMetrics(rootPath, logger)
			internal.ScanFiles(rootPath, fileGlobs, logger)
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
	var rootPath string
	var intervalSec int
	var port int
	var fileGlobs []string

	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "can't initialize zap logger: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		if err := logger.Sync(); err != nil {
			log.Printf("Unable to sync zap logger: %v", err)
		}
	}()

	var rootCmd = &cobra.Command{
		Use:   "du-exporter",
		Short: "Expose Prometheus metrics for files in subfolders",
		Run: func(cmd *cobra.Command, args []string) {
			runService(rootPath, fileGlobs, intervalSec, port, logger)
		},
	}

	rootCmd.Flags().StringVar(&rootPath, "root", "./watched", "Root folder to watch for files")
	rootCmd.Flags().IntVar(&intervalSec, "interval", 300, "Scan interval in seconds")
	rootCmd.Flags().IntVar(&port, "port", 8080, "Port to start the server on")
	rootCmd.Flags().StringArrayVar(&fileGlobs, "glob", []string{"**/*.db"}, "File glob patterns to include (can specify multiple)")

	if err := rootCmd.Execute(); err != nil {
		logger.Fatal("Failed to execute command", zap.Error(err))
	}
}
