package internal

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestScanFiles_Basic(t *testing.T) {
	root := t.TempDir()

	subdir := filepath.Join(root, "sub1")
	require.NoError(t, os.MkdirAll(subdir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(subdir, "data.db"), []byte("12345"), 0644))

	ScanFiles(root, []string{"**/*.db"}, nil, zap.NewNop())

	size := getMetricValue(t, "du_file_size_bytes", map[string]string{
		"path": filepath.Join(subdir, "data.db"),
	})
	assert.Equal(t, float64(5), size)
}

func TestScanFiles_NoGlobs(t *testing.T) {
	root := t.TempDir()

	subdir := filepath.Join(root, "sub1")
	require.NoError(t, os.MkdirAll(subdir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(subdir, "data.db"), []byte("12345"), 0644))

	ScanFiles(root, []string{}, nil, zap.NewNop())

	assert.False(t, metricExists(t, "du_file_size_bytes", map[string]string{
		"path": filepath.Join(subdir, "data.db"),
	}), "expected no file metrics when globs are empty")
}

func TestScanFiles_MultipleGlobs(t *testing.T) {
	root := t.TempDir()

	subdir := filepath.Join(root, "sub1")
	require.NoError(t, os.MkdirAll(subdir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(subdir, "data.db"), []byte("12345"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(subdir, "notes.txt"), []byte("hello world"), 0644))

	ScanFiles(root, []string{"**/*.db", "**/*.txt"}, nil, zap.NewNop())

	dbSize := getMetricValue(t, "du_file_size_bytes", map[string]string{
		"path": filepath.Join(subdir, "data.db"),
	})
	assert.Equal(t, float64(5), dbSize)

	txtSize := getMetricValue(t, "du_file_size_bytes", map[string]string{
		"path": filepath.Join(subdir, "notes.txt"),
	})
	assert.Equal(t, float64(11), txtSize)
}

func TestScanFiles_NonMatchingFiles(t *testing.T) {
	root := t.TempDir()

	subdir := filepath.Join(root, "sub1")
	require.NoError(t, os.MkdirAll(subdir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(subdir, "data.log"), []byte("log data"), 0644))

	ScanFiles(root, []string{"**/*.db"}, nil, zap.NewNop())

	assert.False(t, metricExists(t, "du_file_size_bytes", map[string]string{
		"path": filepath.Join(subdir, "data.log"),
	}), "expected non-matching file to have no metrics")
}

func TestScanFiles_ExcludedDirectories(t *testing.T) {
	root := t.TempDir()

	keepDir := filepath.Join(root, "keep")
	skipDir := filepath.Join(root, "skip")
	for _, d := range []string{keepDir, skipDir} {
		require.NoError(t, os.MkdirAll(d, 0755))
	}
	require.NoError(t, os.WriteFile(filepath.Join(keepDir, "a.db"), []byte("a"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(skipDir, "b.db"), []byte("b"), 0644))

	ScanFiles(root, []string{"**/*.db"}, []string{"skip"}, zap.NewNop())

	keepSize := getMetricValue(t, "du_file_size_bytes", map[string]string{
		"path": filepath.Join(keepDir, "a.db"),
	})
	assert.Equal(t, float64(1), keepSize)

	assert.False(t, metricExists(t, "du_file_size_bytes", map[string]string{
		"path": filepath.Join(skipDir, "b.db"),
	}), "expected excluded directory file to have no metrics")
}

func TestScanFiles_NestedFiles(t *testing.T) {
	root := t.TempDir()

	deep := filepath.Join(root, "level1", "level2")
	require.NoError(t, os.MkdirAll(deep, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(deep, "deep.db"), []byte("deep data"), 0644))

	ScanFiles(root, []string{"**/*.db"}, nil, zap.NewNop())

	size := getMetricValue(t, "du_file_size_bytes", map[string]string{
		"path": filepath.Join(deep, "deep.db"),
	})
	assert.Equal(t, float64(9), size)
}

func TestScanFiles_EmptyRoot(t *testing.T) {
	root := t.TempDir()

	ScanFiles(root, []string{"**/*.db"}, nil, zap.NewNop())

	metrics := metricExists(t, "du_file_size_bytes", map[string]string{})
	assert.False(t, metrics, "expected no file metrics for empty root")
}

func TestScanFiles_DirectoryNotMatched(t *testing.T) {
	root := t.TempDir()

	subdir := filepath.Join(root, "sub1.db")
	require.NoError(t, os.MkdirAll(subdir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(subdir, "file.txt"), []byte("x"), 0644))

	ScanFiles(root, []string{"**/*.db"}, nil, zap.NewNop())

	assert.False(t, metricExists(t, "du_file_size_bytes", map[string]string{
		"path": subdir,
	}), "expected directory named with matching extension to not be reported as a file")
}

func TestScanFiles_FileSizeZero(t *testing.T) {
	root := t.TempDir()

	subdir := filepath.Join(root, "sub1")
	require.NoError(t, os.MkdirAll(subdir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(subdir, "empty.db"), []byte{}, 0644))

	ScanFiles(root, []string{"**/*.db"}, nil, zap.NewNop())

	size := getMetricValue(t, "du_file_size_bytes", map[string]string{
		"path": filepath.Join(subdir, "empty.db"),
	})
	assert.Equal(t, float64(0), size)
}

func TestScanFiles_ScanErrorsOnUnreadableDir(t *testing.T) {
	root := t.TempDir()

	subdir := filepath.Join(root, "sub1")
	require.NoError(t, os.MkdirAll(subdir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(subdir, "data.db"), []byte("data"), 0644))

	require.NoError(t, os.Chmod(subdir, 0000))
	t.Cleanup(func() { _ = os.Chmod(subdir, 0755) })

	errorsBefore := getMetricValue(t, "du_scan_errors_total", map[string]string{})

	ScanFiles(root, []string{"**/*.db"}, nil, zap.NewNop())

	errorsAfter := getMetricValue(t, "du_scan_errors_total", map[string]string{})
	assert.GreaterOrEqual(t, errorsAfter-errorsBefore, float64(1))
}

