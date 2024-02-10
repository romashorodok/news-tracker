
```go
config := prebuiltemplate.NewsFeedConfig{
    NewsFeedURL:             "",
    NewsFeedRefreshInterval: 600000000000,
    NewsFeedArticleSelector: []string{"blog-item"},

    ArticleConfig: prebuiltemplate.ArticleConfig{
        Fields: []prebuiltemplate.Field{
            {Type: prebuiltemplate.FIELD_TYPE_TITLE, ClassSelector: "News__title"},
            {Type: prebuiltemplate.FIELD_TYPE_PREFACE, ClassSelector: "article-main-intro"},
            {Type: prebuiltemplate.FIELD_TYPE_CONTENT, ClassSelector: "article-main-text", IgnoredSentences: []string{"Отримуйте новини в Telegram", "Наші новини є у Facebook", "Дивіться нас на YouTube"}},
            {Type: prebuiltemplate.FIELD_TYPE_PUBLISHED_AT, ClassSelector: "PostInfo__item PostInfo__item_date"},
            {Type: prebuiltemplate.FIELD_TYPE_INFO, ClassSelector: "PostInfo__item PostInfo__item_service"},
            {Type: prebuiltemplate.FIELD_TYPE_MAIN_IMAGE, ClassSelector: "article-main-image NewsImg"},
            {Type: prebuiltemplate.FIELD_TYPE_CONTENT_IMAGES, ClassSelector: "article-main-text"},
        },
    },
    ArticlePrefixURL:    "https:",
    ArticlePullInterval: 30000000000,
    ArticlePageSelector: []string{"AllNewsItemInfo__name"},
}
```
