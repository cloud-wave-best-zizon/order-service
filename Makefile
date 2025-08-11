# Makefile for order-service

# 변수 설정
APP_NAME=order-service
MAIN_PATH=cmd/main.go
DOCKER_IMAGE=order-service:latest
GO=go
GOFLAGS=-v

# 색상 정의
GREEN=\033[0;32m
NC=\033[0m # No Color

.PHONY: help run build test clean docker-build docker-run deps lint fmt

# 기본 타겟
help:
	@echo "사용 가능한 명령어:"
	@echo "  make run         - 애플리케이션 실행 (로컬 모드)"
	@echo "  make build       - 애플리케이션 빌드"
	@echo "  make test        - 테스트 실행"
	@echo "  make clean       - 빌드 파일 정리"
	@echo "  make deps        - 의존성 다운로드"
	@echo "  make docker-build - Docker 이미지 빌드"
	@echo "  make docker-run   - Docker 컨테이너 실행"
	@echo "  make lint        - 코드 린트 검사"
	@echo "  make fmt         - 코드 포맷팅"
	@echo "  make kafka-up    - Kafka 시작"
	@echo "  make kafka-down  - Kafka 중지"
	@echo "  make stack-up    - 전체 스택 시작"
	@echo "  make stack-down  - 전체 스택 중지"

# 애플리케이션 실행 (로컬 모드)
run:
	@echo "$(GREEN)Starting $(APP_NAME)...$(NC)"
	$(GO) run $(MAIN_PATH)

# 애플리케이션 빌드
build:
	@echo "$(GREEN)Building $(APP_NAME)...$(NC)"
	$(GO) build $(GOFLAGS) -o bin/$(APP_NAME) $(MAIN_PATH)

# 테스트 실행
test:
	@echo "$(GREEN)Running tests...$(NC)"
	$(GO) test -v ./...

# 빌드 파일 정리
clean:
	@echo "$(GREEN)Cleaning build files...$(NC)"
	rm -rf bin/
	$(GO) clean

# 의존성 다운로드
deps:
	@echo "$(GREEN)Downloading dependencies...$(NC)"
	$(GO) mod download
	$(GO) mod tidy

# Docker 이미지 빌드
docker-build:
	@echo "$(GREEN)Building Docker image...$(NC)"
	docker build -t $(DOCKER_IMAGE) .

# Docker 컨테이너 실행
docker-run:
	@echo "$(GREEN)Running Docker container...$(NC)"
	docker run -p 8080:8080 \
		-e AWS_REGION=ap-northeast-2 \
		-e ORDER_TABLE_NAME=orders \
		-e KAFKA_BROKERS=host.docker.internal:9092 \
		$(DOCKER_IMAGE)

# 코드 린트 검사
lint:
	@echo "$(GREEN)Running linter...$(NC)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Installing..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
		golangci-lint run ./...; \
	fi

# 코드 포맷팅
fmt:
	@echo "$(GREEN)Formatting code...$(NC)"
	$(GO) fmt ./...

# 모든 빌드 및 테스트 실행
all: deps fmt lint test build

# 개발 모드 실행 (파일 변경 감지)
dev:
	@echo "$(GREEN)Starting in development mode with hot reload...$(NC)"
	@if command -v air >/dev/null 2>&1; then \
		air; \
	else \
		echo "Air not installed. Installing..."; \
		go install github.com/cosmtrek/air@latest; \
		air; \
	fi

# Kafka & DynamoDB Local 시작
kafka-up:
	@echo "$(GREEN)Starting Kafka and DynamoDB Local...$(NC)"
	docker-compose up -d kafka zookeeper dynamodb-local dynamodb-admin

# Kafka & DynamoDB Local 중지
kafka-down:
	@echo "$(GREEN)Stopping Kafka and DynamoDB Local...$(NC)"
	docker-compose down

# 전체 스택 실행 (Kafka + DynamoDB + App)
stack-up:
	@echo "$(GREEN)Starting full stack...$(NC)"
	@make kafka-up
	@sleep 5
	@make run

# 전체 스택 중지
stack-down:
	@echo "$(GREEN)Stopping full stack...$(NC)"
	@make kafka-down

# 프로덕션 빌드
build-prod:
	@echo "$(GREEN)Building for production...$(NC)"
	CGO_ENABLED=0 GOOS=linux $(GO) build -a -installsuffix cgo -o bin/$(APP_NAME) $(MAIN_PATH)

# 버전 정보 출력
version:
	@echo "$(GREEN)Version information:$(NC)"
	@$(GO) version
	@echo "App: $(APP_NAME)"

# Kafka 토픽 생성
kafka-topics:
	@echo "$(GREEN)Creating Kafka topics...$(NC)"
	docker exec kafka kafka-topics --create --topic order-events --bootstrap-server localhost:9092 --partitions 3 --replication-factor 1 --if-not-exists

# DynamoDB 테이블 생성 (로컬)
create-table:
	@echo "$(GREEN)Creating DynamoDB table...$(NC)"
	aws dynamodb create-table \
		--table-name orders \
		--attribute-definitions \
			AttributeName=PK,AttributeType=S \
			AttributeName=SK,AttributeType=S \
			AttributeName=GSI1PK,AttributeType=S \
			AttributeName=GSI1SK,AttributeType=S \
		--key-schema \
			AttributeName=PK,KeyType=HASH \
			AttributeName=SK,KeyType=RANGE \
		--global-secondary-indexes \
			IndexName=GSI1,Keys=[{AttributeName=GSI1PK,KeyType=HASH},{AttributeName=GSI1SK,KeyType=RANGE}],Projection={ProjectionType=ALL},BillingMode=PAY_PER_REQUEST \
		--billing-mode PAY_PER_REQUEST \
		--endpoint-url http://localhost:8000 \
		--region ap-northeast-2