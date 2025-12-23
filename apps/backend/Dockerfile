FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /server main.go

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /server .
COPY migrations ./migrations

EXPOSE 8081
CMD ["./server"]
