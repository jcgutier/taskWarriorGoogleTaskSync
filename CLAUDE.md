# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`twgts` is a Go daemon that bidirectionally synchronizes tasks between **Google Tasks** and **Taskwarrior** (CLI task manager). PostgreSQL stores task ID mappings to prevent duplicate creation across sync cycles.

## Commands

```bash
# Build
go build -o twgts twgts.go

# Test all
go test ./...

# Test a single package
go test ./taskwarrior/...

# Start PostgreSQL for local development
docker compose up -d

# Build Docker image
docker build -t task-sync .
```

## Architecture

The sync runs as a periodic loop in `twgts.go`. Each cycle:

1. Fetches all tasks from Google Tasks API (`googleTasks/`) and Taskwarrior CLI (`taskwarrior/`)
2. Queries PostgreSQL (`postgresSql/`) for existing `gid ↔ tid` mappings
3. Google → Taskwarrior: for each `needsAction` Google task not in DB, adds it via `task add`; marks completed Google tasks done in Taskwarrior
4. Taskwarrior → Google: for each pending Taskwarrior task not in DB, creates a Google task; marks completed Taskwarrior tasks done in Google
5. Title-based matching is used for initial association; after that the DB mapping is authoritative

### Key packages

| Package | Role |
|---|---|
| `twgts.go` | Main loop, sync orchestration |
| `config/` | Config loader: JSON file + env var overrides |
| `googleTasks/` | OAuth2 client for Google Tasks API; manages `token.json` caching |
| `taskwarrior/` | Wrapper around the `task` binary; parses JSON export output |
| `tasksSync/` | Data aggregation helpers (partially extracted from main) |
| `postgresSql/` | PostgreSQL client; owns the `tasks` table schema (`gid`, `tid`, `title`, `due_date`, `status`) |

### Configuration

Config is loaded from `config/config.json` (or `CONFIG_FILE_PATH`) with environment variable overrides. Key env vars:

- `GOOGLE_APPLICATION_CREDENTIALS` — path to OAuth2 credentials.json
- `GOOGLE_TASKS_TOKEN_PATH` — path to cached token.json
- `GOOGLE_TASK_LIST_FILTER` — task list name to sync (default: `"Mis tareas"`)
- `SYNC_INTERVAL_SECONDS` — cycle frequency (default: 300)
- `DRY_RUN` — validate without making changes
- `POSTGRES_HOST/USER/PASSWORD/DB` — database connection

### Google OAuth2 flow

On first run with no `token.json`, the process starts an HTTP server on `:8080`, prints an auth URL, exchanges the authorization code, and saves `token.json` for all subsequent runs. In Docker, mount both `credentials.json` and `token.json` as volumes under `/data/`.

### Testing approach

Taskwarrior tests inject a fake `task` binary into `PATH` via a temp directory to avoid requiring a real Taskwarrior install. Tests in `twgts_test.go` cover the `addToTaskwarrior` function. Google Tasks tests are in `googleTasks/googleTasks_test.go`.

### Known TODOs in codebase

- `twgts.go`: task counting logic should move to `tasksSync`; no cleanup of completed tasks from DB
- `taskwarrior.go`: `AddTask` doesn't yet return the created task's UUID; due-date timezone matching needs more tests
- `postgresSql.go`: `DeleteTask` is an unimplemented stub
