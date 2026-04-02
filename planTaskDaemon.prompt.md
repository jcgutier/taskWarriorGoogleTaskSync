## Plan: Docker daemon sync between Google Tasks and Taskwarrior

TL;DR: Convert the current one-shot sync code into a reusable sync service, add a Docker image that includes Taskwarrior and Google auth support, and implement a periodic bidirectional sync loop.

**Steps**
1. Refactor current sync flow into a dedicated sync manager in `tasksSync/tasksSync.go` or a new package.
   - Create a `SyncManager` or `TasksSync` method that loads Google Tasks and Taskwarrior tasks, then performs bidirectional reconciliation.
   - Keep `twgts.go` minimal as the container entrypoint only.
   - Add a readable centralized logging mechanism for the daemon: consistent timestamp/file-line output, log levels, structured context, and rich action messages.

2. Implement robust bidirectional sync logic.
   - Use title-based matching as the initial mapping strategy.
   - For Google->Taskwarrior: add pending Google tasks that are not yet present in Taskwarrior, complete Taskwarrior tasks when Google tasks are completed, and preserve due/notes/project metadata.
   - For Taskwarrior->Google: add pending Taskwarrior tasks that are missing in Google Tasks, include `project=<name>` in notes when present, and avoid duplicates by title.
   - Implement exponential backoff and retry for `AddTask` errors when inserting Taskwarrior tasks into Google Tasks.
   - Add a stable mapping design note: prefer title equality initially, but prepare fields/notes for later stable ID matching if needed.

3. Improve Google Tasks authentication and configuration.
   - Add a configuration file for hardcoded values and defaults, with environment variables available to override.
   - Parameterize token storage path instead of hardcoding `token.json` in the working directory.
   - Keep `GOOGLE_APPLICATION_CREDENTIALS` support and document mounting credentials into the container.
   - Allow selecting a task list name or ID instead of hardcoded `New` (or `Mis tareas`) when inserting tasks.

4. Harden Taskwarrior integration.
   - Add a `GetPendingTasks`/`ListTasks` distinction and optionally filter by `tags:GoogleTasks` if desired.
   - Ensure `task add` and `task done` operations are safe in Docker by requiring the Taskwarrior binary and data volume mount.

5. Add daemon behavior.
   - Implement a main loop in `twgts.go` backed by `time.Ticker` or `time.Sleep`.
   - Configure sync interval from environment variables such as `SYNC_INTERVAL_SECONDS` with a reasonable default (for example 300 seconds).
   - Add a simple shutdown path if the container receives termination signals (optional but recommended).
   - Add a dry-run mode configurable by environment variables so every single sync action can be validated without making changes.
   - Implement a Prometheus-compatible metrics HTTP endpoint (for example `/metrics`) to expose sync counts, successes, failures, and runtime duration metrics.

6. Add Docker support.
   - Create a `Dockerfile` that builds the Go binary and installs Taskwarrior.
   - Base the image on a Debian/Ubuntu-compatible image to simplify installing `task` and avoid Go build issues.
   - Use `supervisord` as the container entrypoint to manage the daemon process and any auxiliary processes.
   - Add a `supervisord.conf` file that starts the sync daemon and optionally logs output to stdout/stderr.
   - Document required volume mounts for `credentials.json`, `token.json`, Taskwarrior config (`.taskrc`) and data directory if needed.

7. Update documentation and usage.
   - Add Docker build/run instructions to `README.md` with the new daemon usage.
   - Document environment variables: `GOOGLE_APPLICATION_CREDENTIALS`, `GOOGLE_TASKS_TOKEN_PATH`, `GOOGLE_TASK_LIST_FILTER`, `SYNC_INTERVAL_SECONDS`, `TASKWARRIOR_DATA_DIR` or mount expectations.

8. Verify and test.
   - Add or extend unit tests for the new sync manager and Google Task note/project parsing.
   - Validate `go test ./...` passes.
   - Perform a Docker build and run test with env vars and mounted credentials.

**Relevant files**
- `twgts.go` — make this the container entrypoint and periodic sync coordinator.
- `tasksSync/tasksSync.go` — central sync logic and data loading.
- `googleTasks/googleTasks.go` — auth/token path and task list parameterization.
- `taskwarrior/taskwarrior.go` — Taskwarrior add/complete helper behavior.
- `README.md` — Docker and daemon usage documentation.
- `Dockerfile` — build and runtime image for the daemon.
- `config/config.go` and `config/config.json` — configuration and hardcoded defaults.

**Verification**
1. Build the Go binary successfully in local environment.
2. Run `go test ./...` to ensure package-level tests pass.
3. Build the Docker image and verify `docker run` starts successfully with mounted credentials and token file.
4. Confirm the container performs at least one bidirectional sync cycle at the configured interval.
5. Validate that Taskwarrior tasks are created/updated and Google Tasks are inserted when missing.

**Decisions**
- The first implementation will use title equality for cross-system matching because the current code already relies on it.
- The Docker daemon will be a continuously running process inside the container rather than a one-shot container command.
- Taskwarrior data and config will be externalized via volume mounts so the container can operate against the host user’s Taskwarrior state.

**Further considerations**
1. Should the sync container support a one-shot mode as well as continuous daemon mode? I recommend yes, using `SYNC_INTERVAL_SECONDS=0` for one-shot.
2. Should the daemon only sync tasks tagged `GoogleTasks` in Taskwarrior, or should it scan all pending tasks? I recommend using `GoogleTasks` for safer round-trips.
3. If you want conflict handling later, add stable ID mapping using notes/tags instead of only title equality.
