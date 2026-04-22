# TaskWarrior Google Tasks Sync (TWGTS) - Project Context

This project is a synchronization tool written in Go that provides bidirectional sync between Google Tasks and Taskwarrior.

## Project Overview

*   **Purpose:** Synchronize tasks between Google Tasks and Taskwarrior, maintaining status, due dates, and metadata (projects/tags).
*   **Main Technologies:**
    *   **Go:** Core application logic.
    *   **Google Tasks API:** Integration with Google's task management service.
    *   **Taskwarrior:** CLI-based task manager.
    *   **SQLite3:** Used for two purposes:
        1. Reading Taskwarrior's internal state (directly from `taskchampion.sqlite3`).
        2. Maintaining a mapping between Taskwarrior UUIDs and Google Task IDs in a custom `goTasksSync` table.
    *   **Docker & Docker Compose:** Containerization and development environment setup.

## Architecture

*   **Entry Point (`twgts.go`):** Runs a periodic loop (default every 20 seconds) that loads configuration and triggers the sync process.
*   **Sync Logic (`tasksSync/`):** The core engine that orchestrates the flow. It identifies new, updated, or completed tasks on both sides and reconciles them.
*   **Database (`sqlite3/`):** Handles direct interaction with the SQLite3 database used by modern Taskwarrior (Taskchampion). It creates a mapping table `goTasksSync` to track synchronization state.
*   **Taskwarrior Client (`taskwarrior/`):** Wraps Taskwarrior CLI commands (`task add`, `task modify`, `task export`, etc.) for write operations.
*   **Google Tasks Client (`googleTasks/`):** Manages OAuth2 authentication and API calls to Google Tasks.
*   **Configuration (`config/`):** Supports loading from `config/config.json` and environment variables.

## Building and Running

### Prerequisites
*   Go 1.26+
*   Taskwarrior installed locally.
*   Google Cloud Console credentials (`credentials.json`) with Google Tasks API enabled.

### Development Commands
*   **Build:** `go build -o twgts twgts.go`
*   **Run:** `./twgts`
*   **Test:** `go test ./...`
*   **Docker Build:** `docker build -t task-sync .`
*   **Docker Compose (Postgres):** `docker compose up -d` (Note: SQLite is used for sync mapping, but Postgres is supported as an alternative backend in `config`).

## Development Conventions

*   **Error Handling:** Extensive use of `log.Fatalf` in the main loop and `fmt.Errorf` with wrapping for library code.
*   **Logging:** Uses standard library `log` with file and line number information (`Lshortfile`).
*   **Dry Run:** Many operations support a `DryRun` flag in the configuration to simulate changes without modifying Taskwarrior or Google Tasks.
*   **Environment Overrides:** All major configuration options can be overridden via environment variables (see `config/config.go`).
*   **Taskwarrior Integration:** Prefers direct SQLite reads for performance and accuracy when fetching tasks, but uses the CLI for modifications to ensure Taskwarrior's own logic (hooks, etc.) is respected.

## Key Files & Directories

*   `twgts.go`: Application entry point and main loop.
*   `tasksSync/tasksSync.go`: Bidirectional reconciliation logic.
*   `sqlite3/sqlite3.go`: SQLite implementation for Taskwarrior data and sync mappings.
*   `taskwarrior/taskwarrior.go`: CLI wrapper for Taskwarrior.
*   `googleTasks/googleTasks.go`: Google Tasks API client.
*   `config/config.go`: Configuration schema and loading logic.
*   `supervisord.conf`: Process management for the Docker container.
