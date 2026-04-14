package postgressql

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq" // Register the postgres driver
)

type PostgresSqlClient struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	db       *sql.DB
}

func NewPostgresSqlClient(host string, port int, user string, password string, dbName string) (*PostgresSqlClient, error) {
	postgresClient := &PostgresSqlClient{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
		DBName:   dbName,
	}
	err := postgresClient.Connect()
	if err != nil {
		log.Printf("Error while connecting, %v", err)
		return nil, err
	}

	return postgresClient, nil
}

func (psc *PostgresSqlClient) Connect() error {
	var err error

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		psc.Host, psc.Port, psc.User, psc.Password, psc.DBName)

	psc.db, err = sql.Open("postgres", dsn)
	if err != nil {
		return err
	}

	return nil
}

func (psc *PostgresSqlClient) CreateTasksTable() (bool, error) {
	createTableQuery := `
	CREATE TABLE IF NOT EXISTS tasks (
		id SERIAL PRIMARY KEY,
		gid TEXT,
		tid TEXT NOT NULL,
		title TEXT NOT NULL,
		due_date TIMESTAMP,
		status TEXT
	);
	`

	// log.Printf("Creating tasks table with query: %s", createTableQuery)

	_, err := psc.db.Exec(createTableQuery)
	return true, err
}

func (psc *PostgresSqlClient) GetTasks(gid string, tid string) ([]SyncTask, error) {
	getTasksQuery := `SELECT gid, tid, title, due_date, status FROM tasks`
	if gid != "" {
		getTasksQuery = fmt.Sprintf(`%s WHERE gid = '%s';`, getTasksQuery, gid)
	} else if tid != "" {
		getTasksQuery = fmt.Sprintf(`%s WHERE tid = '%s';`, getTasksQuery, tid)
	} else {
		getTasksQuery = fmt.Sprintf(`%s;`, getTasksQuery)
	}
	// log.Printf("Getting tasks with query: %s", getTasksQuery)

	err := psc.db.Ping()
	if err != nil {
		log.Printf("Error: %v", err)
		err := psc.Connect()
		if err != nil {
			return nil, err
		}
	}

	rows, err := psc.db.Query(getTasksQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := []SyncTask{}
	for rows.Next() {
		var task SyncTask
		err := rows.Scan(&task.GID, &task.TID, &task.Title, &task.DUE, &task.Status)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

func (psc *PostgresSqlClient) AddTask(task SyncTask) error {
	addTaskQuery := `INSERT INTO tasks (gid, tid, title, due_date, status) VALUES ($1, $2, $3, $4, $5);`
	log.Printf("Adding task with query: %s", addTaskQuery)

	_, err := psc.db.Exec(addTaskQuery, task.GID, task.TID, task.Title, task.DUE, task.Status)
	return err
}

func (psc *PostgresSqlClient) UpdateTask(task SyncTask) error {
	updateTaskQuery := `UPDATE tasks SET gid = $1, title = $2, due_date = $3, status = $4 WHERE tid = $5;`
	log.Printf("Updating task with query: %s", updateTaskQuery)

	_, err := psc.db.Exec(updateTaskQuery, task.GID, task.Title, task.DUE, task.Status, task.TID)
	return err
}

func (psc *PostgresSqlClient) DeleteTask(taskID string) error {
	// TODO implement logic to delete a task from the database
	return nil
}

func (psc *PostgresSqlClient) UpdateStatusTask(taskID string, status string) error {
	updateStatusTaskQuery := `UPDATE tasks SET status = $1 WHERE tid = $2;`
	// log.Printf("Updating task status with query: %s", updateStatusTaskQuery)

	_, err := psc.db.Exec(updateStatusTaskQuery, status, taskID)
	return err
}

type SyncTask struct {
	GID    string
	TID    string
	DUE    string
	Title  string
	Status string
}
