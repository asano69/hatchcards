# Stage 0: Node (vendor frontend assets via npm)
FROM node:22-alpine AS node-builder


WORKDIR /build
RUN mkdir -p /build/internal/assets
COPY frontend/ ./frontend/

WORKDIR /build/frontend
RUN corepack enable && pnpm install
RUN pnpm run build

# Stage 1
FROM golang:1.26-alpine AS go-builder
WORKDIR /build
COPY go.mod go.sum* ./
RUN go mod download || true

COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY migrations/ ./migrations/


COPY --from=node-builder /build/internal/assets/dist ./internal/assets/dist


RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o hashcards ./cmd/hashcards



# Stage 2: runtime
FROM alpine:3.23

COPY --from=ghcr.io/astral-sh/uv:latest /uv /uvx /usr/local/bin/

WORKDIR /hashcards

RUN apk add --no-cache \
    ca-certificates \
    su-exec \
    busybox-extras \
    tzdata \
    bash \
    nano \
    git \
    openssh-client \
    jq \
    curl \
    rsync \
    python3 \
    gawk


RUN addgroup -g 1000 hashcards && \
    adduser -D -u 1000 -G hashcards hashcards

COPY --from=go-builder /build/hashcards /usr/local/bin/hashcards

RUN mkdir -p /certs

RUN mkdir -p /hashcards/data
RUN mkdir -p /hashcards/cards
# Default location for pre-installed post-sync hook scripts (see
# internal/hook and HOOKS_DIR below). Left empty if no hooks are mounted;
# a missing/empty hooks directory is not an error, it just means no hooks
# are available.
RUN mkdir -p /hashcards/hooks

RUN chown -R 1000:1000 /hashcards

COPY entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

EXPOSE 3000

ENTRYPOINT ["entrypoint.sh"]
CMD ["hashcards", "serve"]

