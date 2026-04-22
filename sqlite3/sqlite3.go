package sqlite3

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"

	// The underscore import registers the driver with database/sql
	_ "github.com/mattn/go-sqlite3"
)

type SQLite3Client struct {
	Db *sql.DB
}

type TaskData struct {
	Project     string `json:"project"`
	Tags        string `json:"tags"`
	Due         string `json:"due"`
	Description string `json:"description"`
}

func NewSQLite3Client(dbPath string) (*SQLite3Client, error) {
	if dbPath == "" {
		dbPath = fmt.Sprintf("%s/Dropbox/.taskwarrior/taskchampion.sqlite3", os.Getenv("HOME"))
		log.Printf("%s", dbPath)
	}
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	const createTableSQL = `CREATE TABLE IF NOT EXISTS goTasksSync (
		uuid TEXT PRIMARY KEY,
		gid TEXT
	);`

	_, err = db.Exec(createTableSQL)
	if err != nil {
		return nil, err
	}
	// rowsAffected, err := sqlResult.RowsAffected()
	// if err != nil {
	// 	return nil, err
	// }
	// log.Printf("Table created or already exists, rows affected: %d", rowsAffected)
	return &SQLite3Client{Db: db}, nil
}

func (c *SQLite3Client) GetPendingTasks() (map[string]TaskData, error) {
	rows, err := c.Db.Query("SELECT * FROM tasks WHERE json_extract(data, '$.status') = 'pending'")
	if err != nil {
		return nil, err
	}

	tasksMap := make(map[string]TaskData)
	for rows.Next() {
		var uuid string
		var taskData string
		var taskDataStruct TaskData
		err := rows.Scan(&uuid, &taskData) // We only care about the uuid, so we can ignore the data column
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal([]byte(taskData), &taskDataStruct)
		if err != nil {
			log.Printf("Failed to unmarshal task data for UUID '%s': %v", uuid, err)
			continue
		}
		tasksMap[uuid] = taskDataStruct
	}
	return tasksMap, nil
}

func (c *SQLite3Client) GetCompletedTasks() ([]string, []TaskData, error) {
	rows, err := c.Db.Query("SELECT * FROM tasks WHERE json_extract(data, '$.status') = 'completed'")
	if err != nil {
		return nil, nil, err
	}

	var tasks []string
	var tasksData []TaskData
	for rows.Next() {
		var uuid string
		var taskData string
		var taskDataStruct TaskData
		err := rows.Scan(&uuid, &taskData)
		if err != nil {
			return nil, nil, err
		}
		tasks = append(tasks, uuid)
		err = json.Unmarshal([]byte(taskData), &taskDataStruct)
		if err != nil {
			log.Printf("Failed to unmarshal task data for UUID '%s': %v", uuid, err)
			continue
		}
		tasksData = append(tasksData, taskDataStruct)
	}
	return tasks, tasksData, nil
}

func (c *SQLite3Client) SearchGoogleTaskID(taskWarriorUUID string) (string, error) {
	var googleTaskID string
	err := c.Db.QueryRow("SELECT gid FROM goTasksSync WHERE uuid = ?", taskWarriorUUID).Scan(&googleTaskID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil // No mapping found, return empty string
		}
		return "", err // Some other error occurred
	}
	return googleTaskID, nil
}

func (c *SQLite3Client) SearchTaskWarriorTaskID(googleTaksId string) (string, error) {
	var taskWarriorTaskID string
	err := c.Db.QueryRow("SELECT uuid FROM goTasksSync WHERE gid = ?", googleTaksId).Scan(&taskWarriorTaskID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return taskWarriorTaskID, nil
}

func (c *SQLite3Client) InsertMapping(taskWarriorUUID, googleTaskID string) error {
	_, err := c.Db.Exec("INSERT INTO goTasksSync (uuid, gid) VALUES (?, ?)", taskWarriorUUID, googleTaskID)
	return err
}
