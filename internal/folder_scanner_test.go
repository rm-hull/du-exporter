package internal

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func getMetricValue(t *testing.T, metricName string, labels map[string]string) float64 {
	t.Helper()
	metrics, err := prometheus.DefaultGatherer.Gather()
	require.NoError(t, err, "Failed to gather metrics")

	for _, mf := range metrics {
		if mf.GetName() != metricName {
			continue
		}
		for _, m := range mf.Metric {
			if matchLabels(m, labels) {
				if m.Gauge != nil {
					return m.Gauge.GetValue()
				}
				if m.Counter != nil {
					return m.Counter.GetValue()
				}
			}
		}
	}
	t.Fatalf("Metric %q not found with labels %v", metricName, labels)
	return 0
}

func metricExists(t *testing.T, metricName string, labels map[string]string) bool {
	t.Helper()
	metrics, err := prometheus.DefaultGatherer.Gather()
	require.NoError(t, err, "Failed to gather metrics")

	for _, mf := range metrics {
		if mf.GetName() != metricName {
			continue
		}
		for _, m := range mf.Metric {
			if matchLabels(m, labels) {
				return true
			}
		}
	}
	return false
}

func matchLabels(metric *io_prometheus_client.Metric, expectedLabels map[string]string) bool {
	if len(metric.Label) < len(expectedLabels) {
		return false
	}
	for _, label := range metric.Label {
		expected, ok := expectedLabels[label.GetName()]
		if !ok {
			return false
		}
		if label.GetValue() != expected {
			return false
		}
	}
	return true
}

func TestScanFolder_Basic(t *testing.T) {
	root := t.TempDir()

	subdir := filepath.Join(root, "sub1")
	require.NoError(t, os.MkdirAll(subdir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(subdir, "file1.txt"), []byte("hello"), 0644))

	ScanFolder([]string{root}, 1, nil, zap.NewNop())

	count := getMetricValue(t, "du_subfolder_file_count", map[string]string{
		"root":   root,
		"folder": "sub1",
	})
	assert.Equal(t, float64(1), count)

	size := getMetricValue(t, "du_subfolder_total_size_bytes", map[string]string{
		"root":   root,
		"folder": "sub1",
	})
	assert.Equal(t, float64(5), size)
}

func TestScanFolder_ExcludedDirectories(t *testing.T) {
	root := t.TempDir()

	sub1 := filepath.Join(root, "sub1")
	sub2 := filepath.Join(root, "excluded")
	for _, d := range []string{sub1, sub2} {
		require.NoError(t, os.MkdirAll(d, 0755))
	}
	require.NoError(t, os.WriteFile(filepath.Join(sub1, "a.txt"), []byte("a"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(sub2, "b.txt"), []byte("b"), 0644))

	ScanFolder([]string{root}, 1, []string{"excluded"}, zap.NewNop())

	count := getMetricValue(t, "du_subfolder_file_count", map[string]string{
		"root":   root,
		"folder": "sub1",
	})
	assert.Equal(t, float64(1), count)

	assert.False(t, metricExists(t, "du_subfolder_file_count", map[string]string{
		"root":   root,
		"folder": "excluded",
	}), "expected excluded directory to have no metrics")
}

func TestScanFolder_DepthLimiting(t *testing.T) {
	root := t.TempDir()

	deep := filepath.Join(root, "level1", "level2")
	require.NoError(t, os.MkdirAll(deep, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(deep, "deep.txt"), []byte("deep"), 0644))

	ScanFolder([]string{root}, 1, nil, zap.NewNop())

	count := getMetricValue(t, "du_subfolder_file_count", map[string]string{
		"root":   root,
		"folder": "level1",
	})
	assert.Equal(t, float64(1), count)

	assert.False(t, metricExists(t, "du_folder_file_count", map[string]string{
		"root":   root,
		"folder": "level1/level2",
		"depth":  "2",
	}), "expected level2 folder metrics to be skipped when maxDepth=1")
}

func TestScanFolder_DepthLimiting_AllowsDeeper(t *testing.T) {
	root := t.TempDir()

	deep := filepath.Join(root, "level1", "level2")
	require.NoError(t, os.MkdirAll(deep, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(deep, "deep.txt"), []byte("deep"), 0644))

	ScanFolder([]string{root}, 2, nil, zap.NewNop())

	count := getMetricValue(t, "du_folder_file_count", map[string]string{
		"root":   root,
		"folder": "level1/level2",
		"depth":  "2",
	})
	assert.Equal(t, float64(1), count)
}

func TestScanFolder_UnreadableDirectory(t *testing.T) {
	root := t.TempDir()

	subdir := filepath.Join(root, "sub1")
	require.NoError(t, os.MkdirAll(subdir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(subdir, "file1.txt"), []byte("data"), 0644))

	require.NoError(t, os.Chmod(subdir, 0000))
	t.Cleanup(func() { _ = os.Chmod(subdir, 0755) })

	errorsBefore := getMetricValue(t, "du_scan_errors_total", map[string]string{})

	ScanFolder([]string{root}, 1, nil, zap.NewNop())

	errorsAfter := getMetricValue(t, "du_scan_errors_total", map[string]string{})
	assert.GreaterOrEqual(t, errorsAfter-errorsBefore, float64(1))
}

func TestScanFolder_MultipleRoots(t *testing.T) {
	root1 := t.TempDir()
	root2 := t.TempDir()

	sub1 := filepath.Join(root1, "sub1")
	sub2 := filepath.Join(root2, "sub2")
	for _, d := range []string{sub1, sub2} {
		require.NoError(t, os.MkdirAll(d, 0755))
	}
	require.NoError(t, os.WriteFile(filepath.Join(sub1, "a.txt"), []byte("a"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(sub2, "b.txt"), []byte("b"), 0644))

	ScanFolder([]string{root1, root2}, 1, nil, zap.NewNop())

	count1 := getMetricValue(t, "du_subfolder_file_count", map[string]string{
		"root":   root1,
		"folder": "sub1",
	})
	assert.Equal(t, float64(1), count1)

	count2 := getMetricValue(t, "du_subfolder_file_count", map[string]string{
		"root":   root2,
		"folder": "sub2",
	})
	assert.Equal(t, float64(1), count2)
}

func TestScanFolder_EmptySubfolder(t *testing.T) {
	root := t.TempDir()

	subdir := filepath.Join(root, "empty")
	require.NoError(t, os.MkdirAll(subdir, 0755))

	ScanFolder([]string{root}, 1, nil, zap.NewNop())

	count := getMetricValue(t, "du_subfolder_file_count", map[string]string{
		"root":   root,
		"folder": "empty",
	})
	assert.Equal(t, float64(0), count)

	size := getMetricValue(t, "du_subfolder_total_size_bytes", map[string]string{
		"root":   root,
		"folder": "empty",
	})
	assert.Equal(t, float64(0), size)
}

func TestScanFolder_MtimeMetrics(t *testing.T) {
	root := t.TempDir()

	subdir := filepath.Join(root, "sub1")
	require.NoError(t, os.MkdirAll(subdir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(subdir, "file1.txt"), []byte("data"), 0644))

	ScanFolder([]string{root}, 1, nil, zap.NewNop())

	newest := getMetricValue(t, "du_subfolder_newest_mtime_seconds", map[string]string{
		"root":   root,
		"folder": "sub1",
	})
	assert.NotZero(t, newest)

	oldest := getMetricValue(t, "du_subfolder_oldest_mtime_seconds", map[string]string{
		"root":   root,
		"folder": "sub1",
	})
	assert.NotZero(t, oldest)
}

func TestScanFolder_FolderSizeMetric(t *testing.T) {
	root := t.TempDir()

	subdir := filepath.Join(root, "sub1")
	require.NoError(t, os.MkdirAll(subdir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(subdir, "file1.txt"), []byte("12345"), 0644))

	ScanFolder([]string{root}, 1, nil, zap.NewNop())

	size := getMetricValue(t, "du_folder_size_bytes", map[string]string{
		"folder":      "sub1",
		"parent_path": root,
		"depth":       "1",
	})
	assert.Equal(t, float64(5), size)
}

func TestScanFolder_ExcludedNestedDirectories(t *testing.T) {
	root := t.TempDir()

	parent := filepath.Join(root, "parent")
	child := filepath.Join(parent, "skipme")
	other := filepath.Join(parent, "keep")
	for _, d := range []string{parent, child, other} {
		require.NoError(t, os.MkdirAll(d, 0755))
	}
	require.NoError(t, os.WriteFile(filepath.Join(child, "ignored.txt"), []byte("x"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(other, "kept.txt"), []byte("y"), 0644))

	ScanFolder([]string{root}, 2, []string{"skipme"}, zap.NewNop())

	count := getMetricValue(t, "du_folder_file_count", map[string]string{
		"root":   root,
		"folder": "parent",
		"depth":  "1",
	})
	assert.Equal(t, float64(1), count)
}

func TestScanFolder_ScanCountIncremented(t *testing.T) {
	root := t.TempDir()
	subdir := filepath.Join(root, "sub1")
	require.NoError(t, os.MkdirAll(subdir, 0755))

	countBefore := getMetricValue(t, "du_scan_total", map[string]string{})

	ScanFolder([]string{root}, 1, nil, zap.NewNop())

	countAfter := getMetricValue(t, "du_scan_total", map[string]string{})
	assert.Equal(t, float64(1), countAfter-countBefore)

	ScanFolder([]string{root}, 1, nil, zap.NewNop())

	countAfter2 := getMetricValue(t, "du_scan_total", map[string]string{})
	assert.Equal(t, float64(2), countAfter2-countBefore)
}

func TestScanFolder_LabelCleanup(t *testing.T) {
	root := t.TempDir()

	sub1 := filepath.Join(root, "sub1")
	sub2 := filepath.Join(root, "sub2")
	for _, d := range []string{sub1, sub2} {
		require.NoError(t, os.MkdirAll(d, 0755))
	}
	require.NoError(t, os.WriteFile(filepath.Join(sub1, "a.txt"), []byte("a"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(sub2, "b.txt"), []byte("b"), 0644))

	ScanFolder([]string{root}, 1, nil, zap.NewNop())

	require.NoError(t, os.RemoveAll(sub2))

	ScanFolder([]string{root}, 1, nil, zap.NewNop())

	assert.False(t, metricExists(t, "du_subfolder_file_count", map[string]string{
		"root":   root,
		"folder": "sub2",
	}), "expected sub2 metrics to be cleaned up after folder removal")
}

func TestIsExcluded(t *testing.T) {
	tests := []struct {
		dirName     string
		excludeList []string
		want        bool
	}{
		{"data", []string{}, false},
		{"backups", []string{"backups", "temp"}, true},
		{"temp", []string{"backups", "temp"}, true},
		{"data", []string{"backups", "temp"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.dirName, func(t *testing.T) {
			assert.Equal(t, tt.want, isExcluded(tt.dirName, tt.excludeList))
		})
	}
}
