# Stage 1: Go
FROM golang:1.26.1-alpine AS go-builder

WORKDIR /build
COPY go.mod go.sum* ./
RUN go mod download || true
COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o hashcards ./cmd/hashcards


# Stage 2: runtime
FROM alpine:3.23

WORKDIR /hashcards
RUN apk add --no-cache \
    ca-certificates \
    su-exec \
    busybox-extras \
    tzdata \
    bash \
    nano \
    git \
    openssh-client

RUN addgroup -g 1000 hashcards && \
    adduser -D -u 1000 -G hashcards hashcards

COPY --from=go-builder /build/hashcards /usr/local/bin/hashcards

RUN mkdir -p /hashcards/data
RUN mkdir -p /hashcards/cards

RUN chown -R 1000:1000 /hashcards
USER 1000:1000

EXPOSE 3000
ENTRYPOINT ["hashcards", "serve"]
