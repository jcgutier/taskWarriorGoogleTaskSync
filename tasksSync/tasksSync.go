package taskssync

import (
	"log"

	googletasks "gitlab.com/jcgutier/jcgutier/Golang/taskSyncPOC/googleTasks"
	"gitlab.com/jcgutier/jcgutier/Golang/taskSyncPOC/taskwarrior"
	"google.golang.org/api/tasks/v1"
)

type TasksSync struct {
	TaskWarriorTasks []taskwarrior.TaskWarriorTask
	GoogleTasks      []*tasks.Task
}

func NewTasksSync() *TasksSync {
	// Get Google Tasks
	googleTaskClient, err := googletasks.NewGoogleTasksClient()
	if err != nil {
		log.Fatalf("Failed to create Google Tasks client: %v", err)
	}
	googleTasks, err := googleTaskClient.GetTasks("")
	if err != nil {
		log.Fatalf("Failed to get Google Tasks: %v", err)
	}

	// Get Taskwarrior tasks
	tadkWarriorClient := taskwarrior.TaskWarriorClient{}
	taskWarriorTasks, err := tadkWarriorClient.ListTasks()
	if err != nil {
		log.Fatalf("Failed to get pending tasks from Taskwarrior: %v", err)
	}

	return &TasksSync{
		GoogleTasks:      googleTasks,
		TaskWarriorTasks: taskWarriorTasks,
	}
}
