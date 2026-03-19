# Stage 1 — Build Go binary
FROM golang:1.25-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ cmd/
COPY internal/ internal/
RUN CGO_ENABLED=0 go build -o /app/bot ./cmd/bot/

# Stage 2 — Runtime with Node.js (required for Claude CLI)
FROM node:22-alpine

RUN npm install -g @anthropic-ai/claude-code && npm cache clean --force

RUN adduser -D -h /app appuser

COPY --from=builder /app/bot /app/bot
COPY scripts/docker-entrypoint.sh /app/docker-entrypoint.sh

RUN mkdir -p /app/data /app/static /app/.claude && chown -R appuser:appuser /app

ENV HOME=/app
WORKDIR /app
USER appuser

EXPOSE 3000

ENTRYPOINT ["/app/docker-entrypoint.sh"]
