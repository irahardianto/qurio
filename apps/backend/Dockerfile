FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /server main.go

FROM alpine:latest

# Create a non-root user
RUN adduser -D -g '' appuser

WORKDIR /app

# Copy binary and migrations
COPY --from=builder /server .
COPY migrations ./migrations

# Set permissions and create upload dir
RUN mkdir -p /var/lib/qurio/uploads && \
    chown -R appuser:appuser /var/lib/qurio && \
    chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

EXPOSE 8081
CMD ["./server"]