package taskwarrior

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func installFakeTaskwarrior(t *testing.T, exportJSON string) (string, string) {
	t.Helper()

	tempDir := t.TempDir()
	argsFile := filepath.Join(tempDir, "task-args.txt")
	scriptPath := filepath.Join(tempDir, "task")
	script := "#!/bin/sh\n" +
		"set -eu\n" +
		"printf '%s\\n' \"$@\" > \"$TASK_TEST_ARGS_FILE\"\n" +
		"if [ \"${1:-}\" = \"export\" ]; then\n" +
		"  cat <<'EOF'\n" + exportJSON + "\nEOF\n" +
		"  exit 0\n" +
		"fi\n" +
		"exit 0\n"

	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write fake task executable: %v", err)
	}

	oldPath := os.Getenv("PATH")
	oldArgsFile := os.Getenv("TASK_TEST_ARGS_FILE")
	if err := os.Setenv("PATH", tempDir+string(os.PathListSeparator)+oldPath); err != nil {
		t.Fatalf("failed to set PATH: %v", err)
	}
	if err := os.Setenv("TASK_TEST_ARGS_FILE", argsFile); err != nil {
		t.Fatalf("failed to set TASK_TEST_ARGS_FILE: %v", err)
	}

	t.Cleanup(func() {
		_ = os.Setenv("PATH", oldPath)
		if oldArgsFile == "" {
			_ = os.Unsetenv("TASK_TEST_ARGS_FILE")
			return
		}
		_ = os.Setenv("TASK_TEST_ARGS_FILE", oldArgsFile)
	})

	return argsFile, scriptPath
}

func readArgsFile(t *testing.T, path string) []string {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read args file: %v", err)
	}
	trimmed := strings.TrimSpace(string(content))
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "\n")
}

func TestListTasksParsesExportedTasks(t *testing.T) {
	argsFile, _ := installFakeTaskwarrior(t, `[
		{"id":1,"description":"task one","status":"pending","due":"2026-03-18T00:00:00Z","project":"home"},
		{"id":2,"description":"task two","status":"completed"}
	]`)

	client := &TaskWarriorClient{}
	tasks, err := client.ListTasks()
	if err != nil {
		t.Fatalf("ListTasks returned error: %v", err)
	}

	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}
	if tasks[0].Title != "task one" {
		t.Fatalf("expected first task title %q, got %q", "task one", tasks[0].Title)
	}
	if tasks[0].Project != "home" {
		t.Fatalf("expected first task project %q, got %q", "home", tasks[0].Project)
	}

	args := readArgsFile(t, argsFile)
	if len(args) != 1 || args[0] != "export" {
		t.Fatalf("expected export command, got %v", args)
	}
}

func TestGetPendingTasksFiltersPendingEntries(t *testing.T) {
	installFakeTaskwarrior(t, `[
		{"description":"pending task","status":"pending"},
		{"description":"completed task","status":"completed"},
		{"description":"deleted task","status":"deleted"}
	]`)

	client := &TaskWarriorClient{}
	pending, err := client.GetPendingTasks()
	if err != nil {
		t.Fatalf("GetPendingTasks returned error: %v", err)
	}

	if len(pending) != 1 {
		t.Fatalf("expected 1 pending task, got %d", len(pending))
	}
	if pending[0].Title != "pending task" {
		t.Fatalf("expected pending task title %q, got %q", "pending task", pending[0].Title)
	}
}

func TestAddTaskSkipsDuplicatePendingTask(t *testing.T) {
	argsFile, _ := installFakeTaskwarrior(t, `[
		{"description":"duplicate task","status":"pending"}
	]`)

	client := &TaskWarriorClient{}
	added, err := client.AddTask(TaskWarriorTask{Title: "duplicate task", Due: "2026-03-18T00:00:00Z"})
	if err != nil {
		t.Fatalf("AddTask returned error: %v", err)
	}
	if added {
		t.Fatal("expected duplicate task not to be added")
	}

	args := readArgsFile(t, argsFile)
	if len(args) != 1 || args[0] != "export" {
		t.Fatalf("expected only export command for duplicate detection, got %v", args)
	}
}

func TestAddTaskAddsProjectAndGoogleTasksTag(t *testing.T) {
	argsFile, _ := installFakeTaskwarrior(t, `[]`)

	client := &TaskWarriorClient{}
	added, err := client.AddTask(TaskWarriorTask{
		Title: "new task",
		Notes: "project=work",
		Due:   "2026-03-18T00:00:00Z",
	})
	if err != nil {
		t.Fatalf("AddTask returned error: %v", err)
	}
	if !added {
		t.Fatal("expected task to be added")
	}

	args := readArgsFile(t, argsFile)
	joined := strings.Join(args, " ")
	for _, expected := range []string{"add", "project:work", "tags:GoogleTasks", "due:2026-03-18T00:00:00Z", "--", "new task"} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("expected command to contain %q, got %q", expected, joined)
		}
	}
}
