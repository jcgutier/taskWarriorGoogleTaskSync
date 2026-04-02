package taskssync

import (
	"fmt"
	"log"

	"gitlab.com/jcgutier/jcgutier/Golang/taskSyncPOC/config"
	googletasks "gitlab.com/jcgutier/jcgutier/Golang/taskSyncPOC/googleTasks"
	"gitlab.com/jcgutier/jcgutier/Golang/taskSyncPOC/taskwarrior"
	"google.golang.org/api/tasks/v1"
)

type TasksSync struct {
	Config            *config.Config
	GoogleTasks       []*tasks.Task
	TaskWarriorTasks  []taskwarrior.TaskWarriorTask
	googleClient      *googletasks.GoogleTasksService
	taskWarriorClient *taskwarrior.TaskWarriorClient
}

func NewTasksSync(cfg *config.Config) (*TasksSync, error) {
	googleTaskClient, err := googletasks.NewGoogleTasksClient(cfg)
	if err != nil {
		return nil, err
	}
	googleTasks, err := googleTaskClient.GetTasks(cfg.GoogleTaskListFilter)
	if err != nil {
		return nil, err
	}

	taskWarriorClient := &taskwarrior.TaskWarriorClient{DryRun: cfg.DryRun}
	taskWarriorTasks, err := taskWarriorClient.ListTasks()
	if err != nil {
		return nil, err
	}

	return &TasksSync{
		Config:            cfg,
		GoogleTasks:       googleTasks,
		TaskWarriorTasks:  taskWarriorTasks,
		googleClient:      googleTaskClient,
		taskWarriorClient: taskWarriorClient,
	}, nil
}

func (s *TasksSync) Sync() error {
	taskWarriorByTitle := map[string]taskwarrior.TaskWarriorTask{}
	for _, tw := range s.TaskWarriorTasks {
		taskWarriorByTitle[tw.Title] = tw
	}

	googleByTitle := map[string]*tasks.Task{}
	for _, gt := range s.GoogleTasks {
		googleByTitle[gt.Title] = gt
	}

	for _, googleTask := range s.GoogleTasks {
		twTask, found := taskWarriorByTitle[googleTask.Title]
		switch googleTask.Status {
		case "completed":
			if found && twTask.Status == "pending" && twTask.ID != 0 {
				log.Printf("Task '%s' exists in Google Tasks as completed and pending in Taskwarrior, completing Taskwarrior task ID %d", googleTask.Title, twTask.ID)
				if err := s.taskWarriorClient.CompleteTask(twTask.ID); err != nil {
					return fmt.Errorf("failed to complete taskwarrior task %d: %w", twTask.ID, err)
				}
			}
		case "needsAction":
			if !found {
				log.Printf("Task '%s' exists in Google Tasks but not Taskwarrior; adding to Taskwarrior", googleTask.Title)
				if _, err := s.taskWarriorClient.AddTask(taskwarrior.TaskWarriorTask{
					Title: googleTask.Title,
					Notes: googleTask.Notes,
					Due:   googleTask.Due,
				}); err != nil {
					return fmt.Errorf("failed to add taskwarrior task for '%s': %w", googleTask.Title, err)
				}
			}
		}
	}

	for _, taskWarriorTask := range s.TaskWarriorTasks {
		if _, found := googleByTitle[taskWarriorTask.Title]; found {
			continue
		}

		newGoogleTask := tasks.Task{
			Title: taskWarriorTask.Title,
		}
		if taskWarriorTask.Project != "" {
			newGoogleTask.Notes = fmt.Sprintf("project=%s", taskWarriorTask.Project)
		}

		log.Printf("Task '%s' exists in Taskwarrior but not Google Tasks; adding to Google Tasks", taskWarriorTask.Title)
		if _, err := s.googleClient.AddTask(&newGoogleTask); err != nil {
			return fmt.Errorf("failed to add google task for '%s': %w", taskWarriorTask.Title, err)
		}
	}

	return nil
}
