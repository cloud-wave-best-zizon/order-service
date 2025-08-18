# order-service/Dockerfile
FROM golang:1.24-alpine AS builder

# 필수 빌드 도구 설치
RUN apk add --no-cache git gcc musl-dev librdkafka-dev pkgconf

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# CGO_ENABLED=1로 변경
RUN CGO_ENABLED=1 GOOS=linux GOARCH=arm64 go build -ldflags="-w -s" -o main cmd/main.go

FROM alpine:3.18
# 런타임에 필요한 라이브러리 설치
RUN apk add --no-cache ca-certificates librdkafka

WORKDIR /root/
COPY --from=builder /app/main .
EXPOSE 8080
CMD ["./main"]