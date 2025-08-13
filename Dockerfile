# syntax=docker/dockerfile:1

############################
# Build stage
############################
FROM golang:1.21-alpine AS builder

# git/ca-certificates: Go modules, HTTPS 통신
# build-base/pkgconf/librdkafka-dev: confluent-kafka-go 빌드용(CGo)
RUN apk update && apk add --no-cache \
    git ca-certificates \
    build-base pkgconf librdkafka-dev

WORKDIR /app

# 모듈 캐시 최적화
COPY go.mod go.sum ./
RUN go mod download

# 소스 복사
COPY . .

# 빌드 (메인 엔트리는 예시로 ./cmd/main.go 사용)
# -s -w: 바이너리 사이즈 축소
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -ldflags="-s -w" -o main ./cmd/main.go

############################
# Runtime stage
############################
FROM alpine:3.19

# 런타임에 필요한 최소 패키지
# - librdkafka: confluent-kafka-go 런타임 의존
# - tzdata/ca-certificates: 로깅 타임존/HTTPS
# - wget: 헬스체크에서 사용
RUN apk --no-cache add ca-certificates tzdata librdkafka wget

WORKDIR /app
COPY --from=builder /app/main .

# 운영 기본값
ENV TZ=Asia/Seoul \
    GIN_MODE=release \
    PORT=8080

EXPOSE 8080

# 헬스체크: order-service의 헬스 엔드포인트 경로에 맞춰 필요시 수정
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://127.0.0.1:${PORT}/api/v1/health || exit 1

# 비루트로 실행(1024 초과 포트이므로 문제 없음)
USER 65532:65532

CMD ["./main"]
