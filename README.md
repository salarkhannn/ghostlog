# ghostlog

`ghostlog` is an ultra-low-footprint background daemon that watches a Git repository and captures a structured audit log of every commit — designed to make AI coding sessions (Claude Code, Cursor, Copilot, etc.) fully auditable.

## How it works

1. `ghostlog` monitors `.git/refs/` using `fsnotify`.
2. On each detected commit, it runs `git diff-tree` to extract the exact diff.
3. The diff is parsed with `go/ast` to classify files (test, generated, core logic) and count new functions.
4. A JSONL record is appended atomically to `.ghostlog.jsonl`.

## Installation

```sh
go install github.com/salarkhannn/ghostlog/cmd/ghostlog@latest
go install github.com/salarkhannn/ghostlog/cmd/ghostlog/replay@latest
```

Or build from source:

```sh
git clone https://github.com/salarkhannn/ghostlog
cd ghostlog
go build -o ghostlog ./cmd/ghostlog
go build -o ghostlog-replay ./cmd/ghostlog/replay
```

## Usage

### Start the daemon

```sh
ghostlog -repo /path/to/your/repo
```

Optional flags:

| Flag   | Default                        | Description                     |
|--------|--------------------------------|---------------------------------|
| `-repo` | `.` (current directory)       | Path to the Git repository      |
| `-out`  | `<repo>/.ghostlog.jsonl`      | Output JSONL log file path      |

On startup:
```
ghostlog: watching /path/to/your/repo...
```

On `SIGINT` / `SIGTERM`:
```
ghostlog: caught interrupt, saving log...
```

The daemon produces **no other stdout/stderr output** during normal operation.

### Replay a session

```sh
ghostlog-replay -log /path/to/repo/.ghostlog.jsonl
```

Example output:

```
ghostlog session — 3 commits
────────────────────────────────────────────────────────────
[01] a1b2c3d4  2026-07-03 09:12:01
     +42   -0  across 2 file(s)
     internal/parser/parser.go               [funcs:3]
     internal/parser/parser_test.go          [test] [funcs:2]

[02] e5f6a7b8  2026-07-03 09:13:44
     +18   -5  across 1 file(s)
     cmd/ghostlog/main.go                    [funcs:1]

[03] c9d0e1f2  2026-07-03 09:14:22
     +120  -0  across 1 file(s)
     zz_generated_mock.go                    [generated]

────────────────────────────────────────────────────────────
session duration: 2m21s
```

## Log format

Each line in `.ghostlog.jsonl` is a JSON object:

```json
{
  "timestamp": "2026-07-03T04:12:01.000Z",
  "commit_hash": "a1b2c3d4",
  "files_changed": [
    {
      "path": "internal/parser/parser.go",
      "lines_added": 34,
      "lines_removed": 0,
      "is_test": false,
      "is_generated": false,
      "func_count": 3
    }
  ]
}
```

## Integration tip

Run `ghostlog` in a background terminal before launching your AI coding agent:

```sh
ghostlog -repo . &
cursor .          # or claude-code, copilot, etc.
```

After the session, replay the log:

```sh
ghostlog-replay
```

## Requirements

- Go 1.23+
- `git` in `$PATH`
- No CGO. No external runtime. Single static binary.
