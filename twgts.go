package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"gitlab.com/jcgutier/jcgutier/Golang/taskSyncPOC/taskwarrior"

	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/tasks/v1"
)

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	config.RedirectURL = "http://localhost:8080/"
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser: \n%v\n", authURL)

	codeCh := make(chan string)
	server := &http.Server{Addr: ":8080"}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code != "" {
			fmt.Fprintf(w, "Authorization successful! You can close this window.")
			codeCh <- code
		} else {
			fmt.Fprintf(w, "Authorization failed.")
		}
	})

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	authCode := <-codeCh
	server.Shutdown(context.Background())

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func main() {
	ctx := context.Background()

	taskWarriorClient := taskwarrior.TaskWarriorClient{}
	twPendingTasks, err := taskWarriorClient.GetPendingTasks() // Just to demonstrate usage of the TaskwarriorClient struct
	if err != nil {
		log.Fatalf("Failed to get pending tasks: %v", err)
	}
	log.Printf("Pending tasks in Taskwarrior: %d", len(twPendingTasks))

	credetialsFile := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	log.Printf("Using credentials file: %s", credetialsFile)
	if credetialsFile == "" {
		log.Fatal("Environment variable GOOGLE_APPLICATION_CREDENTIALS is not set.")
	}
	b, err := os.ReadFile(credetialsFile)
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, tasks.TasksReadonlyScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	srv, err := tasks.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to retrieve tasks Client %v", err)
	}

	r, err := srv.Tasklists.List().MaxResults(10).Do()
	if err != nil {
		log.Fatalf("Unable to retrieve task lists. %v", err)
	}

	fmt.Println("Task Lists:")
	if len(r.Items) > 0 {
		for _, i := range r.Items {
			fmt.Printf("%s (%s)\n", i.Title, i.Id)
		}
	} else {
		fmt.Print("No task lists found.")
	}

	// Get "Mis tareas" task list ID
	var taskListID string
	for _, item := range r.Items {
		if item.Title == "Mis tareas" {
			taskListID = item.Id
			break
		}
	}

	if taskListID == "" {
		log.Fatalf("Task list 'Mis tareas' not found.")
	}

	fmt.Printf("\nTasks in 'Mis tareas':\n")
	pageToken := ""
	pageNum := 1

	taskCount := 0
	needActionTasks := 0
	completedTasks := 0
	taskWarriorAddedCount := 0
	for {
		var tasksResp *tasks.Tasks
		var err error

		if pageToken == "" {
			tasksResp, err = srv.Tasks.List(taskListID).Do()
		} else {
			tasksResp, err = srv.Tasks.List(taskListID).PageToken(pageToken).Do()
		}

		if err != nil {
			log.Fatalf("Unable to retrieve tasks: %v", err)
		}

		if len(tasksResp.Items) > 0 {
			for _, task := range tasksResp.Items {
				fmt.Printf("- %s, %s, %s, %v\n", task.Title, task.Status, task.Due, task.Completed)
				taskCount++

				if task.Status == "needsAction" {
					needActionTasks++

					// add to taskwarrior
					added, err := taskWarriorClient.AddTask(taskwarrior.TaskWarriorTask{
						Title: task.Title,
						Notes: task.Notes,
						Due:   task.Due,
					})
					if err != nil {
						log.Printf("warning: failed to add '%s' to taskwarrior: %v", task.Title, err)
					}
					if added {
						taskWarriorAddedCount++
					}
				}
				// if err := addToTaskwarrior(task.Title); err != nil {
				// 	log.Printf("warning: failed to add '%s' to taskwarrior: %v", task.Title, err)
				// }
			}
		}
		pageNum++

		// Check if there are more pages
		pageToken = tasksResp.NextPageToken
		if pageToken == "" {
			break
		}
		time.Sleep(500 * time.Millisecond) // To avoid hitting rate limits
	}
	fmt.Printf("\nTotal pages retrieved: %d\n", pageNum-1)
	fmt.Printf("\nTotal tasks retrieved from 'Mis tareas': %d\n", taskCount)
	fmt.Printf("Total tasks with status 'needsAction': %d\n", needActionTasks)
	fmt.Printf("Total tasks with status 'completed': %d\n", completedTasks)
	fmt.Printf("Total tasks added to Taskwarrior: %d\n", taskWarriorAddedCount)

	// Final message if no tasks at all
	if pageNum == 1 {
		fmt.Print("No tasks found in 'Mis tareas'.")
	}
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
