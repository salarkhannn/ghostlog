# ghostlog

A live Terminal UI that watches a Git repository and captures every commit an AI coding agent makes — in real-time.

```
⚡ Agent Speed: 12.0 commits/min                          watching /home/user/my-project
┌──────────────────────────────────────┐┌───────────────────────────────────────────────────┐
│ [Burst #1] 3 commits in 4s           ││ commit a1b2c3d4                                   │
│   +142 -0 (+8.2kb) across 4 files    ││ Author: Claude <agent@cursor.sh>                  │
│                                      ││                                                   │
│▶ [Burst #2] 7 commits in 12s         ││ diff --git a/internal/server/handler.go b/...     │
│   +380 -44 (+22kb) across 9 files    ││ @@ -0,0 +1,42 @@                                 │
│                                      ││ +func (s *Server) handleShorten(w http.Re...      │
│ [Burst #3] 1 commit                  ││ +    if r.Method != http.MethodPost {             │
│   +12 -3 (+0.8kb) across 2 files     ││ +        http.Error(w, "method not allowed"...    │
└──────────────────────────────────────┘└───────────────────────────────────────────────────┘
  commits: 11  +534 -47 lines  session: 3m12s  auto-scroll: on  [j/k] navigate  [q] quit
```

## Installation

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

Start ghostlog **before** launching your AI agent. The TUI opens immediately and starts watching for commits.

```sh
# Terminal 1
ghostlog -repo ~/my-project

# Terminal 2 (or use Cursor, Claude Code, Aider, etc.)
cd ~/my-project && aider
```

## Keybindings

| Key | Action |
|-----|--------|
| `j` / `↓` | Select next burst |
| `k` / `↑` | Select previous burst |
| `Ctrl+D` / `PgDn` | Scroll diff down |
| `Ctrl+U` / `PgUp` | Scroll diff up |
| `a` | Toggle auto-scroll (follows newest burst) |
| `c` | Copy commit hashes of selected burst to clipboard |
| `q` / `Ctrl+C` | Quit |

## Concepts

**Burst** — A group of commits that arrive within 5 seconds of each other. AI agents often make 5–20 rapid commits in a single task; grouping them into bursts makes the session readable at a glance.

**Agent Speed** — Rolling 60-second window of commits/min. Spikes indicate the agent is actively writing code.

**Filtering** — Empty commits (no file changes) are silently ignored.

## Works with

Any agent that commits to git: **Aider**, **Claude Code**, **Cursor**, **Devin**, **OpenHands**, custom scripts.
