# Project: du-exporter

## Overview

`du-exporter` is a Go program designed to mimic the functionality of the Unix `du` command, specifically for exporting Prometheus metrics related to subfolder disk usage. It scans a specified root directory and its subfolders, collecting metrics such as:

*   **File Count:** Number of files in each subfolder.
*   **Total Size:** Total size of files (in bytes) within each subfolder.
*   **Modification Times:** Newest and oldest modification times (epoch seconds) of files in each subfolder.
*   **Disk Usage:** Total, used, and free space (in bytes and percentage) on the filesystem of the root path.

These metrics are exposed via an HTTP server, making them easily consumable by Prometheus. The application uses `cobra` for command-line argument parsing.

## Technologies

*   **Language:** Go (version 1.26)
*   **Metrics:** Prometheus client library (`github.com/prometheus/client_golang`)
*   **CLI:** Cobra (`github.com/spf13/cobra`)
*   **Containerization:** Docker
*   **CI/CD:** GitHub Actions

## Building and Running

### Prerequisites

*   Go (version 1.26 or later)
*   Docker (optional, for containerized deployment)

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

*   `--root`: Specifies the root folder to watch for files (default: `./watched`).
*   `--interval`: Sets the scan interval in seconds (default: `300`).
*   `--port`: Sets the port for the HTTP server (default: `8080`).

The application will expose metrics at `http://localhost:<port>/metrics` and a health check endpoint at `http://localhost:<port>/healthz`.

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

*   Tests are run using `gotestsum`.
*   Coverage reports are generated (`profile.cov`) and uploaded to Coveralls.
*   JUnit XML reports are generated and published via GitHub Actions.

To run tests locally:

```bash
go install gotest.tools/gotestsum@latest
gotestsum --junitfile=./test-reports/junit.xml --format github-actions -- -v -coverprofile=profile.cov -coverpkg=./... ./...
```

### Linting

*   Code is linted using `golangci-lint`.

To run linting locally:

```bash
golangci-lint run
```

### CI/CD

*   **GitHub Actions:** The project uses GitHub Actions for automated workflows, defined in `.github/workflows/build.yml`.
    *   **`build-and-test` job:** Builds the Go application, runs tests, collects coverage, and performs linting.
    *   **`docker-publish` job:** Builds and publishes Docker images to `ghcr.io` on pushes to `main` branch and tags.

### Dockerfile

*   Uses a multi-stage build process for a small final image based on Alpine.
*   Runs as a non-root user (`appuser`).
*   Exposes port `8080`.
*   Includes a health check endpoint.
