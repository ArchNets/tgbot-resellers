# Build stage
FROM golang:1.26.4-alpine AS builder

WORKDIR /src

# Install ca-certificates and git for dependency fetching if needed
RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bot ./cmd/bot

# Runtime stage
FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

RUN addgroup -S botgroup && adduser -S botuser -G botgroup && chown -R botuser:botgroup /app

COPY --from=builder --chown=botuser:botgroup /app/bot /app/bot

USER botuser

ENTRYPOINT ["/app/bot"]
