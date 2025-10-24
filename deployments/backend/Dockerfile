# Build stage
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o checkmesh-backend ./cmd/backend

# Runtime stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates && \
    adduser -D -s /bin/sh appuser
WORKDIR /app
COPY --from=builder /app/checkmesh-backend .
COPY configs/ ./configs/
RUN chown -R appuser:appuser /app
USER appuser
EXPOSE 8080
CMD ["./checkmesh-backend"]