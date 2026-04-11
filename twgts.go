package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"gitlab.com/jcgutier/jcgutier/Golang/taskSyncPOC/config"
	googletasks "gitlab.com/jcgutier/jcgutier/Golang/taskSyncPOC/googleTasks"
	postgressql "gitlab.com/jcgutier/jcgutier/Golang/taskSyncPOC/postgresSql"
	taskssync "gitlab.com/jcgutier/jcgutier/Golang/taskSyncPOC/tasksSync"
	"gitlab.com/jcgutier/jcgutier/Golang/taskSyncPOC/taskwarrior"
	"google.golang.org/api/tasks/v1"
)

func SyncGoogleTasks(cfg *config.Config) {

	sqlClient, err := postgressql.NewPostgresSqlClient(cfg.PostgresHost, cfg.PostgresPort, cfg.PostgresUser, cfg.PostgresPassword, cfg.PostgresDBName)
	if err != nil {
		log.Fatalf("Failed to create PostgreSQL client: %v", err)
	}
	isTableCreated, err := sqlClient.CreateTasksTable()
	if err != nil {
		log.Fatalf("Failed to create tasks table: %v", err)
	}
	log.Printf("Tasks table created: %v", isTableCreated)

	syncTask, err := taskssync.NewTasksSync(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize sync task: %v", err)
	}

	// TODO move this to tasksync.sync
	completedGTasks := 0
	needActionGTasks := 0
	for _, gTask := range syncTask.GoogleTasks {
		if gTask.Status == "completed" {
			completedGTasks++
		} else if gTask.Status == "needsAction" {
			needActionGTasks++
		} else {
			log.Printf("Google Task: %s, Status: %s, Due: %s", gTask.Title, gTask.Status, gTask.Due)
		}
	}

	log.Printf("Retrieved %d tasks from Google Tasks (%d completed, %d needs action).", len(syncTask.GoogleTasks), completedGTasks, needActionGTasks)
	log.Printf("Retrieved %d tasks from Task Warrior.", len(syncTask.TaskWarriorTasks))

	// log.Printf("Google tasks: ")
	// for _, task := range syncTask.GoogleTasks {
	// 	log.Printf(" - %s (%s)", task.Title, task.Status)
	// }

	taskWarriorClient := taskwarrior.TaskWarriorClient{
		DryRun: cfg.DryRun,
	}

	googleTasksClient, err := googletasks.NewGoogleTasksClient(cfg)
	if err != nil {
		log.Fatalf("Failed to create Google Tasks client: %v", err)
	}

	for googleTaskIndex, googleTask := range syncTask.GoogleTasks {
		var dbTask postgressql.SyncTask
		log.Printf("Processing Google Task(%d): %s (Status: %s, Due: %s, ID: %s)", googleTaskIndex, googleTask.Title, googleTask.Status, googleTask.Due, googleTask.Id)

		dbTasks, err := sqlClient.GetTasks(googleTask.Id, "")
		if err != nil {
			log.Printf("Failed to get tasks from database: %v", err)
		}
		if len(dbTasks) > 1 {
			log.Printf("Found multiple tasks in database taking the first one", googleTask.Id)
		}
		if len(dbTasks) > 0 {
			dbTask = dbTasks[0]
		}
		// log.Printf("DB task: %s (Status: %s, Due: %s, GID: %s, TID: %s)", dbTask.Title, dbTask.Status, dbTask.DUE, dbTask.GID, dbTask.TID)

		taskWarriorMatch := taskwarrior.TaskWarriorTask{}

		googleTaskDue, _ := time.Parse(time.RFC3339, googleTask.Due)
		googleTaskDueDate := googleTaskDue.Format("20060102")
		if dbTask.GID == "" {
			for _, taskWarriorTask := range syncTask.TaskWarriorTasks {
				taskWarriorTaskDue, _ := time.Parse("20060102T150405Z", taskWarriorTask.Due)
				if googleTask.Title == taskWarriorTask.Title && googleTaskDue.Equal(taskWarriorTaskDue) {
					log.Printf("Task Warrior match: %s (Status: %s, Due: %s, UUID: %s)", taskWarriorTask.Title, taskWarriorTask.Status, taskWarriorTask.Due, taskWarriorTask.UUID)
					taskWarriorMatch = taskWarriorTask
					if googleTask.Due == "" && taskWarriorTask.Due == "" {
						googleTask.Due = time.Unix(0, 0).Format(time.RFC3339)
					}
					dbTask := postgressql.SyncTask{
						GID:    googleTask.Id,
						TID:    taskWarriorTask.UUID,
						Title:  googleTask.Title,
						DUE:    googleTask.Due,
						Status: googleTask.Status,
					}
					err := sqlClient.AddTask(dbTask)
					if err != nil {
						log.Printf("Failed to add task '%s' to database: %v", googleTask.Title, err)
						log.Printf("Error when adding to DB: %v", err)
						return
					} else {
						log.Printf("Task '%s' added to database with Google Task ID '%s' and Taskwarrior UUID '%s'.", googleTask.Title, googleTask.Id, taskWarriorTask.UUID)
					}
					break
				}
			}
		} else {
			taskWarriorMatch = taskwarrior.TaskWarriorTask{
				Title: googleTask.Title,
				Due:   googleTask.Due,
				Notes: fmt.Sprintf("google_task_id=%s", googleTask.Id),
				UUID:  dbTask.TID,
			}
		}

		taskWarriorTaskDue, _ := time.Parse("20060102T150405Z", taskWarriorMatch.Due)
		taskWarriorTaskDueDate := taskWarriorTaskDue.Format("20060102")
		if taskWarriorMatch.Title == "" && googleTask.Status == "needsAction" {
			taskAdded, err := taskWarriorClient.AddTask(taskwarrior.TaskWarriorTask{
				Title: googleTask.Title,
				Notes: googleTask.Notes,
				Due:   googleTask.Due,
			})
			if err != nil {
				log.Printf("Failed to add '%s' with id '%s' and status '%s' to taskwarrior: %v", googleTask.Title, googleTask.Id, googleTask.Status, err)
			}
			if taskAdded.UUID != "" {
				log.Printf("Task '%s' added to taskwarrior with UUID '%s'.", googleTask.Title, taskAdded.UUID)
				dbTask := postgressql.SyncTask{
					GID:    googleTask.Id,
					TID:    taskAdded.UUID,
					Title:  googleTask.Title,
					DUE:    googleTask.Due,
					Status: googleTask.Status,
				}
				err := sqlClient.AddTask(dbTask)
				if err != nil {
					log.Printf("Failed to add task '%s' to database: %v", googleTask.Title, err)
				} else {
					log.Printf("Task '%s' added to database with Google Task ID '%s' and Taskwarrior UUID '%s'.", googleTask.Title, googleTask.Id, taskAdded.UUID)
				}
			}
		} else if googleTask.Status == "completed" {
			// log.Printf("Marking as complete the task warrior task with ID: %s", taskWarriorMatch.UUID)
			var twTaskStatus string
			for _, taskWarriorTask := range syncTask.TaskWarriorTasks {
				if taskWarriorTask.UUID == taskWarriorMatch.UUID {
					twTaskStatus = taskWarriorTask.Status
					break
				}
			}
			// log.Printf("Current task warrior task status: %s", twTaskStatus)
			if twTaskStatus != "completed" && taskWarriorMatch.UUID != "" {
				err := taskWarriorClient.CompleteTask(taskWarriorMatch.UUID)
				if err != nil {
					log.Printf("Failed to complete '%s'(%s) in taskwarrior: %v", googleTask.Title, taskWarriorMatch.UUID, err)
				}
				log.Printf("Task '%s' marked as completed in taskwarrior", googleTask.Title)
			}
			if dbTask.Status != "completed" {
				dbTask.Status = "completed"
				err := sqlClient.UpdateStatusTask(dbTask.TID, "completed")
				if err != nil {
					log.Printf("Failed to update task status in database: %v", err)
				}
			}
		} else if googleTaskDueDate != taskWarriorTaskDueDate {
			log.Printf("Updating due date for task '%s' in taskwarrior to match Google Task due date '%s'", googleTask.Title, googleTaskDueDate)
			taskWarriorClient.UpdateTaskDue(taskWarriorMatch, googleTaskDueDate)
			sqlClient.UpdateTask(postgressql.SyncTask{
				GID:   googleTask.Id,
				TID:   taskWarriorMatch.UUID,
				Title: googleTask.Title,
				DUE:   googleTask.Due,
			})

		}
		// TODO add logic when the project is different
	}

	log.Print("Syncronizing task warrior task to google tasks")
	for twTaskIndex, taskWarriorTask := range syncTask.TaskWarriorTasks {
		googleTaskMatch := tasks.Task{}
		log.Printf("Processing TW task[%d]: %s, (Status: %s, Due: %s, UUID: %s)", twTaskIndex, taskWarriorTask.Title, taskWarriorTask.Status, taskWarriorTask.Due, taskWarriorTask.UUID)
		dbTask, err := sqlClient.GetTasks("", taskWarriorTask.UUID)

		if err != nil {
			log.Printf("Failed to get task from database: %v", err)
		}
		if len(dbTask) > 0 {
			syncTask := dbTask[0]
			log.Printf("Task '%s' with Taskwarrior UUID '%s' already exists in database, skipping.", taskWarriorTask.Title, taskWarriorTask.UUID)
			if syncTask.Status != "completed" && taskWarriorTask.Status == "completed" {
				log.Printf("Marking as complete the google task with ID: %s", syncTask.GID)
				_, err := googleTasksClient.UpdateTask(&tasks.Task{
					Id:     syncTask.GID,
					Status: "completed",
				})
				if err != nil {
					log.Printf("Failed to complete '%s'(%s) in Google Tasks: %v", taskWarriorTask.Title, syncTask.GID, err)
				}
				log.Printf("Task '%s' marked as completed in Google Tasks", taskWarriorTask.Title)
				err = sqlClient.UpdateStatusTask(syncTask.TID, "completed")
				if err != nil {
					log.Printf("Failed to update task status in database: %v", err)
				}
			}
			continue
		} else {
			for _, googleTask := range syncTask.GoogleTasks {
				if googleTask.Title == taskWarriorTask.Title {
					log.Printf("Google Task match: %s (Status: %s, Due: %s, ID: %s)", googleTask.Title, googleTask.Status, googleTask.Due, googleTask.Id)
					googleTaskMatch = *googleTask
					break
				}
			}
			if googleTaskMatch.Title != "" {
				log.Printf("Task '%s' with Taskwarrior UUID '%s' already exists in Google Tasks with ID '%s', skipping.", taskWarriorTask.Title, taskWarriorTask.UUID, googleTaskMatch.Id)
				if googleTaskMatch.Status != "completed" && taskWarriorTask.Status == "completed" {
					log.Printf("Marking as complete the google task with ID: %s", googleTaskMatch.Id)
					_, err := googleTasksClient.UpdateTask(&tasks.Task{
						Id:     googleTaskMatch.Id,
						Status: "completed",
					})
					if err != nil {
						log.Printf("Failed to complete '%s'(%s) in Google Tasks: %v", taskWarriorTask.Title, googleTaskMatch.Id, err)
					}
					log.Printf("Task '%s' marked as completed in Google Tasks", taskWarriorTask.Title)
					err = sqlClient.UpdateStatusTask(taskWarriorTask.UUID, "completed")
					if err != nil {
						log.Printf("Failed to update task status in database: %v", err)
					}
				}
				continue
			}
		}

		gTask := tasks.Task{
			Title: taskWarriorTask.Title,
		}

		if taskWarriorTask.Due != "" {
			parsedDue, err := time.Parse("20060102T150405Z", taskWarriorTask.Due)
			if err != nil {
				log.Printf("Failed to parse due date '%s' for task '%s': %v", taskWarriorTask.Due, taskWarriorTask.Title, err)
			} else {
				gTask.Due = parsedDue.Format(time.RFC3339)
			}
		}
		if taskWarriorTask.Project != "" {
			gTask.Notes = fmt.Sprintf("project=%s", taskWarriorTask.Project)
		}
		if len(taskWarriorTask.Tags) > 0 {
			commaSeparatedTags := strings.Join(taskWarriorTask.Tags, ",")
			if gTask.Notes != "" {
				gTask.Notes += fmt.Sprintf(" tags=%s", commaSeparatedTags)
			} else {
				gTask.Notes = fmt.Sprintf("tags=%s", commaSeparatedTags)
			}
		}
		log.Printf("Adding task '%s' to Google Tasks.\n", taskWarriorTask.Title)
		newGTask, err := googleTasksClient.AddTask(&gTask)
		if err != nil {
			log.Printf("Failed to add task '%s' to Google Tasks: %v", taskWarriorTask.Title, err)
		}
		log.Printf("Google task created, title: %s, Status: %s, Due: %s, ID: %s", newGTask.Title, newGTask.Status, newGTask.Due, newGTask.Id)

		log.Printf("Adding task to DB with Google Task ID '%s' and Taskwarrior UUID '%s'.", newGTask.Id, taskWarriorTask.UUID)
		addDbTask := postgressql.SyncTask{
			GID:    newGTask.Id,
			TID:    taskWarriorTask.UUID,
			Title:  newGTask.Title,
			DUE:    newGTask.Due,
			Status: newGTask.Status,
		}
		err = sqlClient.AddTask(addDbTask)
		if err != nil {
			log.Printf("Failed to add task '%s' to database: %v", newGTask.Title, err)
		}
	}
	// TODO add logic to clean up the database by removing completed tasks
	log.Print("Sync completed.")
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	SyncGoogleTasks(cfg)

	// taskWarriorClient := taskwarrior.TaskWarriorClient{}
	// twPendingTasks, err := taskWarriorClient.GetPendingTasks() // Just to demonstrate usage of the TaskwarriorClient struct
	// if err != nil {
	// 	log.Fatalf("Failed to get pending tasks: %v", err)
	// }
	// log.Printf("Pending tasks in Taskwarrior: %d", len(twPendingTasks))

	// r, err := srv.Tasklists.List().MaxResults(10).Do()
	// if err != nil {
	// 	log.Fatalf("Unable to retrieve task lists. %v", err)
	// }

	// fmt.Println("Task Lists:")
	// if len(r.Items) > 0 {
	// 	for _, i := range r.Items {
	// 		fmt.Printf("%s (%s)\n", i.Title, i.Id)
	// 	}
	// } else {
	// 	fmt.Print("No task lists found.")
	// }

	// Get "Mis tareas" task list ID
	// var taskListID string
	// for _, item := range r.Items {
	// 	if item.Title == "Mis tareas" {
	// 		taskListID = item.Id
	// 		break
	// 	}
	// }

	// if taskListID == "" {
	// 	log.Fatalf("Task list 'Mis tareas' not found.")
	// }

	// fmt.Printf("\nTasks in 'Mis tareas':\n")
	// pageToken := ""
	// pageNum := 1

	// taskCount := 0
	// needActionTasks := 0
	// completedTasks := 0
	// taskWarriorAddedCount := 0
	// googleTasks := []tasks.Task{}
	// for {
	// 	var tasksResp *tasks.Tasks
	// 	var err error

	// 	if pageToken == "" {
	// 		tasksResp, err = srv.Tasks.List(taskListID).Do()
	// 	} else {
	// 		tasksResp, err = srv.Tasks.List(taskListID).PageToken(pageToken).Do()
	// 	}

	// 	if err != nil {
	// 		log.Fatalf("Unable to retrieve tasks: %v", err)
	// 	}

	// 	if len(tasksResp.Items) > 0 {
	// 		for _, task := range tasksResp.Items {
	// 			// fmt.Printf("- %s, %s, %s, %v\n", task.Title, task.Status, task.Due, task.Completed)
	// 			taskCount++

	// 			if task.Status == "needsAction" {
	// 				needActionTasks++

	// 				// add to taskwarrior
	// 				added, err := taskWarriorClient.AddTask(taskwarrior.TaskWarriorTask{
	// 					Title: task.Title,
	// 					Notes: task.Notes,
	// 					Due:   task.Due,
	// 				})
	// 				if err != nil {
	// 					log.Printf("warning: failed to add '%s' to taskwarrior: %v", task.Title, err)
	// 				}
	// 				if added {
	// 					taskWarriorAddedCount++
	// 				}
	// 			}
	// 			googleTasks = append(googleTasks, *task)
	// 			// if err := addToTaskwarrior(task.Title); err != nil {
	// 			// 	log.Printf("warning: failed to add '%s' to taskwarrior: %v", task.Title, err)
	// 			// }
	// 		}
	// 	}
	// 	pageNum++

	// 	// Check if there are more pages
	// 	pageToken = tasksResp.NextPageToken
	// 	if pageToken == "" {
	// 		break
	// 	}
	// 	time.Sleep(500 * time.Millisecond) // To avoid hitting rate limits
	// }

	// twAdded := 0
	// for twPendingIndex, twPending := range twPendingTasks {
	// 	index := 0
	// 	googleTask := tasks.Task{}
	// 	for index, googleTask = range googleTasks {
	// 		if twPending.Title == googleTask.Title {
	// 			// fmt.Printf("Task '%s' exists in both Taskwarrior and Google Tasks.\n", twPending.Title)
	// 			break
	// 		}
	// 	}
	// 	if index == len(googleTasks)-1 {
	// 		// fmt.Printf("Task '%s' exists in Taskwarrior but not in Google Tasks.\n", twPending.Title)
	// 		gTask := tasks.Task{
	// 			Title: twPending.Title,
	// 			Due:   twPending.Due,
	// 		}
	// 		if twPending.Project != "" {
	// 			googleTask.Notes = fmt.Sprintf("project=%s", twPending.Project)
	// 		}
	// 		_, err := srv.Tasks.Insert(taskListID, &gTask).Do()
	// 		if err != nil {
	// 			log.Printf("Failed to add task '%s' to Google Tasks: %v", twPending.Title, err)
	// 		}
	// 		fmt.Printf("[%d]Task '%s' added to Google Tasks.\n", twPendingIndex+1, twPending.Title)
	// 		twAdded++
	// 	}
	// }

	// fmt.Printf("\nTotal pages retrieved: %d\n", pageNum-1)
	// fmt.Printf("\nTotal tasks retrieved from 'Mis tareas': %d\n", taskCount)
	// fmt.Printf("Total tasks with status 'needsAction': %d\n", needActionTasks)
	// fmt.Printf("Total tasks with status 'completed': %d\n", completedTasks)
	// fmt.Printf("Total tasks added to Taskwarrior: %d\n", taskWarriorAddedCount)
	// fmt.Printf("Total tasks added to Google Tasks: %d\n", twAdded)

	// // Final message if no tasks at all
	// if pageNum == 1 {
	// 	fmt.Print("No tasks found in 'Mis tareas'.")
	// }
}

// addToTaskwarrior invokes the `task` command to add a new task with the
// supplied description. It returns an error if the command fails or is not
// installed.
func addToTaskwarrior(desc string) error {
	if desc == "" {
		return nil
	}
	// escape any double quotes
	d := strings.ReplaceAll(desc, "\"", "\\\"")
	cmd := exec.Command("task", "add", d)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
