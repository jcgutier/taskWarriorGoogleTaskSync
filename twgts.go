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

	dbTasks, err := sqlClient.GetTasks()
	if err != nil {
		log.Fatalf("Failed to get tasks from database: %v", err)
	}
	log.Printf("Retrieved %d tasks from database.", len(dbTasks))

	if len(dbTasks) < 0 {
		log.Printf("No tasks found in database, adding sample task.")
	}

	syncTask, err := taskssync.NewTasksSync(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize sync task: %v", err)
	}

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
		// TODO run more test on creation and matching tasks
		if googleTaskIndex == 1 {
			log.Println("Stopped on 3rd task for testing purposes")
			break
		}

		dbTaskMatch := postgressql.SyncTask{}
		for _, dbTask := range dbTasks {
			if googleTask.Id == dbTask.GID {
				log.Printf("Task '%s' with Google Task ID '%s' already exists in database, skipping.\n", googleTask.Title, googleTask.Id)
				dbTaskMatch = dbTask
				break
			}
		}
		taskWarriorMatch := taskwarrior.TaskWarriorTask{}
		log.Printf("Processing Google Task(%d): %s (Status: %s, Due: %s, ID: %s)", googleTaskIndex, googleTask.Title, googleTask.Status, googleTask.Due, googleTask.Id)

		if dbTaskMatch.GID == "" {
			for _, taskWarriorTask := range syncTask.TaskWarriorTasks {
				googleTaskDue, _ := time.Parse(time.RFC3339, googleTask.Due)
				taskWarriorTaskDue, _ := time.Parse("20060102T150405Z", taskWarriorTask.Due)
				if googleTask.Title == taskWarriorTask.Title {
					log.Printf("Found task with matching title")
					log.Printf("Google Task: %s, status: %s, original_due: %v, %v, parsed_due: %v, ID: %v", googleTask.Title, googleTask.Status, googleTask.Due, googleTaskDue, googleTaskDue, googleTask.Id)
					log.Printf("Taskwr Task: %s, status: %s, original_due %v, parsed_due: %v, ID: %s", taskWarriorTask.Title, taskWarriorTask.Status, taskWarriorTask.Due, taskWarriorTaskDue, taskWarriorTask.UUID)
					log.Printf("Time match ? %v", googleTaskDue.Equal(taskWarriorTaskDue))
				}
				if googleTask.Title == taskWarriorTask.Title && googleTaskDue.Equal(taskWarriorTaskDue) {
					log.Printf("Found matching task")
					taskWarriorMatch = taskWarriorTask
					break
				}
			}

			// TODO add task to database, and change the logic below for these values
		} else {
			taskWarriorMatch = taskwarrior.TaskWarriorTask{
				Title: googleTask.Title,
				Due:   googleTask.Due,
				Notes: fmt.Sprintf("google_task_id=%s", googleTask.Id),
			}
		}
		switch googleTask.Status {
		case "completed":
			if taskWarriorMatch.Title != "" && taskWarriorMatch.ID != 0 {
				log.Printf("Task '%s' with due date %s is completed in Google Tasks but pending in Taskwarrior, completing it in Taskwarrior.\n", googleTask.Title, googleTask.Due)
				// complete in taskwarrior
				err := taskWarriorClient.CompleteTask(taskWarriorMatch.ID)
				if err != nil {
					log.Printf("Failed to complete '%s'(%d) in taskwarrior: %v", googleTask.Title, taskWarriorMatch.ID, err)
				}
			}
		case "needsAction":
			if taskWarriorMatch.Title != "" {
				log.Printf("Task '%s' already exists in Taskwarrior, skipping.\n", googleTask.Title)
				break
			}
			// add to taskwarrior
			_, err := taskWarriorClient.AddTask(taskwarrior.TaskWarriorTask{
				Title: googleTask.Title,
				Notes: googleTask.Notes,
				Due:   googleTask.Due,
			})
			if err != nil {
				log.Printf("Failed to add '%s' with id '%s' and status '%s' to taskwarrior: %v", googleTask.Title, googleTask.Id, googleTask.Status, err)
			}
			log.Printf("Task %s added to taskwarrior", googleTask.Title)
		}

	}

	for _, taskWarriorTask := range syncTask.TaskWarriorTasks {
		googleTaskMatch := &tasks.Task{}

		for _, googleTask := range syncTask.GoogleTasks {
			if taskWarriorTask.Title == googleTask.Title {
				// log.Printf("Task '%s' exists in both Taskwarrior and Google Tasks.\n", taskWarriorTask.Title)
				googleTaskMatch = googleTask
				break
			}
		}

		if googleTaskMatch.Title == "" {
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
			log.Printf("Adding task '%s' to Google Tasks.\n", taskWarriorTask.Title)
			isTaskAdded, err := googleTasksClient.AddTask(&gTask)
			if err != nil {
				log.Printf("Failed to add task '%s' to Google Tasks: %v", taskWarriorTask.Title, err)
			}
			if isTaskAdded {
				log.Printf("Task '%s' added to Google Tasks.", taskWarriorTask.Title)
			}
		}
	}
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
