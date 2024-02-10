package prebuiltemplate

import (
	"context"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/romashorodok/news-tracker/pkg/natsinfo"
	"github.com/romashorodok/news-tracker/worker/pkg/parser"
	"github.com/romashorodok/news-tracker/worker/pkg/parser/selector"
)

func getRemotePage(path string) (io.ReadCloser, error) {
	resp, err := http.Get(path)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

type ArticleConfig struct {
	Fields []Field `json:"fields"`
}

type NewsFeedConfig struct {
	NewsFeedURL             string   `json:"news_feed_url"`
	NewsFeedArticleSelector []string `json:"news_feed_article_selector"`
	NewsFeedRefreshInterval int      `json:"news_feed_refresh_interval"`

	ArticlePrefixURL    string        `json:"article_prefix_url"`
	ArticleConfig       ArticleConfig `json:"article_config"`
	ArticlePullInterval int           `json:"article_pull_interval"`
	ArticlePageSelector []string      `json:"article_page_selector"`
}

type NewsFeedProcessor struct {
	ArticleChan                   chan natsinfo.Article
	newsFeedRefreshIntervalTicker *time.Ticker
	articlePullIntervalTicker     *time.Ticker
	config                        NewsFeedConfig
	origin                        string
}

// Process the article node which point to the detail page
//
// Example: Each article container on feed has something which point to the actual page of the article
// <li><a class="article-button" href="http://.../article/id">{Some title}</a></li>
// NewsFeedConfig.ArticlePageSelector must be the `article-button` to select the node here
func (n *NewsFeedProcessor) onArticlePageNode(node *parser.Node) {
	select {
	case <-n.articlePullIntervalTicker.C:
		url := n.config.ArticlePrefixURL + node.Tag.Attr["href"]
		log.Println("Get article page at", url)

		detailPage, err := getRemotePage(url)
		if err != nil {
			log.Println("Unable get remote article page at")
			return
		}
		defer detailPage.Close()
		detailPageExtractor := NewArticlePageExtractor(n.config)

		var selectors []parser.Selector

		for _, field := range n.config.ArticleConfig.Fields {
			switch field.Type {
			case FIELD_TYPE_TITLE:
				selectors = append(selectors, selector.NewClassSelector([]string{field.ClassSelector}, detailPageExtractor.OnTitle(field)))
			case FIELD_TYPE_CONTENT:
				selectors = append(selectors, selector.NewClassSelector([]string{field.ClassSelector}, detailPageExtractor.OnContent(field)))
			case FIELD_TYPE_PREFACE:
				selectors = append(selectors, selector.NewClassSelector([]string{field.ClassSelector}, detailPageExtractor.OnPreface(field)))
			case FIELD_TYPE_PUBLISHED_AT:
				selectors = append(selectors, selector.NewClassSelector([]string{field.ClassSelector}, detailPageExtractor.OnPublishDate(field)))
			case FIELD_TYPE_INFO:
				selectors = append(selectors, selector.NewClassSelector([]string{field.ClassSelector}, detailPageExtractor.OnInfo(field)))
			case FIELD_TYPE_MAIN_IMAGE:
				selectors = append(selectors, selector.NewClassSelector([]string{field.ClassSelector}, detailPageExtractor.OnMainImage(field)))
			case FIELD_TYPE_CONTENT_IMAGES:
				selectors = append(selectors, selector.NewClassSelector([]string{field.ClassSelector}, detailPageExtractor.OnContentImages(field)))
			}
		}

		parser.Parse(detailPage, selectors...)
		article := detailPageExtractor.article
		article.Origin = n.origin
		n.ArticleChan <- article
	}
}

// Process each article items on news feed page
//
// Nodes which selected by the NewsFeedConfig.NewsFeedArticleSelector
// That selector must point to the root of container/wrapper.
//
// Example: Each news feed has something like list of nodes and all articles inside it
// <ol><li class="article-item"></li><li class="article-itme"></li></ol>
//
// NewsFeedConfig.NewsFeedArticleSelector must be the `article-item` to select that nodes here
func (n *NewsFeedProcessor) onNewsFeedArticleNode(node *parser.Node) {
	for node := node; node != nil; node = node.Next {
		classesStr := node.Tag.Attr["class"]
		// Find the node which contain element which point to the article page.
		if parser.ContainsClass(classesStr, n.config.ArticlePageSelector) {
			n.onArticlePageNode(node)
			break
		}
	}
}

func (n *NewsFeedProcessor) GetArticleChan() <-chan natsinfo.Article {
	return n.ArticleChan
}

func (n *NewsFeedProcessor) Start(ctx context.Context) {
	defer close(n.ArticleChan)
	url := n.config.NewsFeedURL
	n.origin = strings.Split(strings.SplitAfter(url, "//")[1], "/")[0]
	for {
		log.Printf("Next news feed refresh at %s", time.Now().Add(time.Duration(n.config.NewsFeedRefreshInterval)))
		select {
		case <-ctx.Done():
			return
		case <-n.newsFeedRefreshIntervalTicker.C:
			log.Println("Refresh news feed page", n.config.NewsFeedURL)
			resp, err := getRemotePage(n.config.NewsFeedURL)
			if err != nil {
				log.Println("Unable get remote news feed page at", n.config.NewsFeedURL)
				continue
			}
			parser.Parse(resp, selector.NewClassSelector(
				n.config.NewsFeedArticleSelector,
				n.onNewsFeedArticleNode,
			))
			resp.Close()
			log.Printf("Done news feed page refresh for %s", n.config.NewsFeedURL)
		}
	}
}

func NewNewsFeedProcessor(config NewsFeedConfig) *NewsFeedProcessor {
	return &NewsFeedProcessor{
		newsFeedRefreshIntervalTicker: time.NewTicker(
			time.Duration(config.NewsFeedRefreshInterval),
		),
		articlePullIntervalTicker: time.NewTicker(
			time.Duration(config.ArticlePullInterval),
		),
		config:      config,
		ArticleChan: make(chan natsinfo.Article),
	}
}
