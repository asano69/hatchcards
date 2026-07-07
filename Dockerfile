# syntax=docker/dockerfile:1

# ==========================================
# Stage 0: Node (vendor frontend assets via npm)
# ==========================================
FROM node:22-alpine AS node-builder
WORKDIR /build/frontend
# Copy only dependency manifests first to leverage Docker layer caching
COPY frontend/package.json frontend/pnpm-lock.yaml* ./
RUN corepack enable
RUN --mount=type=cache,target=/root/.local/share/pnpm/store \
    pnpm install
# Copy the rest of the frontend source code and build
COPY frontend/ ./
RUN --mount=type=cache,target=/root/.local/share/pnpm/store \
    pnpm run build

# ==========================================
# Stage 1: Go Builder
# ==========================================
FROM golang:1.26-alpine AS go-builder
WORKDIR /build
# Copy and download Go dependencies first
COPY go.mod go.sum* ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download
# Copy frontend build artifacts just before the Go compilation step
COPY --from=node-builder /build/internal/assets/dist ./internal/assets/dist
# Copy Go source files last, as they change most frequently
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY migrations/ ./migrations/
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o hatchcards ./cmd/hatchcards

# ==========================================
# Stage 2: Runtime
# ==========================================
FROM alpine:3.23
COPY --from=ghcr.io/astral-sh/uv:0.11.28 /uv /uvx /usr/local/bin/


WORKDIR /hatchcards

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

RUN addgroup -g 1000 hatchcards && \
    adduser -D -u 1000 -G hatchcards hatchcards

COPY --from=go-builder /build/hatchcards /usr/local/bin/hatchcards

RUN mkdir -p /certs /hatchcards/data /hatchcards/cards /hatchcards/hooks
RUN chown -R 1000:1000 /hatchcards

COPY entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

EXPOSE 3000

ENTRYPOINT ["entrypoint.sh"]
CMD ["hatchcards", "serve"]
