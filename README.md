# E-commerce MSA 시스템 - Order & Product Service

MSA(Microservice Architecture) 기반의 주문/상품 관리 시스템으로, Kafka를 통한 이벤트 기반 통신과 AWS DynamoDB를 사용합니다.

## 🏗️ 시스템 아키텍처

```
┌─────────────────┐    Kafka     ┌─────────────────┐
│  Order Service  │ ──events──→  │ Product Service │
│    (8080)       │              │     (8081)      │
└─────────────────┘              └─────────────────┘
         │                                │
         ▼                                ▼
┌─────────────────┐              ┌─────────────────┐
│ AWS DynamoDB    │              │ AWS DynamoDB    │
│ orders 테이블    │              │ products 테이블  │
└─────────────────┘              └─────────────────┘
```

### 핵심 기능
- **Order Service**: 주문 생성 및 관리, Kafka 이벤트 발행
- **Product Service**: 상품 관리, 재고 차감, Kafka 이벤트 구독
- **Event-Driven**: 주문 생성 시 자동 재고 차감
- **AWS DynamoDB**: 데이터 저장
- **Apache Kafka**: 서비스 간 비동기 통신

## 📋 사전 요구사항

### 필수 소프트웨어
- **Go** 1.21 이상
- **Docker & Docker Compose**
- **AWS CLI** 2.0 이상
- **AWS 계정** (DynamoDB 사용)

### 설치 확인
```bash
go version        # go version go1.21+ 
docker --version  # Docker version 20.0+
aws --version     # aws-cli/2.0+
```

## 🚀 빠른 시작

### 1. 프로젝트 클론
```bash
git clone https://github.com/cloud-wave-best-zizon/order-service.git
git clone https://github.com/cloud-wave-best-zizon/product-service.git
```

### 2. AWS 설정
```bash
# AWS 자격증명 설정
aws configure
# AWS Access Key ID: [실제 Access Key]
# AWS Secret Access Key: [실제 Secret Key]
# Default region name: ap-northeast-2
# Default output format: json

# 설정 확인
aws sts get-caller-identity
```

### 3. DynamoDB 테이블 생성
```bash
# Orders 테이블 생성
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
    'IndexName=GSI1,KeySchema=[{AttributeName=GSI1PK,KeyType=HASH},{AttributeName=GSI1SK,KeyType=RANGE}],Projection={ProjectionType=ALL}' \
  --billing-mode PAY_PER_REQUEST \
  --region ap-northeast-2

# Products 테이블 생성
aws dynamodb create-table \
  --table-name products-table \
  --attribute-definitions \
    AttributeName=PK,AttributeType=S \
    AttributeName=SK,AttributeType=S \
  --key-schema \
    AttributeName=PK,KeyType=HASH \
    AttributeName=SK,KeyType=RANGE \
  --billing-mode PAY_PER_REQUEST \
  --region ap-northeast-2

# 테이블 생성 확인
aws dynamodb list-tables --region ap-northeast-2
```

### 4. Kafka 실행
```bash
# Kafka 시작 (order-service 디렉토리에서)
cd order-service
docker compose up -d

# Kafka 준비 대기 (1-2분)
sleep 90

# 토픽 생성
docker compose exec kafka kafka-topics --create \
  --topic order-events \
  --bootstrap-server localhost:9092 \
  --partitions 3 --replication-factor 1

docker compose exec kafka kafka-topics --create \
  --topic order-compensation \
  --bootstrap-server localhost:9092 \
  --partitions 3 --replication-factor 1

# 토픽 확인
docker compose exec kafka kafka-topics --list --bootstrap-server localhost:9092
```

### 5. 환경변수 설정

**order-service/.env:**
```bash
PORT=8080
AWS_REGION=ap-northeast-2
ORDER_TABLE_NAME=orders
KAFKA_BROKERS=localhost:9092
LOG_LEVEL=info
```

**product-service/.env:**
```bash
PORT=8081
AWS_REGION=ap-northeast-2
PRODUCT_TABLE_NAME=products-table
KAFKA_BROKERS=localhost:9092
KAFKA_GROUP_ID=product-service
KAFKA_ENABLED=true
LOCAL_MODE=false
LOG_LEVEL=info
```

### 6. 서비스 실행

**터미널 1 - Product Service:**
```bash
cd product-service
go mod download
make run
```

**터미널 2 - Order Service:**
```bash
cd order-service
go mod download  
make run
```

### 7. 서비스 상태 확인
```bash
# Health Check
curl http://localhost:8081/api/v1/health  # Product Service
curl http://localhost:8080/api/v1/health  # Order Service
```

## 📖 API 사용법

### Product Service (포트 8081)

#### 1. 상품 등록
```bash
curl -X POST http://localhost:8081/api/v1/products \
  -H "Content-Type: application/json" \
  -d '{
    "product_id": "PROD001",
    "name": "MacBook Pro 14inch M3",
    "price": 2690000,
    "stock": 10
  }'
```

**응답:**
```json
{
  "product_id": "PROD001",
  "name": "MacBook Pro 14inch M3",
  "stock": 10,
  "price": 2690000
}
```

#### 2. 상품 조회
```bash
curl http://localhost:8081/api/v1/products/PROD001
```

#### 3. 수동 재고 차감 (테스트용)
```bash
curl -X POST http://localhost:8081/api/v1/products/PROD001/deduct \
  -H "Content-Type: application/json" \
  -d '{"quantity": 2}'
```

### Order Service (포트 8080)

#### 1. 주문 생성
```bash
curl -X POST http://localhost:8080/api/v1/orders \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user123",
    "items": [
      {
        "product_id": 1,
        "product_name": "MacBook Pro 14inch M3",
        "quantity": 2,
        "price": 2690000
      }
    ],
    "idempotency_key": "order-001"
  }'
```

**응답:**
```json
{
  "order_id": 1754966772678,
  "status": "PENDING",
  "message": "Order created successfully"
}
```

#### 2. 주문 조회
```bash
curl http://localhost:8080/api/v1/orders/1754966772678
```

## 🔄 Kafka 이벤트 플로우 테스트

### 1. Kafka 메시지 모니터링 시작

**터미널 3 - Order Events 모니터링:**
```bash
docker compose exec kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic order-events \
  --from-beginning \
  --property print.key=true \
  --property key.separator=" | "
```

### 2. 전체 플로우 테스트

```bash
# 1단계: 상품 등록
curl -X POST http://localhost:8081/api/v1/products \
  -H "Content-Type: application/json" \
  -d '{
    "product_id": "PROD001",
    "name": "MacBook Pro 14inch M3",
    "price": 2690000,
    "stock": 10
  }'

# 2단계: 초기 재고 확인
curl http://localhost:8081/api/v1/products/PROD001
# 예상 결과: "stock": 10

# 3단계: 주문 생성 (Kafka 이벤트 발행)
curl -X POST http://localhost:8080/api/v1/orders \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user123",
    "items": [
      {
        "product_id": 1,
        "product_name": "MacBook Pro 14inch M3",
        "quantity": 2,
        "price": 2690000
      }
    ],
    "idempotency_key": "kafka-test-001"
  }'

# 4단계: Kafka 메시지 확인
# 터미널 3에서 다음과 같은 메시지 확인:
# ORDER#1754966772678 | {"event_id":"...","order_id":1754966772678,...}

# 5단계: 재고 자동 차감 확인
curl http://localhost:8081/api/v1/products/PROD001
# 예상 결과: "stock": 8 (10 - 2 = 8)
```

### 3. 재고 부족 테스트

```bash
# 재고보다 많은 수량 주문
curl -X POST http://localhost:8080/api/v1/orders \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user456",
    "items": [
      {
        "product_id": 1,
        "product_name": "MacBook Pro 14inch M3",
        "quantity": 20,
        "price": 2690000
      }
    ],
    "idempotency_key": "insufficient-stock-test"
  }'

# 보상 이벤트 모니터링
docker compose exec kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic order-compensation \
  --from-beginning
```

## 🔍 모니터링 및 디버깅

### 1. 서비스 로그 확인

```bash
# Product Service 로그 (Kafka Consumer)
tail -f product-service.log | grep -i "kafka\|order\|stock"

# Order Service 로그 (Kafka Producer)  
tail -f order-service.log | grep -i "kafka\|order\|publish"
```

### 2. Kafka 상태 확인

```bash
# Consumer Group 상태
docker compose exec kafka kafka-consumer-groups --describe \
  --group product-service \
  --bootstrap-server localhost:9092

# 토픽 메시지 개수 확인
docker compose exec kafka kafka-run-class kafka.tools.GetOffsetShell \
  --broker-list localhost:9092 \
  --topic order-events

# 특정 메시지 확인
docker compose exec kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic order-events \
  --from-beginning \
  --max-messages 5
```

### 3. DynamoDB 데이터 확인

```bash
# Orders 데이터 확인
aws dynamodb scan --table-name orders --region ap-northeast-2 --max-items 5

# Products 데이터 확인  
aws dynamodb scan --table-name products-table --region ap-northeast-2 --max-items 5

# 특정 상품 조회
aws dynamodb get-item \
  --table-name products-table \
  --key '{"PK": {"S": "PRODUCT#PROD001"}, "SK": {"S": "METADATA"}}' \
  --region ap-northeast-2
```

## ⚙️ 고급 설정

### 1. Docker Compose로 전체 실행

**docker-compose.yml 예시:**
```yaml
services:
  zookeeper:
    image: confluentinc/cp-zookeeper:7.4.0
    environment:
      ZOOKEEPER_CLIENT_PORT: 2181
    ports:
      - "2181:2181"

  kafka:
    image: confluentinc/cp-kafka:7.4.0
    depends_on: [zookeeper]
    ports:
      - "9092:9092"
    environment:
      KAFKA_BROKER_ID: 1
      KAFKA_ZOOKEEPER_CONNECT: zookeeper:2181
      KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://localhost:9092
      KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: 1

  order-service:
    build: ./order-service
    ports:
      - "8080:8080"
    environment:
      - AWS_ACCESS_KEY_ID=${AWS_ACCESS_KEY_ID}
      - AWS_SECRET_ACCESS_KEY=${AWS_SECRET_ACCESS_KEY}
      - AWS_REGION=ap-northeast-2
      - KAFKA_BROKERS=kafka:9092
    depends_on: [kafka]

  product-service:
    build: ./product-service  
    ports:
      - "8081:8081"
    environment:
      - AWS_ACCESS_KEY_ID=${AWS_ACCESS_KEY_ID}
      - AWS_SECRET_ACCESS_KEY=${AWS_SECRET_ACCESS_KEY}
      - AWS_REGION=ap-northeast-2
      - KAFKA_BROKERS=kafka:9092
      - KAFKA_ENABLED=true
    depends_on: [kafka]
```

### 2. 환경별 설정

**개발 환경:**
```bash
# .env.development
LOG_LEVEL=debug
KAFKA_ENABLED=true
```

**프로덕션 환경:**
```bash
# .env.production  
LOG_LEVEL=info
KAFKA_ENABLED=true
```

## 🚨 문제 해결

### 일반적인 문제들

#### 1. AWS 연결 문제
**증상:** `UnrecognizedClientException`, `NoCredentialProviders`
**해결:**
```bash
# AWS 자격증명 재설정
aws configure

# 권한 확인
aws sts get-caller-identity
aws dynamodb list-tables --region ap-northeast-2
```

#### 2. Kafka 연결 문제
**증상:** `Failed to create Kafka consumer`
**해결:**
```bash
# Kafka 상태 확인
docker compose ps

# 포트 확인
lsof -i :9092

# Kafka 재시작
docker compose restart kafka
```

#### 3. 테이블 없음 에러
**증상:** `ResourceNotFoundException`
**해결:**
```bash
# 테이블 존재 확인
aws dynamodb describe-table --table-name orders --region ap-northeast-2

# 테이블 생성 (위의 DynamoDB 설정 참조)
```

#### 4. 포트 충돌
**증상:** `bind: address already in use`
**해결:**
```bash
# 포트 사용 프로세스 확인
lsof -i :8080
lsof -i :8081

# 프로세스 종료 후 재시작
```

### 로그 분석

**성공적인 플로우 로그:**
```
# Order Service
{"level":"info","msg":"Order created successfully","order_id":123,"user_id":"user123"}
{"level":"info","msg":"Kafka event published","order_id":123}

# Product Service  
{"level":"info","msg":"Kafka consumer started","topics":["order-events"]}
{"level":"info","msg":"Processing order created event","order_id":123}
{"level":"info","msg":"Stock deducted successfully","product_id":"PROD001","previous_stock":10,"new_stock":8}
```

## 📊 성능 모니터링

### 메트릭 확인

```bash
# Consumer Lag 확인
docker compose exec kafka kafka-consumer-groups --describe \
  --group product-service \
  --bootstrap-server localhost:9092

# DynamoDB 사용량 확인 
aws cloudwatch get-metric-statistics \
  --namespace AWS/DynamoDB \
  --metric-name ConsumedReadCapacityUnits \
  --dimensions Name=TableName,Value=orders \
  --start-time 2025-08-12T00:00:00Z \
  --end-time 2025-08-12T23:59:59Z \
  --period 3600 \
  --statistics Sum
```

## 🔄 정리

### 환경 중지
```bash
# Kafka 중지
docker compose down

# AWS 리소스 정리 (선택사항)
aws dynamodb delete-table --table-name orders --region ap-northeast-2
aws dynamodb delete-table --table-name products-table --region ap-northeast-2
```

## 📚 추가 리소스

- **Kafka Documentation**: https://kafka.apache.org/documentation/
- **AWS DynamoDB Guide**: https://docs.aws.amazon.com/dynamodb/
- **Go Kafka Library**: https://github.com/confluentinc/confluent-kafka-go
- **AWS SDK for Go**: https://aws.amazon.com/sdk-for-go/