package natsinfo

import (
	"time"

	nats "github.com/nats-io/nats.go"
)

var (
	ARTICLE_COUNT_BUCKET_NAME      = "articles"
	ARTICLE_COUNT_KEY_VALUE_CONFIG = nats.KeyValueConfig{
		Bucket: ARTICLE_COUNT_BUCKET_NAME,
		TTL:    time.Minute * 2,
	}
)
