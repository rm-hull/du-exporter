package internal

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestUpdateDiskMetrics_ValidPath(t *testing.T) {
	root := t.TempDir()

	UpdateDiskMetrics(root, zap.NewNop())

	total := getMetricValue(t, "du_disk_total_bytes", map[string]string{
		"path": root,
	})
	assert.Greater(t, total, float64(0))

	used := getMetricValue(t, "du_disk_used_bytes", map[string]string{
		"path": root,
	})
	assert.GreaterOrEqual(t, used, float64(0))

	freePercent := getMetricValue(t, "du_disk_free_percent", map[string]string{
		"path": root,
	})
	assert.GreaterOrEqual(t, freePercent, float64(0))
	assert.LessOrEqual(t, freePercent, float64(100))

	usedPercent := getMetricValue(t, "du_disk_used_percent", map[string]string{
		"path": root,
	})
	assert.GreaterOrEqual(t, usedPercent, float64(0))
	assert.LessOrEqual(t, usedPercent, float64(100))

	sum := freePercent + usedPercent
	assert.InDelta(t, 100.0, sum, 0.01, "free and used percent should sum to 100")
}

func TestUpdateDiskMetrics_InvalidPath(t *testing.T) {
	errorsBefore := getMetricValue(t, "du_scan_errors_total", map[string]string{})

	UpdateDiskMetrics("/nonexistent/path/that/does/not/exist", zap.NewNop())

	errorsAfter := getMetricValue(t, "du_scan_errors_total", map[string]string{})
	assert.GreaterOrEqual(t, errorsAfter-errorsBefore, float64(1))
}

func TestUpdateDiskMetrics_MultiplePaths(t *testing.T) {
	root1 := t.TempDir()
	root2 := t.TempDir()

	UpdateDiskMetrics(root1, zap.NewNop())
	UpdateDiskMetrics(root2, zap.NewNop())

	total1 := getMetricValue(t, "du_disk_total_bytes", map[string]string{
		"path": root1,
	})
	assert.Greater(t, total1, float64(0))

	total2 := getMetricValue(t, "du_disk_total_bytes", map[string]string{
		"path": root2,
	})
	assert.Greater(t, total2, float64(0))
}

func TestUpdateDiskMetrics_SamePathOverwrite(t *testing.T) {
	root := t.TempDir()

	UpdateDiskMetrics(root, zap.NewNop())
	total1 := getMetricValue(t, "du_disk_total_bytes", map[string]string{
		"path": root,
	})

	UpdateDiskMetrics(root, zap.NewNop())
	total2 := getMetricValue(t, "du_disk_total_bytes", map[string]string{
		"path": root,
	})

	assert.Equal(t, total1, total2, "disk total should remain the same on repeated calls")
}

func TestUpdateDiskMetrics_CurrentDirectory(t *testing.T) {
	UpdateDiskMetrics(".", zap.NewNop())

	total := getMetricValue(t, "du_disk_total_bytes", map[string]string{
		"path": ".",
	})
	assert.Greater(t, total, float64(0))
}

func TestUpdateDiskMetrics_Subdirectory(t *testing.T) {
	root := t.TempDir()

	subdir := filepath.Join(root, "subdir")
	require.NoError(t, os.MkdirAll(subdir, 0755))

	UpdateDiskMetrics(subdir, zap.NewNop())

	total := getMetricValue(t, "du_disk_total_bytes", map[string]string{
		"path": subdir,
	})
	assert.Greater(t, total, float64(0))
}
