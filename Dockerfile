
FROM golang:1.22.0-alpine3.19 as pkg-deps
WORKDIR /app/pkg
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=bind,source=./pkg/go.mod,target=go.mod \
    go mod download -x

FROM golang:1.22.0-alpine3.19 as backend-deps
WORKDIR /app/backend
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=bind,source=./backend/go.mod,target=go.mod \
    --mount=type=bind,source=./backend/go.sum,target=go.sum \
    go mod download -x

FROM golang:1.22.0-alpine3.19 as backend-builder
WORKDIR /app
RUN go env -w GOCACHE=/go-cache
# https://docs.docker.com/storage/bind-mounts/#use-a-read-only-bind-mount
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/go-cache \
    --mount=type=bind,source=.,target=.,readonly \
    apkArch="$(apk --print-arch)"; \
    case "$apkArch" in \
    aarch64) export GOARCH='arm64' ;; \
    *) export GOARCH='amd64' ;; \
    esac; \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /go/bin/backend ./backend/main.go

FROM gcr.io/distroless/base:debug as backend 
COPY --from=backend-builder /go/bin/backend /app/backend
ENTRYPOINT [ "/app/backend" ]

FROM golang:1.22.0-alpine3.19 as worker-deps
WORKDIR /app/worker
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=bind,source=./worker/go.mod,target=go.mod \
    --mount=type=bind,source=./worker/go.sum,target=go.sum \
    go mod download -x

FROM golang:1.22.0-alpine3.19 as worker-builder
WORKDIR /app
RUN go env -w GOCACHE=/go-cache
# https://docs.docker.com/storage/bind-mounts/#use-a-read-only-bind-mount
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/go-cache \
    --mount=type=bind,source=.,target=.,readonly \
    apkArch="$(apk --print-arch)"; \
    case "$apkArch" in \
    aarch64) export GOARCH='arm64' ;; \
    *) export GOARCH='amd64' ;; \
    esac; \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /go/bin/worker ./worker/main.go

FROM gcr.io/distroless/base:debug as worker
COPY --from=worker-builder /go/bin/worker /app/worker
ENTRYPOINT [ "/app/worker" ]
