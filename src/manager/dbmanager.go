package manager

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

type DBManager struct {
	db *sql.DB
}

func NewDBManager(user, password, dbname string) (*DBManager, error) {
	dsn := fmt.Sprintf("%s:%s@/%s", user, password, dbname)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &DBManager{db: db}, nil
}

func (manager *DBManager) Close() error {
	return manager.db.Close()
}

func (manager *DBManager) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return manager.db.Query(query, args...)
}

func (manager *DBManager) Exec(query string, args ...interface{}) (sql.Result, error) {
	return manager.db.Exec(query, args...)
}

func (manager *DBManager) Prepare(query string) (*sql.Stmt, error) {
	return manager.db.Prepare(query)
}

func main() {
	manager, err := NewDBManager("username", "password", "dbname")
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer manager.Close()

	// Example usage
	rows, err := manager.Query("SELECT * FROM tablename")
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		// Process rows
	}
}
