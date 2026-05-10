# TechNews TUI

[![CI](https://github.com/joeyism/technews-tui/actions/workflows/ci.yml/badge.svg)](https://github.com/joeyism/technews-tui/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/joeyism/technews-tui)](https://goreportcard.com/report/github.com/joeyism/technews-tui)

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

A beautiful terminal UI for Hacker News, Reddit, LWN.net, Ars Technica, and any other RSS feed, built with Go and Bubble Tea.

<p align="center" width="100%">
<video src="https://github.com/user-attachments/assets/76d957f3-b39f-43ba-868b-0be3412030d3" width="80%" controls></video>
</p>

## Features

- **Multi-source:** Aggregates Hacker News, Reddit, Lobsters, Lemmy, Dev.to, LWN.net, and Ars Technica — plus any RSS/Atom feed you add.
- **Navigable Comment Trees:** Browse discussions with ease on HN, Reddit, Lobsters, Lemmy, and Dev.to.
- **Fold/Unfold:** Collapse threads to focus on what matters.
- **Customizable:** Add/remove subreddits, Lemmy instances, and RSS feeds directly in the TUI.
- **Fast:** Concurrent fetching for all sources.

## Sources

| Source | Type | Comments | Default Feeds |
|--------|------|----------|---------------|
| Hacker News | API | Yes | — |
| Reddit | API | Yes | r/programming, r/linux, r/opencodecli, r/claudecode |
| Lobsters | API | Yes | — |
| Lemmy | API | Yes | lemmy.ml, programming.dev |
| Dev.to | API | Yes | — |
| RSS | RSS/Atom | Browser | LWN.net headlines, Ars Technica Biz & IT |

RSS feeds open in your default browser (press `o`). Add any RSS/Atom feed URL in Settings to pull in blogs, newsletters, or any site that publishes a feed.

## Installation

### Method 1: Install Script (macOS / Linux)

The fastest way to install is via our one-line script. It automatically downloads the correct binary for your system and installs it to `/usr/local/bin`.

```bash
curl -sSfL https://raw.githubusercontent.com/joeyism/technews-tui/main/install.sh | sh
```

### Method 2: Homebrew (macOS / Linux)

If you use Homebrew, you can install via our tap:

```bash
brew tap joeyism/tap
brew install technews-tui
```

### Method 3: Direct Download

Download the pre-compiled binaries for Windows, macOS, and Linux from the [Releases page](https://github.com/joeyism/technews-tui/releases).

### Method 4: Build from Source

If you have Go installed:

```bash
go install github.com/joeyism/technews-tui@latest
```

## Keybindings

### List View
- `↑/↓` or `j/k`: Navigate stories
- `Enter`: View comments
- `o`: Open link in browser
- `Tab`: Cycle source filter (All → HN → Reddit → Lobsters → Lemmy → Dev.to → RSS)
- `s`: Open settings
- `r`: Refresh feed
- `?`: Toggle help

### Comment View
- `↑/↓` or `j/k`: Navigate comments
- `Enter` or `Space`: Fold/unfold thread
- `ctrl+u / ctrl+d`: Scroll half page up/down
- `o`: Open article in browser
- `Esc` or `q`: Back to list

### Settings
- `j/k`: Navigate
- `a`: Add target (subreddit, Lemmy instance, RSS feed URL)
- `d`: Delete target
- `t`: Cycle sort order
- `space`: Toggle source on/off
- `Esc`: Save and back

## License

MIT
