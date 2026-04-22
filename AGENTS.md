# Repository Guidelines

## Project Structure & Module Organization
`twgts.go` is the main entrypoint. Core packages live in `config/`, `googleTasks/`, `taskwarrior/`, `tasksSync/`, `sqlite3/`, and `postgresSql/`. Tests are colocated with the code they cover, for example `googleTasks/googleTasks_test.go` and `taskwarrior/taskwarrior_test.go`. Runtime and developer docs live in `README.md`, `PLAN.md`, and `CLAUDE.md`. Container-related files are `Dockerfile`, `docker-compose.yml`, and `supervisord.conf`. Helper scripts belong in `scripts/`.

## Build, Test, and Development Commands
Build the binary with:

```bash
go build -o twgts twgts.go
```

Run locally with Google credentials and an optional token path:

```bash
GOOGLE_APPLICATION_CREDENTIALS=/path/credentials.json \
GOOGLE_TASKS_TOKEN_PATH=/path/token.json \
./twgts
```

Run the full test suite with `go test ./...`. For faster iteration, target one package, for example `go test ./taskwarrior ./googleTasks`. Start the local PostgreSQL service with `docker compose up -d` and stop it with `docker compose down`.

## Coding Style & Naming Conventions
Use standard Go formatting with `gofmt -w` before committing. Follow Go naming conventions: exported identifiers use `CamelCase`, unexported helpers use `camelCase`, and tests use `TestXxx`. Keep packages focused on one integration concern. Prefer small functions, explicit error wrapping with `%w`, and environment-driven configuration over hardcoded paths.

## Testing Guidelines
This repository uses Go's built-in `testing` package. Add table-driven tests where multiple input cases are involved. Keep tests adjacent to the package under test and name files `*_test.go`. Cover sync logic, config parsing, and command invocation behavior when changing integrations. Run `go test ./...` before opening a PR.

## Commit & Pull Request Guidelines
Recent history uses short `fix:`-prefixed commit subjects, for example `fix: Adding login to sync files`. Keep commit titles imperative, concise, and scoped to one change. Pull requests should include a brief summary, any required env vars or migration steps, and the exact validation performed, such as `go test ./...` or a local sync run.

## Security & Configuration Tips
Do not commit `token.json`, OAuth credentials, or `.env` secrets. Use `GOOGLE_APPLICATION_CREDENTIALS`, `GOOGLE_TASKS_TOKEN_PATH`, and `CONFIG_FILE_PATH` to point to local files outside the repo when possible.
