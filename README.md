# grojectms

Task Manager API built with Go, Gin, and GORM (MySQL). It supports basic authentication, role-based access control (RBAC), and task workflows including approvals.

## What it can do (so far)

### Authentication

- User registration (`/register`)
- User login (`/login`) returning a JWT
- JWT-protected routes via `Authorization: Bearer <token>`

### RBAC (roles)

Roles are:

- `admin`
- `manager`
- `member`

Access rules implemented in the API:

- **Users management**
  - Only `admin` can list/update users.
- **Tasks**
  - `admin` / `manager` can create tasks.
  - `admin` sees all tasks.
  - `manager` sees tasks created by them, assigned to them, or assigned to people in their reporting hierarchy.
  - `member` sees tasks created by them or assigned to them.

### Tasks workflow

- Create / read / update / delete tasks (role restricted)
- Progress tracking with `progress_percentage` (0..100)
- Status transitions with validation
- Approval decisions:
  - Move a task to `pending_approval` when progress reaches 100%
  - Managers/Admins can `approve` or `reject`
- Audit trail entries are written for approvals/rejections

## Tech stack

- Go
- Gin
- GORM
- MySQL
- JWT authentication

## Setup & installation

### Prerequisites

- Go installed
- MySQL running
- A MySQL database created (for example `taskdbgo`)

### Environment variables

The server reads configuration from environment variables (you can use a local `.env`).

- `PORT`
- `JWT_SECRET`
- `DB_HOST`
- `DB_PORT`
- `DB_USER`
- `DB_PASSWORD`
- `DB_NAME`

See `.env.example`.

### Run locally

1. Copy `.env.example` to `.env` and fill values.
2. Start the server:

```bash
go run .
```

The server listens on `:${PORT}` (defaults to `8000`).

## API Documentation

See `docs/api.md` for detailed endpoint documentation (request/response payloads and error cases).

## Running tests

Tests are written as HTTP endpoint tests (Gin + `httptest`) and require a MySQL database.

1. Create a dedicated test database (recommended):
   - Example: `testdbgo`
2. Point your env vars to the test DB (same host/user/pass as your local DB is fine, but use a separate DB name).
3. Run:

```bash
go test ./...
```

Note: tests currently drop and recreate tables in the configured database.

## Roadmap / things to come

Planned next steps (not implemented yet):

- Finish and harden RBAC rules (more role validations and constraints)
- Add task deadline support
- Add task quality fields/ratings
- Introduce Projects:
  - A project can contain multiple tasks
  - Projects can have managers at different levels
  - More complex hierarchy and visibility rules
