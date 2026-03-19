package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func installFakeTaskBinary(t *testing.T) string {
	t.Helper()

	tempDir := t.TempDir()
	argsFile := filepath.Join(tempDir, "task-args.txt")
	scriptPath := filepath.Join(tempDir, "task")
	script := "#!/bin/sh\n" +
		"set -eu\n" +
		"printf '%s\\n' \"$@\" > \"$TASK_TEST_ARGS_FILE\"\n"
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

	return argsFile
}

func TestAddToTaskwarriorSkipsEmptyDescription(t *testing.T) {
	oldPath := os.Getenv("PATH")
	if err := os.Setenv("PATH", t.TempDir()); err != nil {
		t.Fatalf("failed to set PATH: %v", err)
	}
	defer os.Setenv("PATH", oldPath)

	if err := addToTaskwarrior(""); err != nil {
		t.Fatalf("expected nil error for empty description, got %v", err)
	}
}

func TestAddToTaskwarriorInvokesTaskBinary(t *testing.T) {
	argsFile := installFakeTaskBinary(t)

	if err := addToTaskwarrior("say \"hello\""); err != nil {
		t.Fatalf("addToTaskwarrior returned error: %v", err)
	}

	content, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("failed to read args file: %v", err)
	}
	args := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(args) != 2 {
		t.Fatalf("expected 2 arguments, got %v", args)
	}
	if args[0] != "add" {
		t.Fatalf("expected first argument %q, got %q", "add", args[0])
	}
	if args[1] != "say \\\"hello\\\"" {
		t.Fatalf("expected escaped description, got %q", args[1])
	}
}
