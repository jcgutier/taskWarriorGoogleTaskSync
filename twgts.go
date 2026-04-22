package main

import (
	"log"

	"gitlab.com/jcgutier/jcgutier/Golang/taskSyncPOC/config"
	taskssync "gitlab.com/jcgutier/jcgutier/Golang/taskSyncPOC/tasksSync"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	taskSync, err := taskssync.NewTasksSync(cfg)
	if err != nil {
		log.Fatalf("Failed to create tasks sync: %v", err)
	}
	err = taskSync.Sync()
	if err != nil {
		log.Fatalf("Failed to sync tasks: %v", err)
	}
}
