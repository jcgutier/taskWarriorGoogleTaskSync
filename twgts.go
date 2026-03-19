package main

import (
	"log"
	"os"
	"os/exec"
	"strings"

	taskssync "gitlab.com/jcgutier/jcgutier/Golang/taskSyncPOC/tasksSync"
)

func SyncGoogleTasks() {
	syncTask := taskssync.NewTasksSync()
	log.Printf("Retrieved %d tasks from Google Tasks.", len(syncTask.GoogleTasks))
	log.Printf("Retrieved %d pending tasks from Taskwarrior.", len(syncTask.TaskWarriorTasks))

	log.Printf("Google tasks: ")
	for _, task := range syncTask.GoogleTasks {
		log.Printf(" - %s (%s)", task.Title, task.Status)
	}

	// TODO create logic for bidirectional sync
}

func main() {

	SyncGoogleTasks()

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
