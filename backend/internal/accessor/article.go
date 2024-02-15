package accessor

import (
	"encoding/json"
	"errors"

	"github.com/romashorodok/news-tracker/backend/internal/model"
	"github.com/romashorodok/news-tracker/backend/internal/storage"
	"github.com/romashorodok/news-tracker/pkg/dateutils"
)

var ErrUnableGetArticle = errors.New("unable get article")

type articleRowImage struct {
	URL  string `json:"url"`
	Main bool   `json:"main"`
}

func ArticleFromArticleRows(row storage.ArticlesRow) (model.Article, error) {
	var images []articleRowImage
	if err := json.Unmarshal(row.Images, &images); err != nil {
		return model.NilArticle, ErrUnableGetArticle
	}

	article := model.Article{
		ID:           row.ID,
		Title:        row.Title,
		Preface:      row.Preface,
		Content:      row.Content,
		ViewersCount: row.ViewersCount,
		PublishedAt:  dateutils.Pretify(row.PublishedAt),
	}

	for _, image := range images {
		if image.Main {
			article.MainImage = image.URL
			continue
		}
		article.ContentImages = append(article.ContentImages, image.URL)
	}

	return article, nil
}

func ArticlesFromArticlesRows(rows []storage.ArticlesRow) ([]model.Article, error) {
	var articles []model.Article
	for _, row := range rows {
		article, err := ArticleFromArticleRows(row)
		if err != nil {
			return nil, err
		}
		articles = append(articles, article)
	}
	return articles, nil
}

func ArticleFromArticleByIdRow(row storage.GetArticleByIDRow) (model.Article, error) {
	var images []articleRowImage
	if err := json.Unmarshal(row.Images, &images); err != nil {
		return model.NilArticle, ErrUnableGetArticle
	}

	article := model.Article{
		ID:           row.ID,
		Title:        row.Title,
		Preface:      row.Preface,
		Content:      row.Content,
		ViewersCount: row.ViewersCount,
		PublishedAt:  dateutils.Pretify(row.PublishedAt),
	}

	for _, image := range images {
		if image.Main {
			article.MainImage = image.URL
			continue
		}
		article.ContentImages = append(article.ContentImages, image.URL)
	}

	return article, nil
}
