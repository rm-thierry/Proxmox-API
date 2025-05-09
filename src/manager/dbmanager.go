package manager

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

type DBManager struct {
	db *sql.DB
}

type DBConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
}

func NewDBManager(config DBConfig) (*DBManager, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		config.User,
		config.Password,
		config.Host,
		config.Port,
		config.DBName,
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("error opening database connection: %v", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("error testing database connection: %v", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)

	return &DBManager{db: db}, nil
}

func (m *DBManager) Close() error {
	return m.db.Close()
}

func (m *DBManager) Query(query string, args ...interface{}) (*sql.Rows, error) {
	rows, err := m.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("error executing query: %v", err)
	}
	return rows, nil
}

func (m *DBManager) QueryRow(query string, args ...interface{}) *sql.Row {
	return m.db.QueryRow(query, args...)
}

func (m *DBManager) Exec(query string, args ...interface{}) (sql.Result, error) {
	result, err := m.db.Exec(query, args...)
	if err != nil {
		return nil, fmt.Errorf("error executing statement: %v", err)
	}
	return result, nil
}

func (m *DBManager) Prepare(query string) (*sql.Stmt, error) {
	stmt, err := m.db.Prepare(query)
	if err != nil {
		return nil, fmt.Errorf("error preparing statement: %v", err)
	}
	return stmt, nil
}

func (m *DBManager) GetDB() *sql.DB {
	return m.db
}
