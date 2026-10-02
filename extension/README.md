# Chrome Search Reporter

This extension reports submitted searches from Google, Bing, DuckDuckGo, Yahoo, and YouTube to the local Go monitor. It reads search query parameters from committed top-level navigation URLs; it does not read clipboard data or general page text.

## Setup

1. Start the Go monitor with `go run . -config .\config.json`. It listens on `127.0.0.1:8765` for extension submissions.
2. In Chrome, open `chrome://extensions`, enable **Developer mode**, choose **Load unpacked**, and select this `extension` directory. This is a one-time setup.
3. Search normally. The extension badge shows `OK` after the local monitor accepts a submitted search; `!` means it could not report it.
4. To monitor Incognito searches, open the extension's **Details** page and enable **Allow in Incognito**. Chrome requires this one-time permission and keeps it disabled by default.

The extension requires access to the listed search-engine hosts and the local monitor endpoint. No API key needs to be copied into the extension: the parent-server API key remains in the Go app's `config.json`. The local receiver only accepts requests from a Chrome extension origin. Default terms stay in `scan_keywords` and `scan_keyword_groups`; add parent-specific terms under `custom_keywords` or `custom_keyword_groups`. These custom entries are merged with the defaults, not substituted for them. Matching alerts are sent using the existing configured server URL.
