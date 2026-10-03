# Parental Monitor CLI

A local Go application that receives submitted search queries from a Chrome extension, checks them against configurable keyword groups, records eligible searches locally, and sends matching alerts to a parent-side HTTP API.

## How It Works

```mermaid
flowchart LR
    P[Parent registers/login] --> D[Parent dashboard]
    D --> C[Create child profile]
    C --> I[Generate one-time pairing code]
    I --> M[Child app installs CLI + extension]
    M --> A[Consent + incognito approval]
    A --> B[Device links to parent]
    B --> E[CLI stores searches in local SQLite]
    E --> W[Hourly worker syncs to Postgres]
    W --> G[Postgres monitoring store]
    G --> R[Parent dashboard reads backend data]
    R --> O[Offline-friendly parent SQL cache for 7-day retention]
```

The system is intentionally split in three layers:

- Identity + access: Firebase stores parent, child, and pairing metadata.
- Monitoring data: Postgres stores alert/search events for fast querying and large data sets.
- Local child runtime: the CLI keeps a local SQLite database for offline buffering and sync.

The parent flow is: register email/password -> login -> dashboard -> create child profile -> generate pairing code -> child installs the CLI and extension -> confirm extension + incognito monitoring consent -> device links to parent. Every dashboard page is backed by API data, not static mock content.

The browser extension works only on supported search engines and reports committed search navigation events. It does not read all page text or clipboard content. The CLI listens on `127.0.0.1:8765` and records local searches in `monitor.db`. Incognito traffic is explicitly gated behind the user approval flow, and matching incident alerts still continue to be evaluated when allowed.

The hourly worker reads new rows from SQLite, writes them to Postgres, and tracks the last synced position so it never replays already processed rows. The dashboard reads from the parent API, which aggregates child data and shows real-time monitoring details from the backend.

For performance at scale, the matcher uses a compiled token trie and category-based matching so it remains fast even with large keyword dictionaries and high-volume search traffic. All state-changing requests are protected by JWT and CSRF validation.

## Requirements

- Go toolchain compatible with the version in `go.mod`.
- A C compiler/toolchain if your platform needs one for the SQLite driver (`github.com/mattn/go-sqlite3`).
- Google Chrome or a compatible Chromium browser.
- A parent server that accepts `POST <server_url>/api/alerts` with the `X-API-Key` header.

## Quick Start: Windows

Open PowerShell in the repository root (`D:\Project\parental-monitor-cli`). Copy `.env.example` to `.env` and edit the values:

```powershell
Copy-Item .env.example .env
```

For parent/device setup, use separate PowerShell windows.

1. Start the parent API:

   ```powershell
   go run .\services\api
   ```

   It listens on `http://localhost:8080` and reads `PORT` and `JWT_SECRET` from `.env`.

2. Start the parent dashboard in a second window:

   ```powershell
   Set-Location services\dashboard
   npm.cmd run dev
   ```

   Open the URL printed by Vite. Register/log in, create a child profile, and generate a one-time pairing code.

3. On the child device, run the CLI from the repository root:

   ```powershell
   go run . --env .\.env --pair
   ```

   Type `yes` at the consent prompt, enter the dashboard code, and the CLI saves `CHILD_ID` and `DEVICE_ID` into `.env`.

4. Start the CLI normally with `go run . --env .\.env`. Load the unpacked Chrome extension from the `extension` folder at `chrome://extensions`, then submit a search.

To stop the foreground monitor, press **Ctrl+C**. The receiver status page is `http://127.0.0.1:8765/`; `/search` is an extension API endpoint, not a page to open directly.

The CLI's alert sender uses `SERVER_URL` and `API_KEY`, but the parent API here does not yet expose the `/api/alerts` telemetry endpoint. Configure a compatible ingest server before expecting alerts to reach the dashboard. The included testserver also uses port 8080, so it cannot run at the same time as the parent API on its default port.

## Configuration

The CLI reads settings only from `.env`. Copy `.env.example` to `.env`, edit it, and restart the app after changing settings. Pairing saves `CHILD_ID` and `DEVICE_ID` into that same `.env` file.

| Field | Meaning |
| --- | --- |
| `POSTGRES_DSN` | Neon or Postgres connection string used by the API and worker. Example: `postgresql://user:pass@host/db?sslmode=require`. |
| `USE_FIREBASE` | Enables Firebase-backed identity and pairing repositories. |
| `FIREBASE_PROJECT_ID` | Firebase project ID used for parent-child identity. |
| `FIREBASE_CREDENTIALS_PATH` | Path to the Firebase service account JSON. |
| `JWT_SECRET` | Shared secret for JWT signing and validation. |
| `API_URL` | Parent management API URL used for device pairing. |
| `SERVER_URL` | Telemetry receiver base URL. The client appends `/api/alerts`. |
| `API_KEY` | Sent to the telemetry receiver as `X-API-Key`; it is not copied into the Chrome extension. |
| `DEVICE_NAME` | Included in locally stored search events. |
| `SCAN_KEYWORDS` | JSON string array of terms; terms not already in a group become their own category. |
| `SCAN_KEYWORD_GROUPS` | JSON object mapping categories to string arrays. |
| `CUSTOM_KEYWORDS` | Additional JSON string array merged with `SCAN_KEYWORDS`. |
| `CUSTOM_KEYWORD_GROUPS` | Additional JSON object of categories to merge with default groups. |
| `LOG_LEVEL` | `debug`, `info`, `warn`, or `error`. Defaults to `info`. |
| `MONITOR_INTERVAL_MS`, `BATCH_SIZE`, `FLUSH_INTERVAL_SEC` | Numeric settings; defaults are 50, 10, and 5. |
| `WORKER_ENABLED`, `WORKER_SYNC_INTERVAL_SECONDS`, `MONITOR_DB_PATH` | Enable and configure the local SQLite -> Postgres sync worker. |
| `CHILD_ID`, `DEVICE_ID` | Set by successful device pairing and stored in `.env`. |

Example custom additions:

```dotenv
CUSTOM_KEYWORDS='["family-specific phrase"]'
CUSTOM_KEYWORD_GROUPS='{"school_rules":["custom phrase","another phrase"]}'
```

The analyzer normalizes case, matches complete words or consecutive word phrases, and returns one match per category. It does not infer synonyms: add each desired synonym explicitly. Broad terms can produce false positives, so test and tune the lists for your family.

## Tests And Performance

Run the package tests from the repository root:

```powershell
go test -vet=off ./...
```

`-vet=off` avoids the existing vet warning in `testserver/test_server.go`; tests still run normally.

Run the analyzer benchmark at 10,000, 100,000, and 1,000,000 terms:

```powershell
go test -run '^$' -bench BenchmarkFindMatchesLargeDictionary -benchmem ./internal/monitor
```

The analyzer compiles an Aho-Corasick-style token trie once during startup. Query matching traverses the query tokens and emits actual matches rather than checking every configured term. Index construction time and memory still grow with dictionary size; the benchmark measures lookup after construction.

## Build And Run

Build for the current OS:

```powershell
New-Item -ItemType Directory -Force .\build | Out-Null
go build -o .\build\parental-monitor.exe .
.\build\parental-monitor.exe -env .\.env
```

The source also contains `Makefile` targets for platform builds and `build.sh` for cross-platform builds. Windows users can use `go build` directly as above.

Daemon mode is for a built executable, not the `go run` development flow:

```powershell
.\build\parental-monitor.exe -env .\.env -daemon
.\build\parental-monitor.exe -stop
```

The daemon writes `daemon.log` and `monitor.pid` in its working directory. Foreground app logs are printed and appended to `monitor.log`.

## Repository Map

### Application and configuration

- `main.go` loads config, opens SQLite, compiles the matcher, starts the local HTTP receiver, wires search/alert callbacks, and shuts down gracefully.
- `.env` contains the CLI configuration and pairing state; copy `.env.example` as a starting point.
- `go.mod`, `go.sum` declare and lock Go dependencies.
- `daemon_windows.go`, `daemon_unix.go` detach and stop the app on their respective OSes.
- `Makefile`, `build.sh` provide build, test, formatting, and cross-build commands.

### Runtime packages

- `internal/searchreceiver/handler.go` handles extension submissions at `POST /search`; `GET /` is a local status response. It validates request shape and invokes the search and alert callbacks.
- `internal/monitor/analyzer.go` merges categories, compiles the token trie/failure links, and finds all matching categories efficiently.
- `internal/monitor/monitor.go` defines search/alert data types and retains the earlier keystroke-driven monitor implementation. `main.go` currently does **not** start that monitor; active browser capture is through the extension and `searchreceiver`.
- `internal/monitor/keystroke*.go`, `window*.go` support that older polling implementation and are not used by the active `main.go` flow.
- `internal/network/client.go` queues alerts and posts them to `<server_url>/api/alerts` with the API key.
- `internal/storage/storage.go` creates `monitor.db` tables and stores non-incognito search history and alert queries.
- `internal/log/logger.go` writes structured console lines and appends them to `monitor.log`.
- `testserver/test_server.go` is a simple local development server that prints incoming POST JSON.

### Chrome extension

- `extension/manifest.json` declares the Manifest V3 extension, navigation permissions, and supported host permissions.
- `extension/service-worker.js` detects committed top-level search URLs, extracts the query, notes whether the tab is Incognito, and posts it to the local receiver.
- `extension/popup.html` explains the extension status and Incognito permission.
- `extension/README.md` has extension-specific loading instructions.

### Generated/local files

`monitor.db`, `monitor.log`, `daemon.log`, and `monitor.pid` are runtime data, not source code. `scriptToUpdateTabs.txt` is a local PowerShell utility for reindenting `Makefile`; it is not part of app startup.

## Privacy And Scope

The extension has access only to its declared search-engine hosts and the local receiver. It reports committed search queries, not every keystroke or clipboard item. Benign searches remain local; matching alert events, including the full query, are sent to the configured parent API. Chrome disables extension access in Incognito until the user explicitly enables it.
