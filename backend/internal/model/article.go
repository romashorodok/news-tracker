package model

type Article struct {
	ID            int64    `json:"id"`
	Title         string   `json:"title"`
	Preface       string   `json:"preface"`
	Content       string   `json:"content"`
	ViewersCount  int32    `json:"viewers_count"`
	PublishedAt   string   `json:"published_at"`
	MainImage     string   `json:"main_image"`
	ContentImages []string `json:"content_images,omitempty"`
}

var NilArticle = Article{}
