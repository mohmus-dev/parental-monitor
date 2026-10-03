# Parental Monitor API

This service provides the backend foundation for parent registration, child management, device registration, and monitoring events.

For the recommended frontend stack, browser integration notes, and the API contract available to the frontend, see [Frontend Integration Guide](FRONTEND_INTEGRATION.md).

## Run locally

```bash
git clone <repo>
cd services/api
go run .
```

The API listens on:

```text
http://localhost:8080
```

## Parent registration

### POST /api/v1/parents/register

Creates a new parent and returns a unique UUID-based parent ID.

#### Request body

```json
{
  "name": "John Smith",
  "email": "john@example.com",
  "app_name": "Parent App",
  "platform": "windows",
  "device_id": "parent-device-001"
}
```

#### Success response (201 Created)

```json
{
  "parent": {
    "id": "8f5d4ed2-1b1e-4b6f-8f4e-921648c0e9b3",
    "name": "John Smith",
    "email": "john@example.com",
    "created_at": "2026-10-02T19:00:00Z"
  },
  "message": "parent registered successfully"
}
```

#### Validation rules

- `name` is required
- `email` is required
- duplicate email is rejected with HTTP 409
- invalid JSON returns HTTP 400

## Health

### GET /health

```json
{
  "status": "ok",
  "time": "2026-10-02T19:00:00Z"
}
```

## Auth and parent routes

The API uses a JWT kept in the `auth_token` cookie. Parent-scoped endpoints always resolve the parent from the token claims, never from a URL path value.

### Routes

- `POST /api/v1/auth/login`
- `POST /api/v1/auth/logout`
- `POST /api/v1/parents/register`
- `GET /api/v1/parents/me`
- `DELETE /api/v1/parents/me`
- `GET /api/v1/parents/me/children`
- `POST /api/v1/parents/me/children`
- `POST /api/v1/parents/me/children/{child_id}/pairing-invites`
- `POST /api/v1/devices/pair`
- `DELETE /api/v1/parents/me/children`

### Login

#### POST /api/v1/auth/login

Request body:

```json
{
  "email": "john@example.com",
  "password": "Secret123!"
}
```

Success response:

```json
{
  "token": "<jwt>",
  "expiry": "2026-11-02T19:00:00Z"
}
```

The server also sets a cookie named `auth_token` with the JWT value.

### Logout

#### POST /api/v1/auth/logout

This clears the `auth_token` cookie.

### Current parent profile

#### GET /api/v1/parents/me

Use the `auth_token` cookie to authenticate.

```bash
curl -X GET http://localhost:8080/api/v1/parents/me \
  --cookie "auth_token=<jwt>"
```

### Delete current parent account

#### DELETE /api/v1/parents/me

```bash
curl -X DELETE http://localhost:8080/api/v1/parents/me \
  --cookie "auth_token=<jwt>"
```

### List current parent children

#### GET /api/v1/parents/me/children

```bash
curl -X GET http://localhost:8080/api/v1/parents/me/children \
  --cookie "auth_token=<jwt>"
```

### Create child under current parent

#### POST /api/v1/parents/me/children

Request body:

```json
{
  "name": "Emma",
  "age": 10
}
```

```bash
curl -X POST http://localhost:8080/api/v1/parents/me/children \
  --cookie "auth_token=<jwt>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Emma",
    "age": 10
  }'
```

### Create and redeem a device pairing code

After creating the child profile, an authenticated parent can generate a one-time code:

```bash
curl -X POST http://localhost:8080/api/v1/parents/me/children/<child-id>/pairing-invites \
  --cookie "auth_token=<jwt>"
```

The response includes a random code and an expiry 15 minutes later. The server stores only its hash. Run the CLI manually on the device being paired; the CLI asks its user to review the monitoring disclosure and confirm before redeeming the code:

```powershell
go run . --env .env --pair
```

The CLI reads its settings from the repository-root `.env` file. Set the parent API URL and telemetry receiver settings there, for example:

```dotenv
API_URL=http://localhost:8080
SERVER_URL=https://your-monitoring-ingest.example
```

The user then enters the code shown in the parent dashboard. Pairing links the device to the selected child, records its platform and device ID, and consumes the code. Reuse, expired codes, devices already registered elsewhere, and children already paired to another device are rejected.

This repository does not publish or host CLI installers: pairing is not an install link and will not silently install software. Download/run the CLI through a trusted, explicit installation process. The current CLI captures printable text typed while Chrome is active and active window titles; this can include sensitive text. Pair only with the device user's informed consent.

### Delete selected children for current parent

#### DELETE /api/v1/parents/me/children

Request body:

```json
{
  "child_ids": ["child-1", "child-2"]
}
```

```bash
curl -X DELETE http://localhost:8080/api/v1/parents/me/children \
  --cookie "auth_token=<jwt>" \
  -H "Content-Type: application/json" \
  -d '{
    "child_ids": ["child-1", "child-2"]
  }'
```

## Notes

- Parent identity is resolved from the JWT claims.
- No parent ID is required in the route path for parent-owned actions.
- The parent-specific routes are intentionally scoped to the authenticated user only.
