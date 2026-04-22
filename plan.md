# Project Evolution Plan: Docker, Metrics, and Alerting

This plan outlines the steps to transition the TaskWarrior Google Task Sync (twgts) tool into a robust, containerized daemon with integrated monitoring and alerting.

## 1. Objectives
- **Containerization**: Deploy as a Docker container, managed via Supervisor.
- **Registry Integration**: Support pushing to a local Docker registry.
- **Observability**: Implement Prometheus metrics to track sync performance and health.
- **Alerting**: Integrate Discord webhooks for real-time error reporting.
- **Code Quality**: Refactor for DRY (Don't Repeat Yourself) principles and better maintainability.

## 2. Architectural Changes

### 2.1 Code Refactoring (DRY & Clean Code)
- **Modularize `tasksSync.go`**: The current `Sync()` method is monolithic. It will be broken down into:
    - `syncTaskWarriorToGoogle()`: Handles pending and completed tasks from TW to Google.
    - `syncGoogleToTaskWarrior()`: Handles changes from Google Tasks back to TW.
    - `processTaskMapping()`: Helper to manage UUID/ID associations.
- **Unified Logging & Error Handling**: Replace scattered `log.Printf` with a structured approach that can trigger alerts.
- **Configuration Enhancement**: Ensure all new features (Discord webhook URL, Prometheus port) are configurable via environment variables.

### 2.2 Observability (Prometheus)
- **New Package `internal/metrics`**:
    - `sync_duration_seconds`: Histogram of how long sync cycles take.
    - `sync_errors_total`: Counter for failed sync attempts.
    - `tasks_synced_total`: Counter for number of tasks processed.
    - `google_api_requests_total`: Counter for API calls to track rate limiting.
- **Endpoint**: Expose `:9090/metrics` within the container.

### 2.3 Alerting (Discord)
- **New Package `internal/alerts`**:
    - Implement a Discord client that sends formatted embeds on `Fatalf` or critical `Sync()` errors.
    - Throttle alerts to avoid spamming during persistent network issues.

### 2.4 Docker & Supervisor
- **Dockerfile**:
    - Multi-stage build (already present, will be optimized).
    - Ensure `taskwarrior` is correctly configured inside the container.
    - Expose Prometheus port.
- **Supervisor**:
    - Use `supervisord` as the entrypoint.
    - Configure `supervisorctl` to allow process management from within or outside the container (via unix socket).
- **Local Registry**:
    - Add a `scripts/publish.sh` to tag and push the image to a local registry (e.g., `localhost:5000`).

## 3. Implementation Roadmap

### Phase 1: Foundation & Refactoring
1.  **Refactor `tasksSync.go`**: Extract logic into smaller, testable methods.
2.  **Clean up `twgts.go`**: Improve the main loop to handle signals (SIGTERM/SIGINT) gracefully.
3.  **Update `config.go`**: Add fields for Discord Webhook and Prometheus settings.

### Phase 2: Observability & Alerting
1.  **Implement Metrics**: Add Prometheus instrumentation throughout the sync logic.
2.  **Implement Discord Alerts**: Create the alerting service and hook it into the error paths.
3.  **Validation**: Verify metrics are exposed and Discord messages are sent on simulated failures.

### Phase 3: Docker & Deployment
1.  **Update `supervisord.conf`**: Enable HTTP server for `supervisorctl` and log rotation.
2.  **Update `Dockerfile`**: Optimize for size and security.
3.  **Registry Script**: Create `scripts/setup_registry.sh` and `scripts/publish.sh`.
4.  **Compose Update**: Update `docker-compose.yml` to include the sync service and local registry if necessary.

## 4. Improvement Notes (DRY)
- **Database Access**: Consolidate SQLite3 operations to avoid redundant connection openings if possible, or ensure consistent `defer close()` patterns.
- **Mapping Logic**: Centralize the logic that determines if a Google Task and a TaskWarrior task are "the same" to avoid logic drift between different sync directions.
