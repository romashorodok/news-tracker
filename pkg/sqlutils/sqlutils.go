package sqlutils

import (
	"database/sql"
	"time"
)

var null_time = time.Time{}

func GetNullableSqlTime(u time.Time) sql.NullTime {
	if null_time.Equal(u) {
		return sql.NullTime{Valid: false}
	}
	return sql.NullTime{Time: u, Valid: true}
}
