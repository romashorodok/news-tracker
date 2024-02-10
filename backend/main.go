package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"

	_ "github.com/lib/pq"
	"github.com/romashorodok/news-tracker/backend/internal/storage"
)

type DatabaseConfig struct {
	Username string
	Password string
	Database string
	Host     string
	Port     string
	Driver   string
}

func (dconf *DatabaseConfig) GetURI() string {
	return fmt.Sprintf("%s://%s:%s@%s:%s/%s",
		dconf.Driver,
		dconf.Username,
		dconf.Password,
		dconf.Host,
		dconf.Port,
		dconf.Database,
	)
}

func NewDatabaseConfig() *DatabaseConfig {
	return &DatabaseConfig{
		Driver:   "postgres",
		Username: "admin",
		Password: "admin",
		Host:     "localhost",
		Port:     "5432",
		Database: "postgres",
	}
}

func WithTransaction(db *sql.DB, fn func(queries *storage.Queries) error) (err error) {
	tx, err := db.Begin()
	if err != nil {
		return
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		} else if err != nil {
			err = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	err = fn(storage.New(tx))
	return err
}

type NewDatabaseConnectionParams struct {
	Config *DatabaseConfig
}

func NewDatabaseConnection(params NewDatabaseConnectionParams) (*sql.DB, error) {
	return sql.Open(params.Config.Driver, params.Config.GetURI()+"?sslmode=disable")
}

func main() {
	databaseConfig := NewDatabaseConfig()

	conn, _ := NewDatabaseConnection(NewDatabaseConnectionParams{
		Config: databaseConfig,
	})
	defer conn.Close()

	store := storage.New(conn)

    _, err := store.GetArticleByID(context.Background(), 0)
	log.Println(errors.Is(err, sql.ErrNoRows))
}
