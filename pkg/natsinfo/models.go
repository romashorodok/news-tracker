package natsinfo

import (
	"encoding/json"
	"time"

	"github.com/romashorodok/news-tracker/pkg/dateutils"
)

type Article struct {
	Title         string
	Preface       string
	Content       string
	PublishedAt   time.Time
	ViewersCount  int
	MainImage     string
	ContentImages []string
	Origin        string
}

type articleDTO struct {
	Title         string   `json:"title"`
	Preface       string   `json:"preface"`
	Content       string   `json:"content"`
	PublishedAt   string   `json:"published_at"`
	ViewersCount  int      `json:"viewers_count"`
	MainImage     string   `json:"main_image"`
	ContentImages []string `json:"content_images,omitempty"`
	Origin        string   `json:"origin"`
}

func (a *Article) Marshal() ([]byte, error) {
	return json.Marshal(
		&articleDTO{
			Title:         a.Title,
			Preface:       a.Preface,
			Content:       a.Content,
			PublishedAt:   dateutils.ToString(a.PublishedAt),
			ViewersCount:  a.ViewersCount,
			MainImage:     a.MainImage,
			ContentImages: a.ContentImages,
			Origin:        a.Origin,
		},
	)
}

func (a *Article) Unmarshal(data []byte) error {
	var dto articleDTO

	if err := json.Unmarshal(data, &dto); err != nil {
		return err
	}

	a.Title = dto.Title
	a.Preface = dto.Preface
	a.Content = dto.Content
	a.ViewersCount = dto.ViewersCount
	a.MainImage = dto.MainImage
	a.ContentImages = dto.ContentImages
	a.Origin = dto.Origin

	time, err := dateutils.ParseString(dto.PublishedAt)
	if err != nil {
		return err
	}
	a.PublishedAt = time

	return nil
}
