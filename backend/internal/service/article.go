package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	nats "github.com/nats-io/nats.go"
	"github.com/romashorodok/news-tracker/backend/internal/accessor"
	"github.com/romashorodok/news-tracker/backend/internal/model"
	"github.com/romashorodok/news-tracker/backend/internal/storage"
	"github.com/romashorodok/news-tracker/backend/pkg/txutils"
	"github.com/romashorodok/news-tracker/pkg/sqlutils"
	"go.uber.org/fx"
)

type ArticleService struct {
	db      *sql.DB
	queries *storage.Queries
	kv      nats.KeyValue
}

var (
	ErrUnableCreateImage       = errors.New("unable create the image")
	ErrUnableCreateArticle     = errors.New("unable create the article")
	ErrArticleRequireMainImage = errors.New("article require at least the main image")
	ErrArticleNotFound         = errors.New("article not found")
	ErrArticlesNotFound        = errors.New("articles not found")
	ErrArticlesCount           = errors.New("unable get articles count")
)

type ArticleSorting string

const (
	ARTICLE_SORTING_NEWEST ArticleSorting = "newest"
	ARTICLE_SORTING_OLDEST ArticleSorting = "oldest"
	DEFAULT_PAGE           int            = 1
	DEFAULT_PAGE_SIZE      int            = 7
)

type GetArticlesParams struct {
	Sorting    ArticleSorting
	StartDate  time.Time
	EndDate    time.Time
	TextLexems []string
	Page       int
	PageSize   int
}

var DEFAULT_START_DATE = time.Now().AddDate(-10, 0, 0)

func (s *ArticleService) GetArticles(ctx context.Context, params GetArticlesParams) ([]model.Article, error) {
	articles, err := s.queries.Articles(ctx, storage.ArticlesParams{
		StartDate:        sqlutils.GetNullableSqlTime(params.StartDate),
		StartDateDefault: DEFAULT_START_DATE,
		EndDate:          sqlutils.GetNullableSqlTime(params.EndDate),
		Lexems:           params.TextLexems,
		ArticleSorting:   string(params.Sorting),
		Page:             int64((params.Page - 1) * params.PageSize),
		PageSize:         int64(params.PageSize),
	})
	if errors.Is(err, sql.ErrNoRows) || len(articles) == 0 {
		return nil, ErrArticlesNotFound
	}

	return accessor.ArticlesFromArticlesRows(articles)
}

func (s *ArticleService) GetArticleByID(ctx context.Context, id int64) (model.Article, error) {
	article, err := s.queries.GetArticleByID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return model.NilArticle, ErrArticleNotFound
	}

	return accessor.ArticleFromArticleByIdRow(article)
}

type NewArticleParams struct {
	Article           storage.NewArticleParams
	MainImageURL      string
	ContentImagesURLs []string
}

func (s *ArticleService) NewArticle(ctx context.Context, params NewArticleParams) (id int64, err error) {
	err = txutils.WithTransaction(s.db, func(queries *storage.Queries) error {
		articleID, err := s.queries.NewArticle(ctx, params.Article)
		if err != nil {
			log.Printf("unable create the article. Err:%s", err)
			return ErrUnableCreateArticle
		}

		if err = s.newArticleImage(ctx, articleID, params.MainImageURL, true); err != nil {
			log.Printf("unable create the article image. Err:%s", err)
			return err
		}

		for _, imageURL := range params.ContentImagesURLs {
			if err = s.newArticleImage(ctx, articleID, imageURL, false); err != nil {
				log.Printf("unable create the article image. Err:%s", err)
				return err
			}
		}

		id = articleID
		return nil
	})
	return id, err
}

type GetArticlesCountParams struct {
	StartDate  time.Time
	EndDate    time.Time
	TextLexems []string
}

func (s *ArticleService) GetArticlesCount(ctx context.Context, cacheKey string, params GetArticlesCountParams) (int, error) {
	val, err := s.kv.Get(cacheKey)
	if err == nil {
		count, err := strconv.Atoi(string(val.Value()))
		if err == nil {
			return count, nil
		}
	}

	count, err := s.queries.GetArticleCount(ctx, storage.GetArticleCountParams{
		StartDate:        sqlutils.GetNullableSqlTime(params.StartDate),
		StartDateDefault: DEFAULT_START_DATE,
		EndDate:          sqlutils.GetNullableSqlTime(params.EndDate),
		Lexems:           params.TextLexems,
	})
	if err != nil {
		return -1, errors.Join(ErrArticlesCount, err)
	}

	_, err = s.kv.Put(cacheKey, []byte(fmt.Sprint(count)))
    if err != nil {
        log.Println("Unable store cache for %s", cacheKey)
    }

	// NOTE: cast int64 may be dangerous
	return int(count), nil
}

func (s *ArticleService) newArticleImage(ctx context.Context, articleID int64, url string, main bool) error {
	imageID, err := s.queries.NewImage(ctx, url)
	if err != nil {
		return ErrUnableCreateImage
	}

	if err = s.queries.AttachArticleImage(ctx, storage.AttachArticleImageParams{
		ArticleID: articleID,
		ImageID:   imageID,
		Main:      main,
	}); err != nil {
		return ErrUnableCreateImage
	}
	return nil
}

func (s *ArticleService) GetArticleIDByTitleAndOrigin(ctx context.Context, params storage.GetArticleIDByTitleAndOriginParams) (int64, error) {
	return s.queries.GetArticleIDByTitleAndOrigin(ctx, params)
}

func (s *ArticleService) UpdateArticleStats(ctx context.Context, params storage.UpdateArticleStatsParams) error {
	return s.queries.UpdateArticleStats(ctx, params)
}

type NewArticleServiceParams struct {
	fx.In

	DB *sql.DB
	KV nats.KeyValue
}

func NewArticleSerivce(params NewArticleServiceParams) *ArticleService {
	return &ArticleService{
		db:      params.DB,
		queries: storage.New(params.DB),
		kv:      params.KV,
	}
}
