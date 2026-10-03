# Frontend Integration Guide

This guide describes the frontend that can be built against the API as it exists today. The API currently supports parent accounts and parent-managed child devices. Search history, alerts, and device-listing handlers are not registered as public routes yet, so a monitoring dashboard cannot display those datasets from this API at this stage.

## Recommended frontend stack

Use **Svelte with TypeScript and Vite** for the parent web dashboard. It is a good fit for a form-and-dashboard application: TypeScript provides typed API contracts, Svelte keeps the component model small, and Vite gives a fast development server without requiring a server-rendering framework. The Go service remains the backend; this choice only applies to the browser UI.

Create a starter app with:

```bash
npm create vite@latest parent-dashboard -- --template svelte-ts
cd parent-dashboard
npm install
npm run dev
```

Suggested frontend structure:

```text
src/
  lib/
    api.ts          # shared fetch wrapper and API calls
    types.ts        # Parent and Child response types
  routes-or-views/
    Login.svelte
    Register.svelte
    Dashboard.svelte
    Children.svelte
```

Keep the frontend focused on these flows for this stage:

1. Register or log in as a parent.
2. Load the authenticated parent's profile and children.
3. Register a child's device, then show it in the children list.
4. Allow selecting and deleting child devices.
5. Log out; offer account deletion separately with a confirmation step.

## API base URL

Local API default: `http://localhost:8080`. Use an environment variable in the frontend, for example `VITE_API_BASE_URL`. The API currently has **no CORS middleware**. For local development, configure the Vite dev server to proxy `/api` and `/health` to `http://localhost:8080`, then call relative paths from the browser. This makes frontend calls same-origin from the browser's perspective and avoids requiring CORS for local development.

For a deployed frontend on a separate origin, configure backend CORS to allow that exact origin and credentials, and configure cookies for HTTPS/cross-site use. The current API sets `Secure: false` and `SameSite=Lax`, which is suitable only for local HTTP development, not a production cross-origin deployment.

## Endpoints available now

All paths below are relative to the API base URL. Parent-scoped operations derive the parent ID from the `auth_token` cookie; the frontend must not send a parent ID in the URL.

| Method | Path | Authentication | Purpose |
| --- | --- | --- | --- |
| `GET` | `/health` | No | API health check |
| `POST` | `/api/v1/parents/register` | No | Create a parent account |
| `POST` | `/api/v1/auth/login` | No | Authenticate and set `auth_token` cookie |
| `POST` | `/api/v1/auth/logout` | No | Clear the browser's `auth_token` cookie |
| `GET` | `/api/v1/parents/me` | Cookie | Get the current parent's profile |
| `DELETE` | `/api/v1/parents/me` | Cookie | Delete the parent and associated children |
| `GET` | `/api/v1/parents/me/children` | Cookie | List the current parent's children |
| `POST` | `/api/v1/parents/me/children` | Cookie | Register a child device under the current parent |
| `DELETE` | `/api/v1/parents/me/children` | Cookie | Delete selected child records |

### Register parent

`POST /api/v1/parents/register` accepts:

```json
{
  "name": "John Smith",
  "email": "john@example.com",
  "password": "Secret123!",
  "app_name": "Parent Dashboard",
  "platform": "windows",
  "device_id": "parent-device-001"
}
```

Returns `201 Created` with `{ "parent": { ... }, "message": "parent registered successfully" }`. The returned parent contains `id`, `name`, `email`, `created_at`, and any supplied device metadata. `password_hash` is not returned. Registration does not create a login session; call login afterward.

### Login and cookie session

`POST /api/v1/auth/login` accepts `{ "email": "john@example.com", "password": "Secret123!" }`. A successful response includes `token` and `expiry`, and also sets the `HttpOnly` `auth_token` cookie. The frontend should not put the token in `localStorage` or read it for normal requests; let the browser retain and send the cookie.

Include credentials on browser requests, including login, so the browser accepts and sends the cookie:

```ts
const response = await fetch(`${API_BASE_URL}/api/v1/auth/login`, {
  method: "POST",
  credentials: "include",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({ email, password }),
});
```

`POST /api/v1/auth/logout` clears the cookie. The current implementation does not maintain a server-side token revocation list; logout removes the browser cookie, while a copied JWT remains valid until its expiry. Authenticated endpoints return `401` for invalid or expired sessions. Expired sessions use `{ "error": "session_expired", "message": "Your session has expired. Please log in again." }` and clear the cookie.

### Current parent and child operations

- `GET /api/v1/parents/me` returns the parent object directly.
- `GET /api/v1/parents/me/children` returns an array of child objects.
- `POST /api/v1/parents/me/children` accepts `name` and optional `age`, `device_id`, `app_name`, and `platform`; omit `device_id` to create an unpaired child profile.
- `POST /api/v1/parents/me/children/{child_id}/pairing-invites` requires the parent JWT cookie and returns a single-use code that expires after 15 minutes.
- `POST /api/v1/devices/pair` accepts `{ "code": "...", "device_id": "...", "platform": "windows" }` without a parent cookie and associates the device with the child on the invite. The CLI prompts locally for informed consent before submitting.
- `DELETE /api/v1/parents/me/children` accepts `{ "child_ids": ["child-id-1"] }` and returns `{ "message": "selected child devices deleted successfully" }`.
- `DELETE /api/v1/parents/me` deletes the account and associated children, then clears the cookie.

The child response fields are `id`, `parent_id`, `name`, `age`, `device_id`, `app_name`, `platform`, and `created_at` (optional metadata may be omitted). Pairing codes are returned once and only their hashes are stored. The repository does not host CLI installers; installing and running the CLI remains a separate, visible user action. The CLI currently captures printable text typed while Chrome is active and active window titles, which can include sensitive text.

## Small typed API client

Centralize cookie credentials, JSON headers, and error handling rather than repeating fetch options in each view:

```ts
const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? "";

export type Parent = {
  id: string;
  name: string;
  email: string;
  created_at: string;
  app_name?: string;
  platform?: string;
  device_id?: string;
};

export type Child = {
  id: string;
  parent_id: string;
  name: string;
  age?: number;
  device_id?: string;
  app_name?: string;
  platform?: string;
  created_at: string;
};

export async function api<T>(path: string, init: RequestInit = {}): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...init,
    credentials: "include",
    headers: {
      ...(init.body ? { "Content-Type": "application/json" } : {}),
      ...init.headers,
    },
  });

  if (!response.ok) {
    const message = await response.text();
    throw new Error(message || `Request failed (${response.status})`);
  }

  return response.json() as Promise<T>;
}

export const getCurrentParent = () => api<Parent>("/api/v1/parents/me");
export const getChildren = () => api<Child[]>("/api/v1/parents/me/children");
```

Some handlers return JSON errors (notably authentication errors), while others use Go's plain-text HTTP error response. Handle non-2xx responses without assuming every error body is JSON.

## Known integration and security gaps

- CORS is not configured. Use a same-origin development proxy now; add explicit credentialed CORS before hosting the UI on another origin.
- Cookies currently have `Secure: false`. Production deployment must use HTTPS and secure cookie settings.
- No explicit CSRF token mechanism is implemented. Before production, review CSRF defenses for cookie-authenticated state-changing requests.
- Storage is in-memory by default and loses data when the API process restarts. Firebase-backed storage is selected by the API environment configuration; confirm it is enabled and credentials are configured for persistent environments.
- The router currently does not expose search-event, alert, or device-list APIs. Add and authenticate those routes before implementing monitoring-history screens.