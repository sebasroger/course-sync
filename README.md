# Course Sync

A Go application for synchronizing course data from multiple learning platforms (Udemy, Pluralsight) and exporting it to Eightfold in various formats.

## Overview

Course Sync is a tool designed to:

1. Fetch course data from multiple learning platforms (currently Udemy and Pluralsight)
2. Normalize the data into a unified format
3. Export the data to Eightfold in various formats (CSV, XML)
4. Optionally upload the exported files via SFTP

## Project Structure

```
course-sync/
├── cmd/                    # Command-line applications
│   ├── exportcsv/          # Export courses to CSV format
│   ├── exportempxml/       # Export employee data to XML
│   ├── exportxml/          # Export courses to XML format
│   └── sync/               # Main sync utility
├── internal/               # Internal packages
│   ├── config/             # Configuration loading
│   ├── devutil/            # Development utilities
│   ├── domain/             # Domain models
│   ├── export/             # Export functionality
│   ├── httpx/              # HTTP utilities
│   ├── mappers/            # Data mappers
│   ├── providers/          # Course providers
│   │   ├── eightfold/      # Eightfold integration
│   │   ├── pluralsight/    # Pluralsight API client
│   │   └── udemy/          # Udemy API client
│   └── sftpclient/         # SFTP upload functionality
```

## Commands

### Sync

The main sync command fetches course data from providers and displays it.

```bash
go run ./cmd/sync/main.go
```

### Export CSV

Exports courses from all providers to a CSV file compatible with Eightfold.

```bash
go run ./cmd/exportcsv/main.go [options]
```

Options:
- `-out`: Output CSV path (default: "COURSE-MAIN_ALL.csv")
- `-udemy-max-pages`: Max pages to fetch from Udemy (0 = all)
- `-ps-max-pages`: Max pages to fetch from Pluralsight (0 = all)
- `-page-size`: Page size for providers (default: 100)
- `-sftp`: Upload the generated CSV via SFTP

### Export XML

Exports courses to XML format.

```bash
go run ./cmd/exportxml/main.go [options]
```

### Export Employee XML

Exports employee data to XML format.

```bash
go run ./cmd/exportempxml/main.go [options]
```

## Configuration

The application uses environment variables for configuration:

### General Configuration
- `EIGHTFOLD_BASE_URL`: Eightfold API base URL
- `EIGHTFOLD_BASIC_AUTH`: Basic auth token for Eightfold
- `EIGHTFOLD_USERNAME`: Eightfold username
- `EIGHTFOLD_PASSWORD`: Eightfold password

### Udemy Configuration
- `UDEMY_BASE_URL`: Udemy API base URL
- `UDEMY_CLIENT_ID`: Udemy client ID
- `UDEMY_CLIENT_SECRET`: Udemy client secret

### Pluralsight Configuration
- `PLURALSIGHT_BASE_URL`: Pluralsight API base URL
- `PLURALSIGHT_TOKEN`: Pluralsight API token

### SFTP Configuration
- `SFTP_HOST`: SFTP server hostname
- `SFTP_PORT`: SFTP server port
- `SFTP_USER`: SFTP username
- `SFTP_PASS`: SFTP password
- `SFTP_DIR`: Remote directory for uploads
- `SFTP_INSECURE_IGNORE_HOST_KEY`: Set to true to ignore host key verification (not recommended for production)

## Data Model

The application uses a unified course model (`UnifiedCourse`) to normalize data from different providers:

```go
type UnifiedCourse struct {
    Source        string   // "pluralsight", "udemy", etc.
    SourceID      string   // provider course id
    Title         string
    Description   string
    CourseURL     string
    Language      string
    Category      string
    Difficulty    string
    DurationHours float64
    Status        string   // "active", "inactive", etc.
    PublishedDate string   // ISO string if available
    ImageURL      string
    Skills        []string
}
```

## Dependencies

- Go 1.25.5
- github.com/pkg/sftp v1.13.10
- golang.org/x/crypto v0.46.0
- github.com/andybalholm/brotli v1.2.0

## Development

### Building

```bash
go build -o course-sync ./cmd/sync/main.go
```

### Testing

```bash
go test ./...
```

## License

Proprietary - All rights reserved
