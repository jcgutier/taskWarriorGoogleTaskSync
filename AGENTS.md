# AGENTS.md

## Repository overview
- Small Go CLI that reads Google Tasks and syncs them into Taskwarrior.
- Entry point: `twgts.go`.
- Internal packages:
  - `googleTasks/`: Google Tasks API client and OAuth flow.
  - `taskwarrior/`: wrapper around the `task` CLI.

## Repository layout
- `twgts.go`: main program and current sync entry point.
- `googleTasks/googleTasks.go`: creates the Google Tasks service, handles OAuth token loading/saving, fetches task lists and tasks.
- `taskwarrior/taskwarrior.go`: exports Taskwarrior tasks, filters pending tasks, and adds new tasks.
- `README.md`: only documented setup/run instructions.
- `go.mod`, `go.sum`: Go module metadata.

## Commands actually documented in this repo
These are the only project commands explicitly documented in the repository.

### Build and run
```bash
go build -o twgts twgts.go
./twgts
```

### Docker (documented, but see gotcha below)
```bash
docker build -t task-sync .
docker run -v $(pwd)/credentials.json:/data/credentials.json -v $(pwd)/token.json:/data/token.json -e GOOGLE_APPLICATION_CREDENTIALS=/data/credentials.json task-sync
```

## Runtime requirements observed in code/docs
- Go `1.26` is declared in `go.mod`.
- The program uses Google OAuth via `golang.org/x/oauth2` and `google.golang.org/api/tasks/v1`.
- The program shells out to the external `task` binary, so Taskwarrior must be installed and available on `PATH`.
- `GOOGLE_APPLICATION_CREDENTIALS` must point to the Google OAuth client credentials file. `googleTasks.NewGoogleTasksClient()` exits immediately if the env var is missing.
- OAuth callback flow listens on `http://localhost:8080/` and writes the access token to `token.json` in the working directory.

## Code organization and behavior
### Main flow
- `main()` in `twgts.go` only calls `SyncGoogleTasks()` right now.
- `SyncGoogleTasks()` creates the Google Tasks client and fetches tasks, then logs the number of retrieved tasks.
- A larger bidirectional sync flow exists in commented-out code in `twgts.go`; it is not active.

### Google Tasks integration
- `GoogleTasksService` is a thin wrapper around `*tasks.Service`.
- `GetTaskLists(filter string)` fetches up to 10 task lists and optionally filters by exact title match.
- `GetTasks(taskListID string)` fetches tasks from either all task lists or a specific list ID.
- OAuth token handling is local-file based (`token.json`).

### Taskwarrior integration
- `TaskWarriorClient.ListTasks()` runs `task export` and unmarshals JSON.
- `GetPendingTasks()` filters exported tasks by `status == "pending"`.
- `AddTask()`:
  - extracts `project=<name>` from Google task notes using a regex,
  - checks duplicates by exact title match against current pending Taskwarrior tasks,
  - runs `task add ...` through `sh -c`,
  - always adds the `GoogleTasks` tag,
  - passes through the Google due date string.

## Style and conventions observed
- Package names are lowercase (`googletasks`, `taskwarrior`), even when the directory uses camel case (`googleTasks/`).
- Structs use exported fields with JSON tags where they are part of CLI/API interchange (`TaskWarriorTask`).
- Error handling is inconsistent but heavily relies on `log.Fatal` / `log.Fatalf`, especially in `googleTasks/googleTasks.go`; many functions terminate the process instead of returning recoverable errors.
- Shell interaction is done with `os/exec`; there is no abstraction layer around external command execution.
- Duplicate detection is simplistic: exact title equality only.

## Testing and validation status
- No `*_test.go` files were found.
- No CI workflows were found under `.github/workflows/`.
- No Makefile, Taskfile, lint config, or repo-specific automation files were found.
- Validation in this repo is currently manual/integration-oriented rather than test-driven.

## Important gotchas
- `go.mod` declares the module as `gitlab.com/jcgutier/jcgutier/Golang/taskSyncPOC`, which does not match the current GitHub repository path.
- `README.md` says credentials can be placed in the working directory **or** provided through `GOOGLE_APPLICATION_CREDENTIALS`, but the code only supports the environment variable path and exits if it is unset.
- `README.md` documents Docker usage, but no `Dockerfile` exists in the repository.
- `googleTasks/googleTasks.go` contains a literal `# test if this work with a single task list` line inside Go code; this is not valid Go syntax and will break compilation until fixed.
- OAuth setup assumes localhost callback handling on port `8080`; anything else will need code changes.
- `token.json` is written to the current working directory, so run location matters.

## Guidance for future agents
- Read `README.md`, `twgts.go`, `googleTasks/googleTasks.go`, and `taskwarrior/taskwarrior.go` first; that is effectively the whole application.
- Be careful changing error handling: several code paths currently abort the process with `log.Fatal`, so behavior changes can be broad.
- If you touch sync logic, check both task directions: Google Tasks fetch and Taskwarrior add/export behavior.
- Any end-to-end verification will require real Google credentials, an OAuth browser flow, and a working Taskwarrior installation.
- Do not assume the Docker instructions work without first adding the missing `Dockerfile`.
