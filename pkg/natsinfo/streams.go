package natsinfo

import (
	"strings"

	nats "github.com/nats-io/nats.go"
)

const ARTICLES_STREAM_ANY_ARTICLE_SUBJECT = "article.*.*"

func ArticlesStream_NewArticleSubject(origin string, title string) string {
	title = strings.ReplaceAll(title, " ", "_")
	result := ARTICLES_STREAM_ANY_ARTICLE_SUBJECT
	result = strings.Replace(result, "*", origin, 1)
	result = strings.Replace(result, "*", title, 1)
	return result
}

var ARTICLES_STREAM_CONFIG = &nats.StreamConfig{
	Name:      "ARTICLES",
	Retention: nats.WorkQueuePolicy,
	Discard:   nats.DiscardOld,
	Subjects:  []string{ARTICLES_STREAM_ANY_ARTICLE_SUBJECT},
}
