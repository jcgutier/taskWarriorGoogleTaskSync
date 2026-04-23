# TaskWarrior Google Tasks Sync (TWGTS) - Development Plan

This plan outlines the steps to transition the sync tool into a robust, containerized daemon with integrated monitoring and alerting, using SQLite as the primary data store.

## Status Legend
- ` [ ] ` To Do
- ` [/] ` In Progress
- ` [x] ` Done

## Phase 1: Foundation, Dockerization & Refactoring
Focus on making the application a robust daemon and organizing the codebase for scale.

### 1.1 Robust Daemon Loop ([/])
Improve the main loop in `twgts.go` to handle signals and ensure clean execution.
- ` [x] ` Basic periodic loop implemented.
- ` [ ] ` Replace simple `time.Sleep` with `time.Ticker`.
- ` [ ] ` Add graceful shutdown handling (SIGTERM/SIGINT).
- ` [ ] ` Ensure first sync runs immediately on startup.

### 1.2 Code Refactoring (DRY & Clean Code) ([ ])
Break down the monolithic `Sync()` method in `tasksSync.go` for better maintainability.
- ` [ ] ` Extract `syncTaskWarriorToGoogle()` logic.
- ` [ ] ` Extract `syncGoogleToTaskWarrior()` logic.
- ` [ ] ` Extract `processTaskMapping()` helper.
- ` [ ] ` Consolidate SQLite3 operations to ensure consistent `defer close()` patterns.

### 1.3 Dockerization ([ ])
Optimize the container for daemon use and local deployment.
- ` [ ] ` Update `docker-compose.yml` to run `twgts` as a standalone service (removing Postgres).
- ` [ ] ` Configure volumes for `${GOOGLE_APPLICATION_CREDENTIALS}`, `${GOOGLE_TASKS_TOKEN_PATH}`, and the Taskwarrior SQLite database.
- ` [ ] ` Ensure `taskwarrior` is correctly configured inside the container environment.

## Phase 2: Observability & Alerting
Introduce monitoring and real-time error reporting.

### 2.1 Prometheus Metrics Implementation ([ ])
- ` [ ] ` Create `internal/metrics` package.
- ` [ ] ` Track `sync_duration_seconds` (Histogram).
- ` [ ] ` Track `sync_errors_total` (Counter).
- ` [ ] ` Track `tasks_synced_total` (Counter).
- ` [ ] ` Expose `:9090/metrics` endpoint.

### 2.2 Discord Alerting Integration ([ ])
- ` [ ] ` Create `internal/alerts` package for Discord webhooks.
- ` [ ] ` Implement throttled error reporting for critical failures.

## Phase 3: Deployment & Refinement
Finalize automation and process management.

### 3.1 Supervisor Optimization ([ ])
- ` [ ] ` Enable HTTP server for `supervisorctl` in `supervisord.conf`.
- ` [ ] ` Configure log rotation and ensure logs are directed to stdout/stderr.

### 3.2 Automation & Registry Integration ([ ])
- ` [ ] ` Create `scripts/publish.sh` to tag and push the image to a local registry.
- ` [ ] ` Optimize Dockerfile for minimal size (multi-stage build).

---

## Pre-flight: Generate token.json (once, on host)

The container cannot perform interactive OAuth. Generate the token on the host first:

```bash
GOOGLE_APPLICATION_CREDENTIALS=/path/to/credentials.json \
GOOGLE_TASKS_TOKEN_PATH=/path/to/token.json \
  go run twgts.go
```

The container will then mount this `token.json` read-only.
