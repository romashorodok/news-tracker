## Task points

### Parser
The parser core can be found at [worker/pkg/parser](./worker/pkg/parser)
<br>
Prebuilt template which use it can be found at [worker/internal/prebuiltemplate](./worker/internal/prebuiltemplate)

The articles publishing happened [here](https://github.com/romashorodok/news-tracker/blob/main/worker/main.go#L62)

The article consuming happened [here](https://github.com/romashorodok/news-tracker/blob/main/backend/internal/worker/article_consumer.go#L85)

### Api

Pagination done at database level.

Also I had the pagination view for good frontend experience.
<br>
It can be found at [pkg/paginationutils](./pkg/paginationutils)

## Building

```shell
docker build .
```

> [!IMPORTANT]
> To run need Docker Compose version 2.22 and later.
> <br>
> ```docker-compose watch```

For migration:
```shell
goose -dir ./backend/migrations -table schema_migrations postgres postgres://admin:admin@localhost:5432/postgres up
```

Also for running worker need provide config:
```shell
echo "CONFIG='{...}'" > .env
```

## Database ERD
![erd](./docs/migration.png)
The migration ca be found at [backend/migrations](./backend/migrations)
<br>
The queries at [backend/internal/storage/queries](./backend/internal/storage/queries)