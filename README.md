# du-exporter

## Overview

`du-exporter` is a Go program designed to mimic the functionality of the Unix `du` command, specifically for exporting Prometheus metrics related to subfolder disk usage. It scans a specified root directory and its subfolders, collecting metrics such as:

-   **File Count:** Number of files in each subfolder.
-   **Total Size:** Total size of files (in bytes) within each subfolder.
-   **Modification Times:** Newest and oldest modification times (epoch seconds) of files in each subfolder.
-   **Disk Usage:** Total, used, and free space (in bytes and percentage) on the filesystem of the root path.

These metrics are exposed via an HTTP server, making them easily consumable by Prometheus. The application uses `cobra` for command-line argument parsing.

## Technologies

-   **Language:** Go (version 1.25)
-   **Metrics:** Prometheus client library (`github.com/prometheus/client_golang`)
-   **CLI:** Cobra (`github.com/spf13/cobra`)
-   **Containerization:** Docker
-   **CI/CD:** GitHub Actions

## Building and Running

### Prerequisites

-   Go (version 1.25 or later)
-   Docker (optional, for containerized deployment)

### Build

To build the executable:

```bash
go build -ldflags="-w -s" -o du-exporter .
```

### Run (CLI)

The application can be run directly from the command line:

```bash
./du-exporter --root <path_to_watch> --interval <scan_interval_seconds> --port <server_port>
```

**Example:**

```bash
./du-exporter --root ./watched --interval 60 --port 8080
```

-   `--root`: Specifies the root folder to watch for files (default: `./watched`).
-   `--interval`: Sets the scan interval in seconds (default: `300`).
-   `--port`: Sets the port for the HTTP server (default: `8080`).

The application will expose metrics at `http://localhost:<port>/metrics` and a health check endpoint at `http://localhost:<port>/healthz`.

For example, the metrics show as follows (standard go metrics removed for brevity):

```prometheus
# HELP du_disk_free_percent Percentage of free space on the filesystem of a path
# TYPE du_disk_free_percent gauge
du_disk_free_percent{path="../map-services/data"} 24.00718717662695
# HELP du_disk_total_bytes Total bytes on the filesystem of a path
# TYPE du_disk_total_bytes gauge
du_disk_total_bytes{path="../map-services/data"} 4.94384795648e+11
# HELP du_disk_used_bytes Used bytes on the filesystem of a path
# TYPE du_disk_used_bytes gauge
du_disk_used_bytes{path="../map-services/data"} 3.75696912384e+11
# HELP du_scan_duration_seconds Duration of the folder scan in seconds
# TYPE du_scan_duration_seconds histogram
du_scan_duration_seconds_bucket{le="0.005"} 0
du_scan_duration_seconds_bucket{le="0.01"} 0
du_scan_duration_seconds_bucket{le="0.025"} 0
du_scan_duration_seconds_bucket{le="0.05"} 4
du_scan_duration_seconds_bucket{le="0.1"} 8
du_scan_duration_seconds_bucket{le="0.25"} 9
du_scan_duration_seconds_bucket{le="0.5"} 9
du_scan_duration_seconds_bucket{le="1"} 9
du_scan_duration_seconds_bucket{le="2.5"} 9
du_scan_duration_seconds_bucket{le="5"} 9
du_scan_duration_seconds_bucket{le="10"} 9
du_scan_duration_seconds_bucket{le="+Inf"} 9
du_scan_duration_seconds_sum 0.6477451670000001
du_scan_duration_seconds_count 9
# HELP du_scan_total Total number of folder scans performed
# TYPE du_scan_total counter
du_scan_total 9
# HELP du_subfolder_file_count Number of files in a subfolder
# TYPE du_subfolder_file_count gauge
du_subfolder_file_count{folder="company-data"} 1
du_subfolder_file_count{folder="geods-poi"} 1
du_subfolder_file_count{folder="mapproxy"} 11368
du_subfolder_file_count{folder="metoffice-datahub"} 1
du_subfolder_file_count{folder="street-manager-relay"} 1
# HELP du_subfolder_newest_mtime_seconds Modification time of the newest file in a subfolder (epoch seconds)
# TYPE du_subfolder_newest_mtime_seconds gauge
du_subfolder_newest_mtime_seconds{folder="company-data"} 1.758483112e+09
du_subfolder_newest_mtime_seconds{folder="geods-poi"} 1.758483112e+09
du_subfolder_newest_mtime_seconds{folder="mapproxy"} 1.758487709e+09
du_subfolder_newest_mtime_seconds{folder="metoffice-datahub"} 1.758483112e+09
du_subfolder_newest_mtime_seconds{folder="street-manager-relay"} 1.758483112e+09
# HELP du_subfolder_oldest_mtime_seconds Modification time of the oldest file in a subfolder (epoch seconds)
# TYPE du_subfolder_oldest_mtime_seconds gauge
du_subfolder_oldest_mtime_seconds{folder="company-data"} 1.758483112e+09
du_subfolder_oldest_mtime_seconds{folder="geods-poi"} 1.758483112e+09
du_subfolder_oldest_mtime_seconds{folder="mapproxy"} 1.756578834e+09
du_subfolder_oldest_mtime_seconds{folder="metoffice-datahub"} 1.758483112e+09
du_subfolder_oldest_mtime_seconds{folder="street-manager-relay"} 1.758483112e+09
# HELP du_subfolder_total_size_bytes Total size of files in a subfolder (bytes)
# TYPE du_subfolder_total_size_bytes gauge
du_subfolder_total_size_bytes{folder="company-data"} 0
du_subfolder_total_size_bytes{folder="geods-poi"} 0
du_subfolder_total_size_bytes{folder="mapproxy"} 2.53604661e+08
du_subfolder_total_size_bytes{folder="metoffice-datahub"} 0
du_subfolder_total_size_bytes{folder="street-manager-relay"} 0
```

### Run (Docker)

A Docker image can be built and run. The official image is published to `ghcr.io/rm-hull/du-exporter`.

**Build Docker Image:**

```bash
docker build -t du-exporter .
```

**Run Docker Container:**

```bash
docker run -p 8080:8080 -v /path/to/your/data:/app/watched du-exporter --root /app/watched
```

Adjust the volume mount (`-v`) to point to the directory you wish to monitor inside the container.

## Development Conventions

### Testing

-   Tests are run using `gotestsum`.
-   Coverage reports are generated (`profile.cov`) and uploaded to Coveralls.
-   JUnit XML reports are generated and published via GitHub Actions.

To run tests locally:

```bash
go install gotest.tools/gotestsum@latest
gotestsum --junitfile=./test-reports/junit.xml --format github-actions -- -v -coverprofile=profile.cov -coverpkg=./... ./...
```

### Linting

-   Code is linted using `golangci-lint`.

To run linting locally:

```bash
golangci-lint run
```

### CI/CD

-   **GitHub Actions:** The project uses GitHub Actions for automated workflows, defined in `.github/workflows/build.yml`.
    -   **`build-and-test` job:** Builds the Go application, runs tests, collects coverage, and performs linting.
    -   **`docker-publish` job:** Builds and publishes Docker images to `ghcr.io` on pushes to `main` branch and tags.

### Dockerfile

-   Uses a multi-stage build process for a small final image based on Alpine.
-   Runs as a non-root user (`appuser`).
-   Exposes port `8080`.
-   Includes a health check endpoint.
