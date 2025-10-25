# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Копируем исходный код
COPY go.mod go.sum ./
COPY internal/agent/ ./internal/agent/
COPY internal/agent/domain/ ./internal/agent/domain/
COPY internal/agent/clients/ ./internal/agent/clients/
COPY internal/agent/handlers/ ./internal/agent/handlers/
COPY internal/agent/runners/ ./internal/agent/runners/
COPY pkg/ ./pkg/
COPY cmd/ ./cmd/

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o agent ./cmd/agent

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates iputils bind-tools && \
    adduser -D -s /bin/sh appuser

WORKDIR /app

# Копируем бинарник из builder stage
COPY --from=builder /app/agent .

# Настраиваем права для сетевых утилит
RUN chown root:appuser /bin/ping && \
    chmod 4710 /bin/ping && \
    chown root:appuser /usr/bin/traceroute && \
    chmod 4710 /usr/bin/traceroute && \
    chown -R appuser:appuser /app

# Переключаемся на непривилегированного пользователя
USER appuser

# Запускаем приложение
CMD ["./agent"]