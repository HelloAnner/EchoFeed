# 构建阶段
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o echofeed ./cmd/server

# 运行阶段
FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/echofeed .
COPY --from=builder /app/web ./web

EXPOSE 8080
VOLUME ["/app/data"]

CMD ["./echofeed"]
