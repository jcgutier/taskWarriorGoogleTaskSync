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
		log.Printf("Failed to create google task client")
		return nil, err
	}
	googleTasks, err := googleTaskClient.GetTasks(cfg.GoogleTaskListFilter)
	if err != nil {
		return nil, err
	}

	taskWarriorClient := &taskwarrior.TaskWarriorClient{DryRun: cfg.DryRun}

	return &TasksSync{
		Config:            cfg,
		GoogleTasks:       googleTasks,
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
	pTasks, err := sqlite3Client.GetPendingTasks()
	if err != nil {
		return fmt.Errorf("failed to get pending tasks: %w", err)
	}
	log.Print("Pending tasks found on Taskwarrior: ", len(pTasks))

	// Create tasks on Google Tasks for each pending Taskwarrior task that doesn't exist in Google Tasks
	for taskUUID, taskData := range pTasks {
		taskDueInt, err := strconv.ParseInt(taskData.Due, 10, 64)
		if err != nil {
			log.Printf("Failed to parse due date for Taskwarrior UUID '%s': %v", taskUUID, err)
			continue
		}
		taskDue := time.Unix(taskDueInt, 0)
		taskDueDate := taskDue.Format(time.DateOnly)
		taskDueDateTime, _ := time.Parse("2006-01-02", taskDueDate)
		log.Printf("Processing pending Taskwarrior task UUID '%s' with project '%s', tags '%s', due '%s', and description '%s'", taskUUID, taskData.Project, taskData.Tags, taskDueDateTime.Format(time.RFC3339), taskData.Description)

		gTaskNotes := ""
		if taskData.Project != "" {
			gTaskNotes += fmt.Sprintf("project=%s ", taskData.Project)
		}
		if taskData.Tags != "" {
			if gTaskNotes == "" {
				gTaskNotes += fmt.Sprintf("tags=%s", taskData.Tags)
			} else {
				gTaskNotes += fmt.Sprintf(" tags=%s", taskData.Tags)
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
				log.Printf("No Google Task found with ID '%s' for Taskwarrior UUID '%s', investigate why this is happening", gid, taskUUID)
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
			} else if googleTask.Title != taskData.Description {
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
				googleTask.Title = taskData.Description
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
				Notes: fmt.Sprintf("project=%s tags=%s", taskData.Project, taskData.Tags),
				Title: taskData.Description,
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

	cTasks, cTasksData, err := sqlite3Client.GetCompletedTasks()
	if err != nil {
		log.Printf("Failed to get completed tasks. Error: %v", err)
	}
	for taskIndex, taskUUID := range cTasks {
		log.Printf("Processing completed Taskwarrior task UUID '%s' with project '%s', tags '%s', due '%s', and description '%s'", taskUUID, cTasksData[taskIndex].Project, cTasksData[taskIndex].Tags, "", cTasksData[taskIndex].Description)
		gid, err := sqlite3Client.SearchGoogleTaskID(taskUUID)
		if err != nil {
			log.Printf("Failed to search Google Task ID for Taskwarrior UUID '%s': %v", taskUUID, err)
			continue
		}
		if gid != "" {
			var googleTask *tasks.Task
			for _, gTask := range s.GoogleTasks {
				if gTask.Id == gid {
					// log.Printf("Found matching Google Task with ID '%s' for Taskwarrior UUID '%s'", gid, taskUUID)
					googleTask = gTask
					break
				}
			}
			if googleTask == nil {
				log.Printf("No Google Task found with ID '%s' for Taskwarrior UUID '%s', investigate why this is happening", gid, taskUUID)
				continue
			}
			googleTask.Status = "completed"
			_, err = s.googleClient.UpdateTask(googleTask)
			if err != nil {
				log.Printf("Failed to update google task ID: %s", googleTask.Id)
			} else {
				log.Printf("Google task Id: %s status updated to completed", googleTask.Id)
			}
		}
	}

	completedGTasks := 0
	needActionGTasks := 0
	for _, gTask := range s.GoogleTasks {
		switch gTask.Status {
		case "completed":
			completedGTasks++
		case "needsAction":
			needActionGTasks++
		default:
			log.Printf("Unknown, pls check: Google Task: %s, Status: %s, Due: %s", gTask.Title, gTask.Status, gTask.Due)
		}
	}

	for googleTaskIndex, googleTask := range s.GoogleTasks {
		log.Printf("Processing Google Task(%d): %s (Status: %s, Due: %s, ID: %s)", googleTaskIndex, googleTask.Title, googleTask.Status, googleTask.Due, googleTask.Id)

		tid, err := sqlite3Client.SearchTaskWarriorTaskID(googleTask.Id)
		if err != nil {
			log.Printf("Error while searching the task warrior ID from google task id %s. Error %v", googleTask.Id, err)
		}

		taskWarriorMatch := taskwarrior.TaskWarriorTask{}
		googleTaskDue, _ := time.Parse(time.RFC3339, googleTask.Due)
		googleTaskDueDate := googleTaskDue.Format("20060102")
		if tid == "" {
			for _, taskWarriorTask := range s.TaskWarriorTasks {
				taskWarriorTaskDue, _ := time.Parse("20060102T150405Z", taskWarriorTask.Due)
				if googleTask.Title == taskWarriorTask.Title && googleTaskDue.Equal(taskWarriorTaskDue) {
					log.Printf("Task Warrior match: %s (Status: %s, Due: %s, UUID: %s)", taskWarriorTask.Title, taskWarriorTask.Status, taskWarriorTask.Due, taskWarriorTask.UUID)
					taskWarriorMatch = taskWarriorTask
					if googleTask.Due == "" && taskWarriorTask.Due == "" {
						googleTask.Due = time.Unix(0, 0).Format(time.RFC3339)
					}
					err = sqlite3Client.InsertMapping(taskWarriorMatch.UUID, googleTask.Id)
					if err != nil {
						log.Printf("Failed to insert mapping with values: tid %s, gid %s. Error: %v", taskWarriorMatch.UUID, googleTask.Id, err)
					} else {
						log.Printf("Mapping inserted to DB with values: tid %s, gid %s.", taskWarriorMatch.UUID, googleTask.Id)
					}
					break
				}
			}
		} else {
			taskWarriorMatch = taskwarrior.TaskWarriorTask{
				Title: pTasks[tid].Description,
				Due:   pTasks[tid].Due,
				UUID:  tid,
			}
		}

		taskWarriorTaskDue, _ := time.Parse("20060102T150405Z", taskWarriorMatch.Due)
		taskWarriorTaskDueDate := taskWarriorTaskDue.Format("20060102")
		if taskWarriorMatch.Title == "" && googleTask.Status == "needsAction" {
			taskAdded, err := s.taskWarriorClient.AddTask(taskwarrior.TaskWarriorTask{
				Title: googleTask.Title,
				Notes: googleTask.Notes,
				Due:   googleTask.Due,
			})
			if err != nil {
				log.Printf("Failed to add '%s' with id '%s' and status '%s' to taskwarrior: %v", googleTask.Title, googleTask.Id, googleTask.Status, err)
			}
			if taskAdded.UUID != "" {
				log.Printf("Task '%s' added to taskwarrior with UUID '%s'.", googleTask.Title, taskAdded.UUID)
				err = sqlite3Client.InsertMapping(taskAdded.UUID, googleTask.Id)
				if err != nil {
					log.Printf("Failed to insert mapping to sqlite DB for values: tid: %s, gid: %s. Error: %v", taskAdded.UUID, googleTask.Id, err)
				}
			}
		} else if googleTask.Status == "completed" {
			// log.Printf("Marking as complete the task warrior task with ID: %s", taskWarriorMatch.UUID)
			var twTaskStatus string
			for _, taskWarriorTask := range s.TaskWarriorTasks {
				if taskWarriorTask.UUID == taskWarriorMatch.UUID {
					twTaskStatus = taskWarriorTask.Status
					break
				}
			}
			// log.Printf("Current task warrior task status: %s", twTaskStatus)
			if twTaskStatus != "completed" && taskWarriorMatch.UUID != "" {
				err := s.taskWarriorClient.CompleteTask(taskWarriorMatch.UUID)
				if err != nil {
					log.Printf("Failed to complete '%s'(%s) in taskwarrior: %v", googleTask.Title, taskWarriorMatch.UUID, err)
				}
				log.Printf("Task '%s' marked as completed in taskwarrior", googleTask.Title)
			}
		} else if googleTaskDueDate != taskWarriorTaskDueDate {
			log.Printf("Updating due date for task '%s' in taskwarrior to match Google Task due date '%s', previous date: '%s'", googleTask.Title, googleTaskDueDate, googleTaskDueDate)
			s.taskWarriorClient.UpdateTaskDue(taskWarriorMatch, googleTaskDueDate)
		}
	}

	deletedTWTasks := 0
	recurringTWTasks := 0
	taskWarriorTasks := len(cTasks) + len(pTasks)

	log.Printf("Retrieved %d tasks from Google Tasks (%d completed, %d needs action).", len(s.GoogleTasks), completedGTasks, needActionGTasks)
	log.Printf("Retrieved %d tasks from Task Warrior (%d completed, %d pending, %d deleted, %d recurring).", taskWarriorTasks, len(cTasks), len(pTasks), deletedTWTasks, recurringTWTasks)

	return nil
}
