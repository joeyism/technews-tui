# TechNews TUI

[![CI](https://github.com/joeyism/technews-tui/actions/workflows/ci.yml/badge.svg)](https://github.com/joeyism/technews-tui/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/joeyism/technews-tui)](https://goreportcard.com/report/github.com/joeyism/technews-tui)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

A beautiful terminal UI for Hacker News and Reddit, built with Go and Bubble Tea.

## Features

- **Multi-source:** Aggregates Hacker News and your favorite subreddits.
- **Navigable Comment Trees:** Browse discussions with ease.
- **Fold/Unfold:** Collapse threads to focus on what matters.
- **Customizable:** Add/remove subreddits and change sort order directly in the TUI.
- **Fast:** Concurrent fetching for all sources.

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
- `Tab`: Cycle source filter (All → HN → r/linux → ...)
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
- `a`: Add subreddit
- `d`: Delete subreddit
- `t`: Cycle sort order
- `Esc`: Save and back

## License

MIT
