# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Копируем исходный код
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Собираем приложение
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o checkmesh-agent ./cmd/agent

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates iputils bind-tools && \
    adduser -D -s /bin/sh appuser

WORKDIR /app

# Копируем бинарник из builder stage
COPY --from=builder /app/checkmesh-agent .

# Настраиваем права для сетевых утилит
RUN chown root:appuser /bin/ping && \
    chmod 4710 /bin/ping && \
    chown root:appuser /usr/bin/traceroute && \
    chmod 4710 /usr/bin/traceroute && \
    chown -R appuser:appuser /app

# Переключаемся на непривилегированного пользователя
USER appuser

# Запускаем приложение
CMD ["./checkmesh-agent"]