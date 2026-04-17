package taskwarrior

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var taskWarriorInfoLineRegexp = regexp.MustCompile(`^(.+?)[ \t]{2,}(.*)$`)

type TaskWarriorTask struct {
	ID       int    `json:"id"`
	End      string `json:"end"`
	Title    string `json:"description"`
	Status   string `json:"status"`
	Due      string `json:"due"`
	UUID     string `json:"uuid"`
	Notes    string
	Project  string   `json:"project,omitempty"`
	Tags     []string `json:"tags,omitempty"`
	Modified string   `json:"modified,omitempty"`
}

type TaskWarriorClient struct {
	DryRun bool
}

func (t *TaskWarriorClient) GetTasks() ([]TaskWarriorTask, error) {
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

func (t *TaskWarriorClient) ListTasks() ([]TaskWarriorTask, error) {
	return t.GetTasks()
}

func (t *TaskWarriorClient) GetTaskInfo(taskID string) (TaskWarriorTask, error) {
	cmd := exec.Command("task", taskID, "info")
	output, err := cmd.Output()
	if err != nil {
		return TaskWarriorTask{}, fmt.Errorf("failed to execute task info: %w", err)
	}

	return ParseTaskWarriorInfoOutput(string(output))
}

func ParseTaskWarriorInfoOutput(output string) (TaskWarriorTask, error) {
	task := TaskWarriorTask{}

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Name") || strings.HasPrefix(line, "---") {
			continue
		}

		key, value := parseTaskWarriorInfoLine(line)
		if key == "" {
			continue
		}

		switch strings.ToLower(key) {
		case "id":
			parsedID, err := strconv.Atoi(value)
			if err != nil {
				return task, fmt.Errorf("invalid id in task info output: %w", err)
			}
			task.ID = parsedID
		case "description":
			task.Title = value
		case "status":
			task.Status = value
		case "due":
			task.Due = value
		case "uuid":
			task.UUID = value
		case "project":
			task.Project = value
		case "tags":
			if value != "" {
				task.Tags = strings.Fields(value)
			}
		case "notes":
			task.Notes = value
		}
	}

	if task.ID == 0 && task.UUID == "" {
		return task, fmt.Errorf("failed to parse task info output")
	}

	return task, nil
}

func parseTaskWarriorInfoLine(line string) (string, string) {
	matches := taskWarriorInfoLineRegexp.FindStringSubmatch(line)
	if len(matches) != 3 {
		return "", ""
	}
	return strings.TrimSpace(matches[1]), strings.TrimSpace(matches[2])
}

func (t *TaskWarriorClient) GetPendingTasks() ([]TaskWarriorTask, error) {
	tasks, err := t.GetTasks()
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

func (t *TaskWarriorClient) AddTask(task TaskWarriorTask) (TaskWarriorTask, error) {
	var taskProject string
	var createdTask TaskWarriorTask

	// Get project from description if it exists
	re := regexp.MustCompile(`project=([^\s]+)`)
	matches := re.FindStringSubmatch(task.Notes)
	if len(matches) > 1 {
		taskProject = matches[1]
	} else {
		taskProject = ""
	}

	// Get tags from description if they exist
	task.Tags = append(task.Tags, "GoogleTasks")
	reTags := regexp.MustCompile(`tags=([^\s]+)`)
	matchesTags := reTags.FindStringSubmatch(task.Notes)
	if len(matchesTags) > 1 {
		task.Tags = append(task.Tags, matchesTags[1])
	}
	taskTags := strings.Join(task.Tags, ",")

	// Check if task exists in taskwarrior
	existingTasks, err := t.GetPendingTasks()
	if err != nil {
		return TaskWarriorTask{}, fmt.Errorf("failed to get pending tasks: %w", err)
	}
	for _, existingTask := range existingTasks {
		if existingTask.Title == task.Title {
			log.Printf("Task '%s' already exists in taskwarrior ID: %s, skipping.\n", task.Title, existingTask.UUID)
			return TaskWarriorTask{}, nil
		}
	}

	// TODO run more test here as the time seems not to be matching
	dueTime, err := time.Parse(time.RFC3339, task.Due)
	if err != nil {
		return TaskWarriorTask{}, fmt.Errorf("failed to parse due date: %w", err)
	}
	dueDate := dueTime.Format("2006-01-02")

	execCommand := ""
	if taskProject != "" {
		execCommand = fmt.Sprintf("task add project:%s tags:%s due:%s -- '%s'", taskProject, taskTags, dueDate, task.Title)
	} else {
		execCommand = fmt.Sprintf("task add tags:%s due:%s -- '%s'", taskTags, dueDate, task.Title)
	}
	log.Printf("Running exec command: %s", execCommand)

	if t.DryRun {
		log.Printf("[Dry Run] Would add task: %s", task.Title)
		log.Println(execCommand)
		return TaskWarriorTask{}, nil
	}

	cmd := exec.Command("sh", "-c", execCommand)
	output, err := cmd.Output()
	if err != nil {
		return TaskWarriorTask{}, fmt.Errorf("failed to add task: %w", err)
	}
	log.Printf("Task added %s", string(output))
	tagID := regexp.MustCompile(`Created task (\d+)`)
	matchedId := tagID.FindStringSubmatch(string(output))
	if len(matchedId) > 1 {
		createdTask, err = t.GetTaskInfo(matchedId[1])
		if err != nil {
			log.Printf("Failed to get info for created task: %v", err)
		}
	} else {
		log.Printf("Task '%s' added to taskwarrior but failed to parse ID from output: %s", task.Title, string(output))
	}
	// TODO get the ID of the created task and return it
	return createdTask, nil
}

func (t *TaskWarriorClient) CompleteTask(taskID string) error {
	if t.DryRun {
		log.Printf("[Dry Run] Would complete task with ID: %s", taskID)
		return nil
	}
	execCommand := fmt.Sprintf("task %s done", taskID)
	log.Printf("Running exec command: %s", execCommand)
	cmd := exec.Command("sh", "-c", execCommand)
	output, err := cmd.Output()
	log.Printf("Output: %s", output)
	if err != nil {
		return fmt.Errorf("failed to complete task: %v", err)
	}
	return nil
}

func (t *TaskWarriorClient) UpdateTaskDue(task TaskWarriorTask, newDue string) error {
	if t.DryRun {
		log.Printf("[Dry Run] Would update due date for task '%s' to '%s'", task.Title, newDue)
		return nil
	}
	execCommand := fmt.Sprintf("task %s modify due:%s", task.UUID, newDue)
	log.Printf("Running exec command: %s", execCommand)
	cmd := exec.Command("sh", "-c", execCommand)
	output, err := cmd.Output()
	log.Printf("Output: %s", output)
	if err != nil {
		return fmt.Errorf("failed to update task due date: %v", err)
	}
	return nil
}

func (t *TaskWarriorClient) PurgeTask(taskID string) error {
	if t.DryRun {
		log.Printf("[Dry Run] Would purge task with ID: %s", taskID)
		return nil
	}
	execCommand := fmt.Sprintf("task %s purge rc.confirmation=off", taskID)
	log.Printf("Running exec command: %s", execCommand)
	cmd := exec.Command("sh", "-c", execCommand)
	_, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to purge task: %v", err)
	}
	return nil
}

func (t *TaskWarriorClient) DeleteTask(taskID string) error {
	if t.DryRun {
		log.Printf("[Dry Run] Would delete task with ID: %s", taskID)
		return nil
	}
	execCommand := fmt.Sprintf("task %s delete rc.confirmation=off", taskID)
	log.Printf("Running exec command: %s", execCommand)
	cmd := exec.Command("sh", "-c", execCommand)
	_, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to delete task: %v", err)
	}
	return nil
}
