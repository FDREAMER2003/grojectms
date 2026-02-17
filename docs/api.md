# API Documentation

## Base URL

All endpoints are served from the same host/port your server runs on.

- Default: `http://localhost:8000`

## Authentication

### JWT

Most endpoints require a JWT access token.

- Send it via header:
  - `Authorization: Bearer <token>`

### Roles

Some endpoints are restricted by role.

- `admin`
- `manager`
- `member`

## Common responses

### Error format

Errors are returned as JSON:

```json
{
  "error": "message"
}
```

### Auth errors

- `401 Unauthorized`
  - Missing/invalid `Authorization` header
  - Invalid token
- `403 Forbidden`
  - Valid token but insufficient permissions

## Endpoints

### POST /register

Create a user.

- **Auth**: Not required
- **Request body**: `models.User` (password is accepted but not returned)

#### Request

```json
{
  "name": "Alice",
  "email": "alice@example.com",
  "password": "plaintext-password",
  "role": "member",
  "manager_id": 1
}
```

- Notes:
  - `email` should be unique.
  - `role` is not validated in the controller.

#### Success Response (200)

```json
{
  "message": "User registered"
}
```

#### Error Responses

- `400`

```json
{
  "error": "<binding error>"
}
```

### POST /login

Login with email + password, returns a JWT.

- **Auth**: Not required
- **Request body**: uses `models.User` shape (only `email` and `password` are used)

#### Request

```json
{
  "email": "alice@example.com",
  "password": "plaintext-password"
}
```

#### Success Response (200)

```json
{
  "token": "<jwt>"
}
```

#### Error Responses

- `400`

```json
{
  "error": "<binding error>"
}
```

- `401`

```json
{
  "error": "Invalid credentials"
}
```

---

## Tasks (Requires JWT)

All `/tasks` endpoints require:

- Header: `Authorization: Bearer <token>`

### POST /tasks

Create a task.

- **Auth**: Required
- **Role**: `admin`, `manager`

#### Request

```json
{
  "title": "Implement feature X",
  "description": "Details...",
  "assigned_to_id": 2,
  "progress_percentage": 0
}
```

- Notes:
  - If `assigned_to_id` is `0` or omitted, the task is created as `created`.
  - If `assigned_to_id` is non-zero, status becomes `assigned`.
  - `progress_percentage` must be `0..100`.

#### Success Response (200)

Returns the created task.

```json
{
  "id": 1,
  "title": "Implement feature X",
  "description": "Details...",
  "status": "assigned",
  "progress_percentage": 0,
  "created_by_id": 10,
  "assigned_to_id": 2,
  "completed_at": null,
  "completion_locked": false,
  "pending_approval_notified_at": null,
  "approved_by_id": null,
  "approved_at": null,
  "rejected_by_id": null,
  "rejected_at": null,
  "rejection_reason": "",
  "created_at": "2026-02-17T05:00:00Z"
}
```

#### Error Responses

- `400`

```json
{ "error": "progress_percentage must be between 0 and 100" }
```

- `403`

```json
{ "error": "You do not have permission to assign a task to this user" }
```

- `500`

```json
{ "error": "Failed to create task" }
```

### GET /tasks

List tasks visible to the authenticated user.

- **Auth**: Required
- **Role**: any authenticated role

#### Success Response (200)

Returns an array of tasks (preloaded with `audit_trail` when present).

```json
[
  {
    "id": 1,
    "title": "...",
    "description": "...",
    "status": "in_progress",
    "progress_percentage": 40,
    "created_by_id": 10,
    "assigned_to_id": 2,
    "audit_trail": [
      {
        "id": 5,
        "task_id": 1,
        "action": "approved",
        "actor_id": 3,
        "comments": "Looks good",
        "created_at": "2026-02-17T05:00:00Z"
      }
    ]
  }
]
```

#### Error Responses

- `403`

```json
{ "error": "Unauthorized role" }
```

### GET /tasks/:id

Get a single task by id.

- **Auth**: Required
- **Role**: any authenticated role (must have access)

#### Path params

- `id` (number)

#### Success Response (200)

Returns the task (preloaded with `audit_trail`).

#### Error Responses

- `404`

```json
{ "error": "Task not found" }
```

- `403`

```json
{ "error": "Unauthorized access" }
```

### PUT /tasks/:id

Update an existing task.

- **Auth**: Required
- **Role**:
  - `admin`, `manager`: can update title/description/assignee/status/progress (with constraints)
  - `member`: only allowed to update `progress_percentage` and/or `status` on tasks assigned to themselves

#### Request

All fields are optional.

```json
{
  "title": "New title",
  "description": "New description",
  "assigned_to_id": 3,
  "status": "in_progress",
  "progress_percentage": 50
}
```

#### Status values

Valid values:

- `created`
- `assigned`
- `in_progress`
- `pending_approval`
- `approved`
- `rejected`

Special mapping:

- If you send `"status": "completed"`, it is normalized to `pending_approval`.

#### Status transition rules

- `created` -> `assigned`
- `assigned` -> `in_progress`
- `in_progress` -> `pending_approval`
- `pending_approval` -> `approved` or `rejected`
- `rejected` -> `in_progress`
- `approved` -> (no transitions allowed)

Additional constraints:

- `progress_percentage` must be `0..100`
- To move to `pending_approval`, `progress_percentage` must be `100`
- Approved tasks are locked

#### Success Response (200)

Returns the updated task.

#### Error Responses

- `400`

```json
{ "error": "Invalid task status" }
```

```json
{ "error": "Invalid status transition" }
```

```json
{ "error": "progress_percentage must be 100 before moving to pending_approval" }
```

```json
{ "error": "Approved tasks are locked" }
```

```json
{ "error": "Use /tasks/:id/approve or /tasks/:id/reject for approval decisions" }
```

- `403`

```json
{ "error": "Members can only update tasks assigned to themselves" }
```

```json
{ "error": "Members can only update progress and status on their own tasks" }
```

```json
{ "error": "Unauthorized access" }
```

- `404`

```json
{ "error": "Task not found" }
```

- `500`

```json
{ "error": "Failed to update task" }
```

### POST /tasks/:id/approve

Approve a task that is pending approval.

- **Auth**: Required
- **Role**: `admin`, `manager`

#### Request

Body is optional.

```json
{
  "comments": "Looks good"
}
```

#### Success Response (200)

Returns the updated task.

#### Error Responses

- `400`

```json
{ "error": "Only pending_approval tasks can be approved" }
```

- `403`

```json
{ "error": "Only manager/admin can approve tasks" }
```

```json
{ "error": "Unauthorized access" }
```

- `404`

```json
{ "error": "Task not found" }
```

- `500`

```json
{ "error": "Failed to approve task" }
```

### POST /tasks/:id/reject

Reject a task that is pending approval.

- **Auth**: Required
- **Role**: `admin`, `manager`

#### Request

```json
{
  "reason": "Missing tests",
  "comments": "Please add unit tests"
}
```

- Notes:
  - `reason` is required.
  - If `comments` is empty, it is set to `reason` in the audit record.

#### Success Response (200)

Returns the updated task.

#### Error Responses

- `400`

```json
{ "error": "rejection reason is required" }
```

- `400`

```json
{ "error": "Only pending_approval tasks can be rejected" }
```

- `403`

```json
{ "error": "Only manager/admin can reject tasks" }
```

```json
{ "error": "Unauthorized access" }
```

- `404`

```json
{ "error": "Task not found" }
```

- `500`

```json
{ "error": "Failed to reject task" }
```

### DELETE /tasks/:id

Delete a task.

- **Auth**: Required
- **Role**: `admin`

#### Success Response (200)

```json
{ "message": "Deleted" }
```

#### Error Responses

- `404`

```json
{ "error": "Task not found" }
```

- `403`

```json
{ "error": "Unauthorized access" }
```

---

## Users (Requires JWT + Admin)

All `/users` endpoints require:

- Header: `Authorization: Bearer <token>`
- Role: `admin`

### GET /users

List all users.

#### Success Response (200)

```json
[
  {
    "id": 1,
    "name": "Alice",
    "email": "alice@example.com",
    "role": "admin",
    "manager_id": null,
    "created_at": "2026-02-17T05:00:00Z"
  }
]
```

### PUT /users/:id

Update a user.

#### Request

```json
{
  "role": "manager",
  "manager_id": 2
}
```

- Notes:
  - If `manager_id` is set to the same as the user id, request is rejected.

#### Success Response (200)

Returns the updated user.

#### Error Responses

- `404`

```json
{ "error": "User not found" }
```

- `400`

```json
{ "error": "User cannot be their own manager" }
```

- `400`

```json
{ "error": "<binding error>" }
```
