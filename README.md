# Apotek HRIS

A Human Resource Information System designed for pharmacy operations, built with Go and PostgreSQL.

## Features

- **Employee Management**: Track employee information, shift fees, and attendance visibility
- **Work Log Tracking**: Record detailed work logs with multiple work types and patient information
- **Attendance System**: Monitor daily attendance with configurable attendance types
- **Salary Calculation**: Comprehensive salary management with three component types:
  - Static components (recurring monthly)
  - Additional components (one-time per month)
  - Dynamic components (calculated from work logs and attendance)
- **Salary Snapshots**: Preserve historical salary data for record-keeping
- **RESTful API**: Clean HTTP API with JSON responses

## Prerequisites

- Go 1.25.0 or higher
- PostgreSQL 12 or higher
- Jujutsu (jj) for version control

## Installation

1. Clone the repository:

```bash
jj git clone <repository-url>
cd apotek-hris
```

2. Install dependencies:

```bash
go mod download
```

3. Set up configuration files:

```bash
cp config/config.example.yaml config/config.yaml
cp config/secret.example.yaml config/secret.yaml
```

4. Edit configuration files with your database credentials and server settings.

## Configuration

Configuration is split across two files:

**config/config.yaml** - General configuration:

```yaml
database:
  host: localhost
  port: 5432
  db_name: apotek_hris
  ssl_mode: disable

server:
  port: 8080
  host: 0.0.0.0
```

**config/secret.yaml** - Sensitive credentials:

```yaml
database:
  user: your_db_user
  password: your_db_password
```

Configuration can be overridden using environment variables.

## Database Setup

1. Create a PostgreSQL database:

```sql
CREATE DATABASE apotek_hris;
```

2. Run migrations:

```bash
go run . migrate up
```

## Running the Application

Start the server:

```bash
go run . serve
```

The API will be available at `http://localhost:8080`

Health check endpoint:

```bash
curl http://localhost:8080/health
```

## API Documentation

### Interactive Documentation

When running with Docker, interactive API documentation is automatically generated using Redocly:

- **Interactive Docs**: `GET /docs` - Beautiful, interactive API documentation

Open in your browser:

```bash
open http://localhost:8080/docs
```

### OpenAPI Specification

The complete API documentation is available as an OpenAPI 3.1.0 specification:

- **OpenAPI Spec**: `GET /docs/openapi.yaml`
- **Source File**: `openapi.yaml` (project root)

View the spec:

```bash
curl http://localhost:8080/docs/openapi.yaml
```

You can also use tools like [Swagger Editor](https://editor.swagger.io/) or [Redocly](https://redocly.github.io/redoc/) to visualize the OpenAPI specification.

## API Endpoints

All API endpoints are prefixed with `/api/v1`.

### Employees

- `GET /api/v1/employees` - List all employees
- `POST /api/v1/employees` - Create new employee

### Work Types

- `GET /api/v1/work-types` - List all work types
- `POST /api/v1/work-types` - Create new work type

### Work Logs

- `GET /api/v1/work-logs` - List work logs
- `POST /api/v1/work-logs` - Create new work log
- `GET /api/v1/work-logs/{id}/for-patient` - Print work log for patient
- `DELETE /api/v1/work-logs/{id}` - Soft delete work log

### Attendance

- `GET /api/v1/attendances` - Get attendances between dates
- `GET /api/v1/attendances/types` - List attendance types
- `POST /api/v1/attendances/types` - Create attendance type
- `PUT /api/v1/attendances/{employeeID}/{date}` - Upsert attendance

### Salary

- `GET /api/v1/salary/{employeeID}/static-components` - Get employee static components
- `POST /api/v1/salary/{employeeID}/static-components` - Create static component
- `DELETE /api/v1/salary/{employeeID}/static-components/{id}` - Delete static component
- `GET /api/v1/salary/{month}/{employeeID}/additional-components` - Get additional components
- `POST /api/v1/salary/{month}/{employeeID}/additional-components` - Create additional component
- `POST /api/v1/salary/additional-components/bulk` - Bulk create additional components for multiple employees
- `DELETE /api/v1/salary/{month}/{employeeID}/additional-components/{id}` - Delete additional component
- `GET /api/v1/salary/{month}/{employeeID}/extra-infos` - Get extra infos
- `POST /api/v1/salary/{month}/{employeeID}/extra-infos` - Create extra info
- `DELETE /api/v1/salary/{month}/{employeeID}/extra-infos/{id}` - Delete extra info
- `GET /api/v1/salary/{month}/{employeeID}` - Calculate salary for employee and month
- `GET /api/v1/salary/snapshots` - List salary snapshots
- `POST /api/v1/salary/snapshots` - Create salary snapshot
- `GET /api/v1/salary/snapshots/{id}` - Get salary snapshot
- `DELETE /api/v1/salary/snapshots/{id}` - Delete salary snapshot

## Development

### Version Control

This project uses Jujutsu (jj) for version control. Always start work from an empty change:

```bash
jj new                    # Create new empty change
# Make your changes
jj describe -m "message"  # Describe your change
jj new                    # Create new empty change for next work
```

### Running Tests

```bash
go test ./...
```

### Creating Database Migrations

```bash
go run . migrate create add_new_feature
```

This creates a pair of migration files (up and down) in the `migrations/` directory.

### Building

Build the binary:

```bash
go build -o apotek-hris
```

Run the binary:

```bash
./apotek-hris serve
```

### Docker

Build the Docker image:

```bash
docker build -t apotek-hris .
```

Run with Docker:

```bash
docker run -p 8080:8080 apotek-hris
```

## Technology Stack

- **Language**: Go 1.25.0
- **Database**: PostgreSQL with pgx/v5 driver
- **HTTP Router**: Chi (go-chi/chi/v5)
- **CLI Framework**: Cobra
- **Configuration**: Viper
- **Database Migrations**: golang-migrate
- **Decimal Arithmetic**: shopspring/decimal (for precise financial calculations)
- **Validation**: go-playground/validator

## Project Structure

```
.
├── cmd/hris/           # CLI commands
├── internal/           # Domain modules
│   ├── hris/          # Employee and work log management
│   ├── attendance/    # Attendance tracking
│   ├── salary/        # Salary calculation
│   └── config/        # Configuration loading
├── pkg/               # Reusable packages
│   ├── database/      # Database connection
│   ├── server/        # HTTP server
│   ├── httpx/         # HTTP helpers
│   └── timex/         # Time utilities
├── migrations/        # SQL migrations
└── config/           # Configuration files
```

## Timezone and Locale

The application is configured for Indonesian pharmacy operations:

- **Timezone**: Asia/Jakarta
- **Locale**: Indonesian (id_ID)

## License

This project is licensed under the GNU GPLv3 License - see the [LICENSE](LICENSE) file for details.
