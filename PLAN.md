# Plan: Run twgts as a Continuous Docker Container

## Context

The sync logic in `twgts.go:SyncGoogleTasks()` is fully implemented but `main()` calls it once and exits. The Dockerfile and supervisord.conf already exist and are correctly configured for daemon use. The goal is to wire up a periodic loop, fix the missing Postgres env-var overrides, and complete the docker-compose file so the whole stack (Postgres + twgts) runs as a single `docker compose up`.

## Issues Found

1. **No daemon loop** — `main()` calls `SyncGoogleTasks(cfg)` once then exits (line 301).
2. **Postgres config not env-overridable** — `config/config.go:applyEnvOverrides()` has no cases for `POSTGRES_HOST/PORT/USER/PASSWORD/DB`. Values only come from the JSON file, so the Compose container can't override `POSTGRES_HOST=postgres` via env.
3. **docker-compose.yml missing twgts service** — only Postgres is defined.
4. **`.env` key mismatch** — file uses `POSTGRES_DB=twgts` but `docker-compose.yml` references `${POSTGRES_DB_NAME}`, so Postgres container never reads the DB name correctly.

## Changes

### 1. `config/config.go` — Add Postgres env overrides

Add to `applyEnvOverrides()` after the existing block:

```go
if val, ok := os.LookupEnv("POSTGRES_HOST"); ok && val != "" {
    cfg.PostgresHost = val
}
if val, ok := os.LookupEnv("POSTGRES_PORT"); ok && val != "" {
    if parsed, err := strconv.Atoi(val); err == nil {
        cfg.PostgresPort = parsed
    }
}
if val, ok := os.LookupEnv("POSTGRES_USER"); ok && val != "" {
    cfg.PostgresUser = val
}
if val, ok := os.LookupEnv("POSTGRES_PASSWORD"); ok && val != "" {
    cfg.PostgresPassword = val
}
if val, ok := os.LookupEnv("POSTGRES_DB_NAME"); ok && val != "" {
    cfg.PostgresDBName = val
}
```

Also add Postgres defaults to `defaultConfig()`:
```go
PostgresHost:     "localhost",
PostgresPort:     5432,
PostgresUser:     "twgts",
PostgresPassword: "twgts_password",
PostgresDBName:   "twgts",
```

### 2. `twgts.go` — Replace `main()` with a daemon loop

Replace lines 293–436 (the entire `main()` + commented-out dead code) with:

```go
func main() {
    log.SetFlags(log.LstdFlags | log.Lshortfile)

    cfg, err := config.LoadConfig()
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    interval := time.Duration(cfg.SyncIntervalSeconds) * time.Second
    log.Printf("Starting sync daemon (interval: %v)", interval)

    // Run once immediately so the first sync isn't delayed.
    SyncGoogleTasks(cfg)

    ticker := time.NewTicker(interval)
    defer ticker.Stop()

    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)

    for {
        select {
        case <-ticker.C:
            SyncGoogleTasks(cfg)
        case sig := <-quit:
            log.Printf("Received signal %v, shutting down.", sig)
            return
        }
    }
}
```

Add `"os/signal"` and `"syscall"` to the import block (`"time"` is already imported).
Keep the `addToTaskwarrior` helper below `main()` — it's used by tests.

### 3. `docker-compose.yml` — Add twgts service

Append to `services:` and add `taskwarrior_data` named volume:

```yaml
  twgts:
    build: .
    container_name: twgts-app
    restart: unless-stopped
    depends_on:
      - postgres
    env_file:
      - .env
    environment:
      POSTGRES_HOST: postgres   # override .env; resolves via Docker internal DNS
    volumes:
      - ${GOOGLE_APPLICATION_CREDENTIALS}:/credentials/credentials.json:ro
      - ${GOOGLE_TASKS_TOKEN_PATH:-/home/carlos/.shared_config/taskSync/token.json}:/credentials/token.json:ro
      - taskwarrior_data:/root/.task
    ports:
      - "9090:9090"

volumes:
  pgdata:
  taskwarrior_data:
```

### 4. `.env` — Fix key name and add missing vars

Change `POSTGRES_DB` → `POSTGRES_DB_NAME` (fixes docker-compose.yml Postgres container), and add token path and interval:

```
POSTGRES_USER=twgts
POSTGRES_PASSWORD=twgts_password
POSTGRES_DB_NAME=twgts
GOOGLE_APPLICATION_CREDENTIALS=/home/carlos/.shared_config/taskSync/credentials.json
GOOGLE_TASKS_TOKEN_PATH=/home/carlos/.shared_config/taskSync/token.json
SYNC_INTERVAL_SECONDS=300
```

## Pre-flight: Generate token.json (once, on host)

The container can't do interactive OAuth. Generate the token on the host first:

```bash
GOOGLE_APPLICATION_CREDENTIALS=/home/carlos/.shared_config/taskSync/credentials.json \
GOOGLE_TASKS_TOKEN_PATH=/home/carlos/.shared_config/taskSync/token.json \
POSTGRES_HOST=localhost \
  go run twgts.go
```

This writes `token.json` at the path above; the container mounts it read-only thereafter.

## Verification

```bash
# Build and start the full stack
docker compose up -d

# Tail logs — should see "Starting sync daemon" then sync output
docker compose logs -f twgts

# Confirm Postgres gets the mappings
docker compose exec postgres psql -U twgts -d twgts -c "SELECT COUNT(*) FROM tasks;"

# Test graceful shutdown
docker compose stop twgts   # supervisord sends SIGTERM; should log "Received signal"
```
