package taskwarrior

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"regexp"
)

type TaskWarriorTask struct {
	ID      int    `json:"id"`
	End     string `json:"end"`
	Title   string `json:"description"`
	Status  string `json:"status"`
	Due     string `json:"due"`
	Notes   string
	Project string `json:"project,omitempty"`
}

type TaskWarriorClient struct {
	DryRun bool
}

func (t *TaskWarriorClient) ListTasks() ([]TaskWarriorTask, error) {
	cmd := exec.Command("task", "export")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute task export: %w", err)
	}

	var tasks []TaskWarriorTask
	err = json.Unmarshal(output, &tasks)
	if err != nil {
		return nil, fmt.Errorf("failed to parse task export output: %w", err)
	}

	return tasks, nil
}

func (t *TaskWarriorClient) GetPendingTasks() ([]TaskWarriorTask, error) {
	tasks, err := t.ListTasks()
	if err != nil {
		return nil, err
	}

	var pendingTasks []TaskWarriorTask
	for _, task := range tasks {
		if task.Status == "pending" {
			pendingTasks = append(pendingTasks, task)
		}
	}

	return pendingTasks, nil
}

func (t *TaskWarriorClient) AddTask(task TaskWarriorTask) (bool, error) {
	var taskProject string

	// Get project from description if it exists
	re := regexp.MustCompile(`project=([^\s]+)`)
	matches := re.FindStringSubmatch(task.Notes)
	if len(matches) > 1 {
		taskProject = matches[1]
	} else {
		taskProject = ""
	}

	// Check if task exists in taskwarrior
	existingTasks, err := t.GetPendingTasks()
	if err != nil {
		return false, fmt.Errorf("failed to get pending tasks: %w", err)
	}
	for _, existingTask := range existingTasks {
		if existingTask.Title == task.Title {
			// fmt.Printf("Task '%s' already exists in taskwarrior, skipping.\n", task.Title)
			return false, nil
		}
	}

	execCommand := ""
	if taskProject != "" {
		execCommand = fmt.Sprintf("task add project:%s tags:GoogleTasks due:%s -- %s", taskProject, task.Due, task.Title)
	} else {
		execCommand = fmt.Sprintf("task add tags:GoogleTasks due:%s -- %s", task.Due, task.Title)
	}

	cmd := exec.Command("sh", "-c", execCommand)
	_, err = cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to add task: %w", err)
	}
	return true, nil
}

func (t *TaskWarriorClient) CompleteTask(taskID int) error {
	if t.DryRun {
		log.Printf("[Dry Run] Would complete task with ID: %d", taskID)
		return nil
	}
	cmd := exec.Command("task", fmt.Sprintf("%d", taskID), "done")
	_, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to complete task: %w", err)
	}
	return nil
}
