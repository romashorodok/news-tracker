package txutils

import (
	"database/sql"

	"github.com/romashorodok/news-tracker/backend/internal/storage"
)

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
