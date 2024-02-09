package prebuiltemplate

import (
	"strings"
	"time"

	"github.com/romashorodok/news-tracker/pkg/dateutils"
	"github.com/romashorodok/news-tracker/worker/pkg/parser"
)

const (
	FIELD_TYPE_TITLE          = "title"
	FIELD_TYPE_PREFACE        = "preface"
	FIELD_TYPE_CONTENT        = "content"
	FIELD_TYPE_PUBLISHED_AT   = "published_at"
	FIELD_TYPE_INFO           = "info"
	FIELD_TYPE_MAIN_IMAGE     = "main_image"
	FIELD_TYPE_CONTENT_IMAGES = "content_images"
)

type Field struct {
	Type             string   `json:"type"`
	ClassSelector    string   `json:"class_selector"`
	IgnoredSentences []string `json:"ignored_sentences"`
}

type ArticleExtractorConfig struct {
	Fields []Field `json:"fields"`
}

type Article struct {
	Title         string   `json:"title"`
	Preface       string   `json:"preface"`
	Content       string   `json:"content"`
	PublishedAt   string   `json:"published_at"`
	ViewersCount  string   `json:"viewers_count"`
	MainImage     string   `json:"main_image"`
	ContentImages []string `json:"content_images,omitempty"`
}

type ArticlePageExtractor struct {
	article Article
	config  NewsFeedConfig
}

func (n *ArticlePageExtractor) OnMainImage(field Field) func(*parser.Node) {
	return func(node *parser.Node) {
		for node := node; node != nil; node = node.Next {
			if node.Name == "img" {
				if node.Tag.Attr == nil {
					continue
				}
				n.article.MainImage = n.config.ArticlePrefixURL + node.Tag.Attr["src"]
				break
			}
		}
	}
}

func (n *ArticlePageExtractor) OnContentImages(field Field) func(*parser.Node) {
	return func(node *parser.Node) {
		for node := node; node != nil; node = node.Next {
			if node.Name == "img" {
				if node.Tag.Attr == nil {
					continue
				}
				img := n.config.ArticlePrefixURL + node.Tag.Attr["src"]
				n.article.ContentImages = append(n.article.ContentImages, img)
			}
		}
	}
}

func (n *ArticlePageExtractor) OnContent(field Field) func(*parser.Node) {
	return func(node *parser.Node) {
		var content string
		for node := node; node != nil; node = node.Next {
			content += node.Content
		}
		for _, sentence := range field.IgnoredSentences {
			content = strings.Replace(content, sentence, "", -1)
		}
		n.article.Content = content
	}
}

func (n *ArticlePageExtractor) OnPreface(field Field) func(*parser.Node) {
	return func(node *parser.Node) {
		n.article.Preface = node.Next.Content
	}
}

func (n *ArticlePageExtractor) OnTitle(field Field) func(*parser.Node) {
	return func(node *parser.Node) {
		n.article.Title = node.Next.Content
	}
}

func (n *ArticlePageExtractor) OnPublishDate(field Field) func(*parser.Node) {
	return func(node *parser.Node) {
		date, err := dateutils.ParseDateUA(node.Next.Content)
		if err != nil {
			n.article.PublishedAt = dateutils.ToString(time.Now())
			return
		}
		n.article.PublishedAt = dateutils.ToString(date)
	}
}

const VIEWERS_COUNT_SELECTOR = "ServicePeopleItem__icon ServicePeopleItem__icon_look"

func (n *ArticlePageExtractor) OnInfo(field Field) func(*parser.Node) {
	return func(node *parser.Node) {
		for node := node; node != nil; node = node.Next {
			if node.Tag.Attr == nil {
				continue
			}

			if parser.ContainsClass(node.Tag.Attr["class"], []string{VIEWERS_COUNT_SELECTOR}) {
				n.article.ViewersCount = node.Next.Next.Next.Content
				return
			}
		}
	}
}

func NewArticlePageExtractor(config NewsFeedConfig) *ArticlePageExtractor {
	return &ArticlePageExtractor{
		config: config,
	}
}
