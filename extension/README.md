# Chrome Search Reporter

This extension reports submitted searches from Google, Bing, DuckDuckGo, Yahoo, and YouTube to the local Go monitor. It reads search query parameters from committed top-level navigation URLs; it does not read clipboard data or general page text.

## Setup

1. Copy the repository-root `.env.example` to `.env`, set `API_URL`, `SERVER_URL`, and `API_KEY`, then start the Go monitor with `go run . --env .\.env`. It listens on `127.0.0.1:8765` for extension submissions.
2. In Chrome, open `chrome://extensions`, enable **Developer mode**, choose **Load unpacked**, and select this `extension` directory. This is a one-time setup.
3. Search normally. The extension badge shows `OK` after the local monitor accepts a submitted search; `!` means it could not report it.
4. To monitor Incognito searches, open the extension's **Details** page and enable **Allow in Incognito**. Chrome requires this one-time permission and keeps it disabled by default.

The extension requires access to the listed search-engine hosts and the local monitor endpoint. No API key needs to be copied into the extension: the parent-server API key remains in the Go app's `.env`. The local receiver only accepts requests from a Chrome extension origin. Set default terms in `SCAN_KEYWORDS` and `SCAN_KEYWORD_GROUPS`; add parent-specific terms under `CUSTOM_KEYWORDS` or `CUSTOM_KEYWORD_GROUPS`. Values for these settings are JSON arrays/objects inside quoted dotenv values. Custom entries merge with defaults. Matching alerts are sent using `SERVER_URL`.
