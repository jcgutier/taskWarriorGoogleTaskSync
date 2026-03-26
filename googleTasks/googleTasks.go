package googletasks

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/tasks/v1"
)

type GoogleTasksService struct {
	Service *tasks.Service
}

func NewGoogleTasksClient() (*GoogleTasksService, error) {
	ctx := context.Background()
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
	config, err := google.ConfigFromJSON(b, tasks.TasksScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)
	srv, err := tasks.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to retrieve tasks Client %v", err)
	}
	return &GoogleTasksService{Service: srv}, nil
}

func (c *GoogleTasksService) GetTaskLists(filter string) ([]*tasks.TaskList, error) {
	r, err := c.Service.Tasklists.List().Do()
	if err != nil {
		log.Fatalf("Unable to retrieve task lists. %v", err)
	}
	if filter != "" {
		var filteredTaskLists []*tasks.TaskList
		for _, item := range r.Items {
			if item.Title == filter {
				filteredTaskLists = append(filteredTaskLists, item)
			}
		}
		return filteredTaskLists, nil
	}
	return r.Items, nil
}

func (c *GoogleTasksService) GetTasks(taskListID string) ([]*tasks.Task, error) {
	allTasks := []*tasks.Task{}
	pageToken := ""
	taskLists := []*tasks.TaskList{}
	err := error(nil)
	taskLists, err = c.GetTaskLists(taskListID)
	if err != nil {
		log.Fatalf("Unable to retrieve task lists. %v", err)
		return nil, err
	}

	for _, item := range taskLists {
		for {
			r, err := c.Service.Tasks.List(item.Id).ShowCompleted(true).ShowDeleted(true).ShowHidden(true).PageToken(pageToken).Do()
			if err != nil {
				log.Fatalf("Unable to retrieve tasks for task list %s. %v", item.Title, err)
				return nil, err
			}
			allTasks = append(allTasks, r.Items...)
			// log.Printf("Tasks in list '%s': %d", item.Title, len(r.Items))
			pageToken = r.NextPageToken
			if pageToken == "" {
				break
			}
		}
	}
	return allTasks, nil
}

func (c *GoogleTasksService) AddTask(task *tasks.Task) (bool, error) {
	isTaskAdded := false
	tasksList, err := c.GetTaskLists("New")
	if err != nil {
		return false, fmt.Errorf("failed to get task lists: %w", err)
	}
	_, err = c.Service.Tasks.Insert(tasksList[0].Id, task).Do()
	if err != nil {
		return false, fmt.Errorf("failed to add task: %w", err)
	}
	isTaskAdded = true
	return isTaskAdded, nil
}

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
