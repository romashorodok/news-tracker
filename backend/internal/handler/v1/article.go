package handler

import (
	"encoding/json"
	"net/http"

	"github.com/romashorodok/news-tracker/backend/internal/model"
	"github.com/romashorodok/news-tracker/backend/internal/service"
	"github.com/romashorodok/news-tracker/pkg/hashutils"
	"github.com/romashorodok/news-tracker/pkg/paginationutils"
	"go.uber.org/fx"
)

type articleHandler struct {
	articleService *service.ArticleService
}

type getArticlesResponse struct {
	Articles []model.Article                  `json:"articles"`
	Pages    []paginationutils.PaginationLink `json:"pages"`
}

func (hand *articleHandler) GetArticles(w http.ResponseWriter, r *http.Request, queryParams *GetArticlesQueryParams) {
	articles, err := hand.articleService.GetArticles(r.Context(), service.GetArticlesParams{
		Sorting:    queryParams.Sorting,
		StartDate:  queryParams.StartDate,
		EndDate:    queryParams.EndDate,
		TextLexems: queryParams.TextLexems,
		Page:       queryParams.Page,
		PageSize:   queryParams.PageSize,
	})
	if err != nil {
		articleErrHandler(w, err)
		return
	}

	// TODO: I can run it in a goroutine
	cacheKey := hashutils.GetCacheKey(queryParams.StartDate, queryParams.EndDate, queryParams.TextLexems)

	articlesCount, err := hand.articleService.GetArticlesCount(r.Context(), cacheKey, service.GetArticlesCountParams{
		StartDate:  queryParams.StartDate,
		EndDate:    queryParams.EndDate,
		TextLexems: queryParams.TextLexems,
	})

	pagination := paginationutils.NewPaginationView(*r.URL, paginationutils.NewPaginationViewParams{
		ItemsPerPage:       queryParams.PageSize,
		ItemsCount:         articlesCount,
		PageQueryParamName: PAGE_QUERY_PARAM_NAME,
	})

	pagesLinks, err := pagination.PagesLinks(queryParams.Page)
	if err != nil {
		articleErrHandler(w, err)
		return
	}

	json.NewEncoder(w).Encode(&getArticlesResponse{
		Articles: articles,
		Pages:    pagesLinks,
	})
}

func (hand *articleHandler) GetArticleByID(w http.ResponseWriter, r *http.Request, params *GetArticleByIDUrlParams) {
	article, err := hand.articleService.GetArticleByID(r.Context(), params.ID)
	if err != nil {
		articleErrHandler(w, err)
		return
	}
	json.NewEncoder(w).Encode(&article)
}

var _ ArticleHandler = (*articleHandler)(nil)

type NewArticleHandlerParams struct {
	fx.In

	ArticleService *service.ArticleService
}

func NewArticleHandler(params NewArticleHandlerParams) *articleParamsWrapperHandler {
	return newArticleParamsWrapper(&articleHandler{
		articleService: params.ArticleService,
	})
}
