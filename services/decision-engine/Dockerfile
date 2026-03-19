FROM golang:1.24-alpine AS base

FROM base AS builder

RUN apk add --no-cache git build-base libc6-compat

# Set working directory
WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./

# Download Go modules
RUN go mod download

# Copy the rest of the application
COPY ./ ./

# # Install sqlc and generate code
# RUN go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
# RUN sqlc generate

# Build the application
RUN CGO_ENABLED=0 go build -o ./chat-bot-start ./cmd/app/main.go

FROM base AS runner
WORKDIR /app

# Don't run production as root
RUN addgroup --system --gid 1001 chat-bot-group
RUN adduser --system --uid 1001 chat-bot-user

# Copy built application
COPY --from=builder --chown=chat-bot-group:chat-bot-user /app/chat-bot-start /usr/local/bin/chat-bot

# Copy configuration files
COPY --from=builder --chown=chat-bot-group:chat-bot-user /app/configs ./configs

USER chat-bot-user

# Set APP_ENV to production by default (loads config.prod.yaml)
ENV APP_ENV=dev

EXPOSE 3000

CMD ["chat-bot"]