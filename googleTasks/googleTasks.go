package googletasks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"gitlab.com/jcgutier/jcgutier/Golang/taskSyncPOC/config"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/tasks/v1"
)

type GoogleTasksService struct {
	Service *tasks.Service
	Config  *config.Config
}

func NewGoogleTasksClient(cfg *config.Config) (*GoogleTasksService, error) {
	ctx := context.Background()
	credentialsFile := cfg.GoogleCredentialsPath
	if credentialsFile == "" {
		credentialsFile = os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	}
	if credentialsFile == "" {
		return nil, fmt.Errorf("GOOGLE_APPLICATION_CREDENTIALS must be set either by env or config")
	}

	b, err := os.ReadFile(credentialsFile)
	if err != nil {
		return nil, fmt.Errorf("unable to read client secret file: %w", err)
	}

	googleConfig, err := google.ConfigFromJSON(b, tasks.TasksScope)
	if err != nil {
		return nil, fmt.Errorf("unable to parse client secret file to config: %w", err)
	}
	tokFile := cfg.GoogleTokenPath
	if tokFile == "" {
		tokFile = "token.json"
	}

	client := getClient(googleConfig, tokFile)
	srv, err := tasks.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve tasks client: %w", err)
	}
	return &GoogleTasksService{Service: srv, Config: cfg}, nil
}

func (c *GoogleTasksService) GetTaskLists(filter string) ([]*tasks.TaskList, error) {
	r, err := c.Service.Tasklists.List().Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve task lists: %w", err)
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

func (c *GoogleTasksService) GetTasks(taskListFilter string) ([]*tasks.Task, error) {
	if taskListFilter == "" {
		taskListFilter = c.Config.GoogleTaskListFilter
	}
	allTasks := []*tasks.Task{}
	pageToken := ""

	taskLists, err := c.GetTaskLists(taskListFilter)
	if err != nil {
		return nil, err
	}

	for _, item := range taskLists {
		for {
			r, err := c.Service.Tasks.List(item.Id).ShowCompleted(true).ShowDeleted(true).ShowHidden(true).PageToken(pageToken).Do()
			if err != nil {
				return nil, fmt.Errorf("unable to retrieve tasks for task list %s: %w", item.Title, err)
			}
			allTasks = append(allTasks, r.Items...)
			pageToken = r.NextPageToken
			if pageToken == "" {
				break
			}
		}
	}
	return allTasks, nil
}

func (c *GoogleTasksService) AddTask(task *tasks.Task) (bool, error) {
	filter := c.Config.GoogleTaskListFilter
	tasksList, err := c.GetTaskLists(filter)
	if err != nil {
		return false, fmt.Errorf("failed to get task lists: %w", err)
	}
	if len(tasksList) == 0 {
		return false, fmt.Errorf("no task list found matching filter '%s'", filter)
	}
	_, err = c.Service.Tasks.Insert(tasksList[0].Id, task).Do()
	if err != nil {
		return false, fmt.Errorf("failed to add task: %w", err)
	}
	return true, nil
}

func getClient(config *oauth2.Config, tokenFile string) *http.Client {
	tok, err := tokenFromFile(tokenFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokenFile, tok)
	}
	return config.Client(context.Background(), tok)
}

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
			panic(err)
		}
	}()

	authCode := <-codeCh
	server.Shutdown(context.Background())

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		panic(err)
	}
	return tok
}

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

func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}
