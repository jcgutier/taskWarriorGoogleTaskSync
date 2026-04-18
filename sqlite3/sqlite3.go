package sqlite3

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	// The underscore import registers the driver with database/sql
	_ "github.com/mattn/go-sqlite3"
)

type SQLite3Client struct {
	Db *sql.DB
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

func (c *SQLite3Client) GetPendingTasks() ([]string, error) {
	rows, err := c.Db.Query("SELECT * FROM tasks WHERE json_extract(data, '$.status') = 'pending'")
	if err != nil {
		return nil, err
	}

	var tasks []string
	for rows.Next() {
		var task string
		if err := rows.Scan(&task); err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	log.Print("Pending tasks found: ", len(tasks))
	return tasks, nil
}
