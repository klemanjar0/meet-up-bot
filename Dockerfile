# syntax=docker/dockerfile:1

# ---- build stage ---------------------------------------------------------
FROM golang:1.25-alpine AS build

WORKDIR /src

# Cache module downloads separately from the source.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build both binaries statically so they run on a scratch/alpine base.
ARG TARGETOS TARGETARCH
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build -ldflags="-s -w" -o /out/bot ./cmd/bot && \
    CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build -ldflags="-s -w" -o /out/migrate ./cmd/migrate

# ---- runtime stage -------------------------------------------------------
FROM alpine:3.20 AS runtime

RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -u 10001 app

COPY --from=build /out/bot /usr/local/bin/bot
COPY --from=build /out/migrate /usr/local/bin/migrate

USER app

# Default to running the bot; docker-compose overrides the command for the
# one-shot migration container.
ENTRYPOINT ["bot"]
