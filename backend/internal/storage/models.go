// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.25.0

package storage

import (
	"time"
)

type Article struct {
	ID           int64
	Title        string
	Preface      string
	Content      string
	Origin       string
	ViewersCount int32
	CreatedAt    time.Time
	UpdatedAt    time.Time
	PublishedAt  time.Time
}

type ArticleImage struct {
	ID        int64
	ArticleID int64
	Url       string
}
