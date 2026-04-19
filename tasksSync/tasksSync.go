package taskssync

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"gitlab.com/jcgutier/jcgutier/Golang/taskSyncPOC/config"
	googletasks "gitlab.com/jcgutier/jcgutier/Golang/taskSyncPOC/googleTasks"
	"gitlab.com/jcgutier/jcgutier/Golang/taskSyncPOC/sqlite3"
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
	taskWarriorTasks, err := taskWarriorClient.GetTasks()
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
	sqlite3Client, err := sqlite3.NewSQLite3Client("")
	if err != nil {
		return fmt.Errorf("failed to initialize SQLite3 client: %w", err)
	}

	defer sqlite3Client.Db.Close()

	// Get pending tasks from Taskwarrior using the SQLite3 client
	pTasks, pTasksData, err := sqlite3Client.GetPendingTasks()
	if err != nil {
		return fmt.Errorf("failed to get pending tasks: %w", err)
	}
	log.Print("Pending tasks found on Taskwarrior: ", len(pTasks))

	// Create tasks on Google Tasks for each pending Taskwarrior task that doesn't exist in Google Tasks
	for taskIndex, taskUUID := range pTasks {
		taskDueInt, err := strconv.ParseInt(pTasksData[taskIndex].Due, 10, 64)
		if err != nil {
			log.Printf("Failed to parse due date for Taskwarrior UUID '%s': %v", taskUUID, err)
			continue
		}
		taskDue := time.Unix(taskDueInt, 0)
		taskDueDate := taskDue.Format(time.DateOnly)
		taskDueDateTime, _ := time.Parse("2006-01-02", taskDueDate)
		log.Printf("Processing Taskwarrior task UUID '%s' with project '%s', tags '%s', due '%s', and description '%s'", taskUUID, pTasksData[taskIndex].Project, pTasksData[taskIndex].Tags, taskDueDateTime.Format(time.RFC3339), pTasksData[taskIndex].Description)

		gTaskNotes := ""
		if pTasksData[taskIndex].Project != "" {
			gTaskNotes += fmt.Sprintf("project=%s ", pTasksData[taskIndex].Project)
		}
		if pTasksData[taskIndex].Tags != "" {
			if gTaskNotes == "" {
				gTaskNotes += fmt.Sprintf("tags=%s", pTasksData[taskIndex].Tags)
			} else {
				gTaskNotes += fmt.Sprintf(" tags=%s", pTasksData[taskIndex].Tags)
			}
		}

		gid, err := sqlite3Client.SearchGoogleTaskID(taskUUID)
		if err != nil {
			log.Printf("Failed to search Google Task ID for Taskwarrior UUID '%s': %v", taskUUID, err)
			continue
		}
		if gid != "" {
			// log.Printf("Taskwarrior UUID '%s' maps to Google Task ID '%s'", taskUUID, gid)
			var googleTask *tasks.Task
			for _, gTask := range s.GoogleTasks {
				if gTask.Id == gid {
					// log.Printf("Found matching Google Task with ID '%s' for Taskwarrior UUID '%s'", gid, taskUUID)
					googleTask = gTask
					break
				}
			}
			if googleTask == nil {
				log.Printf("No Google Task found with ID '%s' for Taskwarrior UUID '%s'", gid, taskUUID)
				continue
			}

			// Getting only task date to compare as seems there is a bug on Google Task API
			gTaskDue, err := time.Parse(time.RFC3339, googleTask.Due)
			if err != nil {
				log.Printf("Failed to parse google task due")
			}
			gTaskDueDate := gTaskDue.Format(time.DateOnly)

			needsUpdate := false
			reason := ""
			if googleTask.Status != "needsAction" {
				needsUpdate = true
				reason = "status is different"
			} else if googleTask.Title != pTasksData[taskIndex].Description {
				needsUpdate = true
				reason = "title is different"
			} else if googleTask.Notes != gTaskNotes {
				needsUpdate = true
				reason = "notes is different"
			} else if gTaskDueDate != taskDueDate {
				// } else if googleTask.Due != taskDue.Format(time.RFC3339) {
				needsUpdate = true
				reason = "due is different"
			}

			if needsUpdate {
				googleTask.Status = "needsAction"
				googleTask.Title = pTasksData[taskIndex].Description
				googleTask.Notes = gTaskNotes
				googleTask.Due = taskDueDateTime.Format(time.RFC3339)
				_, err = s.googleClient.UpdateTask(googleTask)
				if err != nil {
					log.Printf("Failed to update Google Task ID '%s' to 'needsAction' for Taskwarrior UUID '%s': %v", gid, taskUUID, err)
					continue
				}
				log.Printf("Updated Google Task ID '%s' because %s for Taskwarrior UUID '%s'", gid, reason, taskUUID)
			}
		} else {
			log.Printf("No Google Task mapping found for Taskwarrior UUID '%s'", taskUUID)
			addTask := &tasks.Task{
				Due:   taskDueDateTime.Format(time.RFC3339),
				Notes: fmt.Sprintf("project=%s tags=%s", pTasksData[taskIndex].Project, pTasksData[taskIndex].Tags),
				Title: pTasksData[taskIndex].Description,
			}
			addedTask, err := s.googleClient.AddTask(addTask)
			if err != nil {
				log.Printf("Failed to add Google Task for Taskwarrior UUID '%s': %v", taskUUID, err)
				continue
			}
			log.Printf("Successfully added Google Task with ID '%s'", addedTask.Id)
			sqlite3Client.InsertMapping(taskUUID, addedTask.Id)
			log.Printf("Inserted mapping for Taskwarrior UUID '%s' and Google Task ID '%s'", taskUUID, addedTask.Id)
		}
	}

	return nil

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
				if err := s.taskWarriorClient.CompleteTask(""); err != nil {
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
