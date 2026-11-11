# du-exporter — Agent Guidelines

## Purpose

`du-exporter` exposes Prometheus metrics about disk usage for subfolders under a root directory. It runs a periodic scan loop and serves metrics over HTTP.

## Architecture

The application is small and flat. Entry point `main.go` wires together CLI flags, logging, a background scan goroutine, and an HTTP server.

### File Map

| File | Role |
|---|---|
| `main.go` | CLI (cobra), logger init, HTTP server, scan loop |
| `internal/metrics.go` | Prometheus metric definitions and registration |
| `internal/folder_scanner.go` | Walks immediate subdirectories of root; collects file count, size, mtime |
| `internal/file_scanner.go` | Walks entire tree matching glob patterns; reports per-file size |
| `internal/disk_usage.go` | Reads filesystem stats via `syscall.Statfs`; reports total/used/free |

### Data Flow

1. `main.go` parses flags (`--root`, `--interval`, `--port`, `--glob`).
2. A goroutine calls `ScanFolder`, `UpdateDiskMetrics`, and `ScanFiles` on every tick.
3. Each scanner writes to Prometheus gauge/counter/histogram metrics defined in `internal/metrics.go`.
4. The HTTP server exposes `/metrics` (Prometheus handler) and `/healthz` (plain-text OK).

### Metrics

All metrics are prefixed with `du_`. They fall into three groups:

- **Subfolder metrics** (labeled by `folder`): `du_subfolder_file_count`, `du_subfolder_total_size_bytes`, `du_subfolder_newest_mtime_seconds`, `du_subfolder_oldest_mtime_seconds`
- **Disk metrics** (labeled by `path`): `du_disk_total_bytes`, `du_disk_used_bytes`, `du_disk_free_percent`, `du_disk_used_percent`
- **Scan metrics** (no labels): `du_scan_duration_seconds` (histogram), `du_scan_total` (counter), `du_scan_errors_total` (counter)
- **File metrics** (labeled by `path`): `du_file_size_bytes`

### Key Dependencies

- `github.com/prometheus/client_golang` — metric types and `/metrics` handler
- `github.com/spf13/cobra` — CLI flag parsing
- `go.uber.org/zap` — structured logging (production config)
- `github.com/bmatcuk/doublestar/v4` — glob pattern matching in `file_scanner.go`

## Code Conventions

- **No KDoc/comments** — the codebase does not use function-level documentation. Do not add comments unless the logic is non-obvious.
- **Package-level metrics** — all Prometheus metrics are package-level variables in `internal/metrics.go`, registered in `init()`. New metrics must follow this pattern.
- **Label naming** — use `folder` for subfolder names, `path` for filesystem paths.
- **Error handling** — log errors with `zap.Error(err)` and increment `scanErrors`. Do not panic or exit on scan errors.
- **Logger parameter** — every function in `internal/` accepts `*zap.Logger` as its last parameter. Pass it through; never create a new logger inside a function.
- **No tests exist** — the project currently has no `*_test.go` files. If adding tests, place them alongside the source file (e.g., `internal/folder_scanner_test.go`).

## Building and Running

```bash
go build -ldflags="-w -s" -o du-exporter .
./du-exporter --root ./watched --interval 60 --port 8080
```

Flags: `--root` (default `./watched`), `--interval` (default `300`), `--port` (default `8080`), `--glob` (repeatable, default `**/*.db`).

## Testing and Linting

Tests use `gotestsum`. Linting uses `golangci-lint`.

```bash
go install gotest.tools/gotestsum@latest
gotestsum -- -v -coverprofile=profile.cov -coverpkg=./... ./...

golangci-lint run
```

## Docker

Multi-stage Alpine build in `Dockerfile`. Runs as non-root `appuser`. Health check on `/healthz`. Published to `ghcr.io/rm-hull/du-exporter`.

## CI/CD

GitHub Actions (`.github/workflows/build.yml`):
- `build-and-test` — tests, coverage (Coveralls), lint (golangci-lint).
- `docker-publish` — builds and pushes Docker images on push to `main` and on tags.

## Common Tasks

- **Add a new metric**: Define it in `internal/metrics.go` and register it in `init()`. Then set it from the appropriate scanner.
- **Change scan behavior**: Modify the corresponding function in `internal/`. The scan loop in `main.go` calls all three scanners sequentially on each tick.
- **Add a CLI flag**: Declare the flag variable in `main()`, bind it with `rootCmd.Flags()`, and pass it to `runService()`.
- **Add tests**: Create `*_test.go` files in `internal/`. The project has no existing tests, so there are no test patterns to follow beyond standard Go testing conventions.
