package natsinfo

import (
	"strings"
	"time"

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

// Create config for queue group. Which filter all `article.*.*` into queue group
// NOTE: Only one consumer may consume the `article.*.*` messages.
// I can recv specific messages like `article.google-news.*` in one place and in the another `article.bing-news.*` from one `ARTICLES` stream.
func ArticlesStream_NewArticleConsumerConfig(queueGroup string) (stream string, subject string, subOpts []nats.SubOpt, config *nats.ConsumerConfig) {
	// Batch the 15 messages, each message has 15 second to be commited explicitly. If not that message will retry.
	// Has redelivered message priority higher then unprocessed.
	// In that case unprocessed message will wait until redelivered message will commited
	// Redelivered Messages: 15
	// Unprocessed Messages: 1
	config = &nats.ConsumerConfig{
		Durable:        queueGroup,
		DeliverSubject: queueGroup,
		DeliverGroup:   queueGroup,
		AckWait:        time.Second * 15,
		AckPolicy:      nats.AckExplicitPolicy,
		DeliverPolicy:  nats.DeliverAllPolicy,
		FilterSubject:  ARTICLES_STREAM_ANY_ARTICLE_SUBJECT,
		MaxAckPending:  15,
	}
	subOpts = []nats.SubOpt{
		nats.Bind(ARTICLES_STREAM_CONFIG.Name, queueGroup),
		nats.DeliverAll(),
		nats.ManualAck(),
		nats.AckExplicit(),
		nats.MaxAckPending(15),
	}
	return ARTICLES_STREAM_CONFIG.Name, ARTICLES_STREAM_ANY_ARTICLE_SUBJECT, subOpts, config
}
