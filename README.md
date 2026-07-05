<p align="center">
  <img src="logo.png?v=2" width="300" alt="Ghostlog Logo">
</p>

<p align="center">
  <video src="https://github.com/salarkhannn/ghostlog/raw/main/demo.mp4" width="100%" autoplay loop muted playsinline></video>
</p>

# ghostlog

A live Terminal UI that watches a Git repository and captures every commit an AI coding agent makes — in real-time.

```text
[AGENT SPEED: 12.0 commits/min] | SESSION: 00:03:12 | watching /path/to/repo
┌──────────────────────────────────────┐┌───────────────────────────────────────────────────┐
│ [#1] 3 commits in <1s  [OK]          ││ commit a1b2c3d4                                   │
│   +142 -0 (8.2kb) across 4 files     ││ Author: Claude <agent@cursor.sh>                  │
│                                      ││                                                   │
│> [#2] 7 commits in 12s  [!!]         ││ diff --git a/internal/server/handler.go b/...     │
│   +380 -44 (22.0kb) across 9 files   ││ @@ -0,0 +1,42 @@                                 │
│                                      ││ +func (s *Server) handleShorten(w http.Re...      │
│ [#3] 1 commits in <1s  [OK]          ││ +    if r.Method != http.MethodPost {             │
│   +12 -3 (800b) across 2 files       ││ +        http.Error(w, "method not allowed"...    │
└──────────────────────────────────────┘└───────────────────────────────────────────────────┘
Total: +534 -47 | 3 bursts | auto: on | [a]uto / [c]opy / [q]uit
```

## Installation

### One-Command Install (macOS / Linux)

Instantly download and install the latest precompiled binary:

```sh
curl -sL https://raw.githubusercontent.com/salarkhannn/ghostlog/main/install.sh | bash
```

### Windows (PowerShell)

Download the latest `Windows_x86_64.tar.gz` from the [Releases](https://github.com/salarkhannn/ghostlog/releases) tab, extract `ghostlog.exe`, and place it in your system PATH.

### Install via Go

If you have Go 1.23+ installed, you can build and install it globally:

```sh
go install github.com/salarkhannn/ghostlog@latest
```

Or build from source:

```sh
git clone https://github.com/salarkhannn/ghostlog
cd ghostlog
go build -o ghostlog .
```

**Requirements:** Go 1.23+, `git` in `$PATH`. No CGO. No external runtime.

## Usage

```sh
ghostlog -repo /path/to/project
```

Start ghostlog **before** launching your AI agent. The TUI opens immediately and starts watching for commits in a separate terminal.

```sh
# Terminal 1
ghostlog -repo ~/my-project

# Terminal 2 (or use Cursor, Claude Code, Aider, etc.)
cd ~/my-project && aider
```

### Subcommands

**Export Session Manifest**
Export a JSONL manifest of the burst log with metadata and complexity deltas:
```sh
ghostlog export -session /path/to/project -out manifest.jsonl
```

**CI Gate Mode**
Run ghostlog in a headless mode in CI pipelines to block complex or untested AI code:
```sh
ghostlog check -session /path/to/project -fail-on complexity,coverage -max-complexity-delta 10 -min-coverage-touch 0.8
```

## Keybindings

| Key | Action |
|-----|--------|
| `Tab` | Switch focus between Burst List and Diff Viewport |
| `j` / `↓` | Scroll selected pane down |
| `k` / `↑` | Scroll selected pane up |
| `p` / `n` / `[` / `]` | Cycle through bursts globally (previous/next) |
| `Ctrl+D` / `PgDn` | Scroll diff down |
| `Ctrl+U` / `PgUp` | Scroll diff up |
| `a` | Toggle auto-scroll (follows newest burst) |
| `c` | Copy commit hashes of selected burst to clipboard |
| `v` | Toggle file-flash treemap view |
| `s` | Open the Session Manager |
| `q` / `Ctrl+C` | Quit |

## Concepts

**Burst** — A group of commits that arrive within 5 seconds of each other. AI agents often make 5–20 rapid commits in a single task; grouping them into bursts makes the session readable at a glance.

**Agent Speed** — Rolling 60-second window of commits/min. Spikes indicate the agent is actively writing code.

**Filtering** — Empty commits (no file changes) are silently ignored.

## Works with

Any agent that commits to git: **Aider**, **Claude Code**, **Cursor**, **Devin**, **OpenHands**, custom scripts.
