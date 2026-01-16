# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Apotek HRIS is a pharmacy Human Resource Information System built with Go. The system manages employees, work logs, attendance tracking, and salary calculations with support for static and additional salary components.

The application uses:
- **Go 1.25.0** with standard library patterns
- **PostgreSQL** database with pgx driver (v5)
- **Chi router** for HTTP routing
- **Cobra** for CLI commands
- **Viper** for configuration management
- **golang-migrate** for database migrations
- **shopspring/decimal** for precise decimal arithmetic (salary calculations)
- **Asia/Jakarta timezone** set globally in main.go

## Common Commands

### Running the Server
```bash
go run . serve                                    # Run with default config files
go run . serve -c config/custom.yaml              # Run with custom config
```

### Database Migrations
```bash
go run . migrate up                               # Apply all pending migrations
go run . migrate down                             # Rollback last migration
go run . migrate create <name>                    # Create new migration files
```

### API Documentation
```bash
# Interactive API docs (Redocly)
open http://localhost:8080/docs                   # View interactive API docs in browser

# OpenAPI spec is served at /docs/openapi.yaml
curl http://localhost:8080/docs/openapi.yaml      # View OpenAPI spec
```

The OpenAPI specification (v3.1.0) is defined in `docs/openapi.yaml`.

When running in Docker, interactive API documentation is automatically generated using Redocly and served at `/docs`.

### Development
```bash
go build                                          # Build binary
go test ./...                                     # Run all tests
go mod tidy                                       # Clean up dependencies
```

### Docker
```bash
docker build -t apotek-hris .                     # Build image
docker run -p 8080:8080 apotek-hris               # Run container
```

## Version Control with Jujutsu

**IMPORTANT**: This project uses Jujutsu (jj) as the version control system, NOT git.

### Workflow Rules
1. **Always start from an empty change**: Use `jj new` before beginning work
2. **Always create an empty change after finishing**: Use `jj new` after committing to prepare for next work
3. **Never push unless explicitly requested**: Do not run `jj git push` unless the user asks
4. **Always add co-author in commit messages**: Every commit must end with `Co-Authored-By: Claude <model-name> <noreply@anthropic.com>` where `<model-name>` is the actual Claude model being used (e.g., "Claude Sonnet 4.5", "Claude Opus 4.5")
5. Use jj commands for all version control operations

### Common jj Commands
```bash
jj new                                            # Create new empty change to work in
jj status                                         # Show status of current change
jj diff                                           # Show diff of current change
jj describe -m "message"                          # Add description to current change
jj log                                            # View change history
jj squash                                         # Squash current change into parent
jj abandon                                        # Abandon current empty change
```

### Commit Message Format
Always include the co-author line at the end of commit messages with the actual model being used:

```bash
jj describe -m "$(cat <<'EOF'
feat: add new feature

Description of the changes made.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
EOF
)"
```

### Before Starting Any Work
Always create a new empty change first:
```bash
jj new
```

This ensures clean separation of changes and follows the project's workflow conventions.

## Architecture

### Project Structure

```
.
├── cmd/hris/           # CLI commands (serve, migrate)
├── internal/           # Domain modules (not importable by other projects)
│   ├── hris/          # Employee, WorkType, WorkLog management
│   ├── attendance/    # Attendance tracking and types
│   ├── salary/        # Salary calculation and components
│   └── config/        # Configuration loading
├── pkg/               # Reusable packages
│   ├── database/      # PostgreSQL connection setup
│   ├── server/        # HTTP server initialization and routing
│   ├── httpx/         # HTTP response helpers
│   ├── timex/         # Time utilities (Month type)
│   ├── validatorx/    # Validation helpers
│   └── slicex/        # Slice utilities
└── migrations/        # SQL migration files
```

### Domain Modules

Each domain module (hris, attendance, salary) follows a consistent layered architecture:

1. **router.go**: Defines HTTP routes using Chi router
2. **handler.go**: HTTP request/response handling, validation
3. **service.go**: Business logic layer
4. **db.go**: Database operations (queries, transactions)
5. **models.go**: Domain models and request/response types

**Dependency flow**: Handler → Service → DB

The `salary` module depends on both `hris` and `attendance` services for calculating total salary.

### Key Architectural Patterns

#### Configuration Management
- Configuration files: `config/config.yaml` and `config/secret.yaml`
- Multiple config files are merged via Viper's `MergeInConfig`
- Can be overridden with environment variables (automatic via Viper)
- Config loading happens in commands, not in main.go

#### Database Layer
- Uses `sqlx.DB` for enhanced SQL operations
- Connection pool managed via pgx/v5 with custom configuration
- Decimal types registered via `pgx-shopspring-decimal` for precise calculations
- Database logging enabled via `pgx-slog` tracer
- All queries use context for cancellation and timeouts

#### HTTP Server
- Server setup in `pkg/server/server.go`
- Middleware: CORS (allow all), RequestID, RealIP, Logger, Recoverer, Compress, Timeout (60s)
- All API routes under `/api/v1`
- Graceful shutdown with 15-second timeout
- Health check endpoint: `GET /health` (pings database)

#### Salary Calculation
The salary system has three types of components:

1. **Static Components**: Monthly recurring (e.g., base salary, allowances)
2. **Additional Components**: One-time per month (e.g., bonuses, deductions)
3. **Dynamic Components**: Calculated from work logs and attendance (e.g., shift fees, work outcomes)

Salary snapshots can be created to preserve historical salary data as JSON in the database.

#### Work Logs
- Work logs track employee work with multiple units (WorkLogUnit)
- Each unit has a WorkType with a multiplier for salary calculation
- Soft delete support: deleted_at and deleted_by fields
- Can generate patient-facing work log printouts

#### Attendance System
- Tracks daily attendance per employee
- Configurable attendance types (e.g., present, sick leave, annual leave)
- Used in salary calculation for shift fees

### Database Conventions

- All tables use `id` as primary key (BIGSERIAL)
- Timestamps: `created_at`, `updated_at` (automatically managed)
- Soft deletes: `deleted_at`, `deleted_by` (nullable)
- Foreign keys use `_id` suffix (e.g., `employee_id`)
- Month storage: Uses custom `timex.Month` type (YYYY-MM format)

### JSON Handling

- Uses `github.com/go-json-experiment/json` instead of standard library
- Custom MarshalJSON methods for computed fields (e.g., salary totals)
- JSON field names use camelCase (via struct tags)

### Validation

- Request validation via `go-playground/validator/v10`
- Custom decimal validators via `sblackstone/shopspring-decimal-validators`
- Validation performed in handlers before passing to services

## Development Notes

### When Adding New Endpoints

1. Define models and request/response types in `models.go`
2. Add database queries in `db.go`
3. Implement business logic in `service.go`
4. Add HTTP handlers in `handler.go`
5. Register routes in `router.go` using Chi's route groups
6. **Update `docs/openapi.yaml`** with the new endpoint specifications and schemas

### When Creating Migrations

```bash
go run . migrate create add_new_field_to_employees
```

This creates paired up/down migration files in the `migrations/` directory with timestamp prefixes.

### Decimal Handling

Always use `shopspring/decimal.Decimal` for monetary values and multipliers. Never use float64 for financial calculations.

```go
amount := decimal.NewFromFloat(1000.50)
result := amount.Mul(multiplier).RoundUp(0)  // Round to nearest integer
```

### Time and Locale

The application is hardcoded to use Asia/Jakarta timezone and Indonesian locale (id_ID). Time zone is set globally in `main.go` via `time.Local`.
