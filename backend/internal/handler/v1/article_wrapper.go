package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	chi "github.com/go-chi/chi/v5"
	"github.com/romashorodok/news-tracker/backend/internal/service"
	"github.com/romashorodok/news-tracker/pkg/dateutils"
	"github.com/romashorodok/news-tracker/pkg/httputils"
)

const (
	SORTING_QUERY_PARAM_NAME    = "sort"
	START_DATE_QUERY_PARAM_NAME = "start_date"
	END_DATE_QUERY_PARAM_NAME   = "end_date"
	TEXT_QUERY_PARAM_NAME       = "text"
	PAGE_QUERY_PARAM_NAME       = "page"
	PAGE_SIZE_QUERY_PARAM_NAME  = "page_size"
)

var ErrUnsupportedQueryParam = errors.New("")

type GetArticlesQueryParams struct {
	Sorting    service.ArticleSorting
	StartDate  time.Time
	EndDate    time.Time
	TextLexems []string
	Page       int
	PageSize   int
}

type GetArticleByIDUrlParams struct {
	ID int64
}

type ArticleHandler interface {
	GetArticles(w http.ResponseWriter, r *http.Request, queryParams *GetArticlesQueryParams)
	GetArticleByID(w http.ResponseWriter, r *http.Request, params *GetArticleByIDUrlParams)
}

type ArticleHandlerWrapper interface {
	GetArticles(w http.ResponseWriter, r *http.Request)
	GetArticleByID(w http.ResponseWriter, r *http.Request)
}

type articleParamsWrapperHandler struct {
	handler ArticleHandler
}

func (h *articleParamsWrapperHandler) GetArticleByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httputils.WriteErrorResponse(w, http.StatusPreconditionRequired, err.Error())
		return
	}

	handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.handler.GetArticleByID(w, r, &GetArticleByIDUrlParams{
			ID: int64(id),
		})
	}))
	handler.ServeHTTP(w, r)
}

func getArticleSortingQuery(r *http.Request, defaultVal service.ArticleSorting) (service.ArticleSorting, error) {
	sortingParam := r.URL.Query().Get(SORTING_QUERY_PARAM_NAME)
	switch service.ArticleSorting(sortingParam) {
	case service.ARTICLE_SORTING_NEWEST:
		return service.ARTICLE_SORTING_NEWEST, nil
	case service.ARTICLE_SORTING_OLDEST:
		return service.ARTICLE_SORTING_OLDEST, nil
	case "":
		return defaultVal, nil
	default:
		return "", errors.Join(fmt.Errorf("unsupported `%s` query value %s", SORTING_QUERY_PARAM_NAME, sortingParam), ErrUnsupportedQueryParam)
	}
}

func getDateQuery(r *http.Request, queryName string) (time.Time, error) {
	date := r.URL.Query().Get(queryName)
	if date == "" {
		return time.Time{}, nil
	}
	t, err := dateutils.ParseQueryString(date)
	if err != nil {
		return time.Time{}, errors.Join(fmt.Errorf("unsupported `%s` query value %s. Format must be like `2024-10-12T10:01`, `2024-10-12`, `YYYY-MM-DD` ", SORTING_QUERY_PARAM_NAME, date), ErrUnsupportedQueryParam)
	}
	return t, nil
}

func getTextQuery(r *http.Request) []string {
	return strings.Split(r.URL.Query().Get(TEXT_QUERY_PARAM_NAME), " ")
}

func getPageQuery(r *http.Request, defaultPage int) (int, error) {
	pageStr := r.URL.Query().Get(PAGE_QUERY_PARAM_NAME)
	if pageStr == "" {
		return defaultPage, nil
	}
	page, err := strconv.Atoi(pageStr)
	if err != nil {
		return -1, errors.Join(fmt.Errorf("unsupported `%s` page value %s. Support only numbers", PAGE_QUERY_PARAM_NAME, pageStr), ErrUnsupportedQueryParam)
	}
	return page, nil
}

func getPageSizeQuery(r *http.Request, defaultPageSize int) (int, error) {
	pageSizeStr := r.URL.Query().Get(PAGE_SIZE_QUERY_PARAM_NAME)
	if pageSizeStr == "" {
		return defaultPageSize, nil
	}
	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil {
		return -1, errors.Join(fmt.Errorf("unsupported `%s` page size value %s. Support only numbers", PAGE_QUERY_PARAM_NAME, pageSizeStr), ErrUnsupportedQueryParam)
	}
	return pageSize, nil
}

func (h *articleParamsWrapperHandler) GetArticles(w http.ResponseWriter, r *http.Request) {
	sorting, err := getArticleSortingQuery(r, service.ARTICLE_SORTING_NEWEST)
	if err != nil {
		articleErrHandler(w, err)
	}

	startDate, err := getDateQuery(r, START_DATE_QUERY_PARAM_NAME)
	if err != nil {
		articleErrHandler(w, err)
		return
	}

	endDate, err := getDateQuery(r, END_DATE_QUERY_PARAM_NAME)
	if err != nil {
		articleErrHandler(w, err)
		return
	}

	textLexems := getTextQuery(r)

	page, err := getPageQuery(r, service.DEFAULT_PAGE)
	if err != nil {
		articleErrHandler(w, err)
		return
	}

	pageSize, err := getPageSizeQuery(r, service.DEFAULT_PAGE_SIZE)
	if err != nil {
		articleErrHandler(w, err)
		return
	}

	handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.handler.GetArticles(w, r, &GetArticlesQueryParams{
			Sorting:    sorting,
			StartDate:  startDate,
			EndDate:    endDate,
			TextLexems: textLexems,
			Page:       page,
			PageSize:   pageSize,
		})
	}))
	handler.ServeHTTP(w, r)
}

func (h *articleParamsWrapperHandler) OnRouter(router http.Handler) {
	switch r := router.(type) {
	case *chi.Mux:
		baseURL := "/api/v1"
		r.Get(baseURL+"/articles", h.GetArticles)
		r.Get(baseURL+"/articles/{id}", h.GetArticleByID)
	}
}

var _ httputils.Handler = (*articleParamsWrapperHandler)(nil)

func newArticleParamsWrapper(handler ArticleHandler) *articleParamsWrapperHandler {
	return &articleParamsWrapperHandler{
		handler: handler,
	}
}

func articleErrHandler(w http.ResponseWriter, err error) {
	switch err {
	case service.ErrArticlesNotFound:
		httputils.WriteErrorResponse(w, http.StatusNotFound, err.Error())
		return
	case service.ErrArticleNotFound:
		httputils.WriteErrorResponse(w, http.StatusNotFound, err.Error())
		return
	case ErrUnsupportedQueryParam:
		httputils.WriteErrorResponse(w, http.StatusNotAcceptable, err.Error())
	default:
		httputils.WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
	}
}
