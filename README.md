# TaskWarrior Google Tasks Sync

A Go program that synchronizes Google Tasks to Taskwarrior. It fetches incomplete tasks from the "Mis tareas" task list and adds them to Taskwarrior with appropriate tags and due dates.

## Quick Start

1. Create OAuth 2.0 Client ID credentials in Google Cloud Console (Web application). Set the redirect URI to `http://localhost:8080/`.

2. Download the `credentials.json` and place it in the working directory, or set the `GOOGLE_APPLICATION_CREDENTIALS` environment variable to its path.

3. Build and run:

   ```bash
   go build -o twgts twgts.go
   ./twgts
   ```

   The program will prompt for authentication if no token is saved. Follow the link in the browser, grant access, and the token will be saved to `token.json`.

4. The program will sync tasks from Google Tasks "Mis tareas" list to Taskwarrior, adding incomplete tasks with the tag `GoogleTasks`.

## Docker

```bash
# Build
docker build -t task-sync .

# Run (mount credentials and token)
docker run -v $(pwd)/credentials.json:/data/credentials.json -v $(pwd)/token.json:/data/token.json -e GOOGLE_APPLICATION_CREDENTIALS=/data/credentials.json task-sync
```

## Notes

- The app uses the Google Tasks readonly scope.
- Tasks are added to Taskwarrior with the tag `GoogleTasks`.
- If a task's notes contain `project=<name>`, it will be assigned to that project in Taskwarrior.
- Duplicate tasks (by title) are skipped.
- For production, secure the token storage.
