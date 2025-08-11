package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/cloud-wave-best-zizon/order-service/internal/domain"
	pkgconfig "github.com/cloud-wave-best-zizon/order-service/pkg/config"
)

type OrderRepository struct {
	client    *dynamodb.Client
	tableName string
}

func NewDynamoDBClient(cfg *pkgconfig.Config) (*dynamodb.Client, error) {
	var awsCfg aws.Config
	var err error

	if cfg.DynamoDBEndpoint != "" {
		// DynamoDB Local 사용
		customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			if service == dynamodb.ServiceID {
				return aws.Endpoint{
					URL:           cfg.DynamoDBEndpoint,
					SigningRegion: cfg.AWSRegion,
				}, nil
			}
			return aws.Endpoint{}, &aws.EndpointNotFoundError{}
		})

		awsCfg, err = config.LoadDefaultConfig(context.TODO(),
			config.WithRegion(cfg.AWSRegion),
			config.WithEndpointResolverWithOptions(customResolver),
			config.WithCredentialsProvider(credentials.StaticCredentialsProvider{
				Value: aws.Credentials{
					AccessKeyID:     "dummy",
					SecretAccessKey: "dummy",
					SessionToken:    "",
				},
			}),
		)
	} else {
		// 실제 AWS DynamoDB 사용
		awsCfg, err = config.LoadDefaultConfig(context.TODO(),
			config.WithRegion(cfg.AWSRegion),
		)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return dynamodb.NewFromConfig(awsCfg), nil
}

func NewOrderRepository(client *dynamodb.Client, tableName string) *OrderRepository {
	return &OrderRepository{
		client:    client,
		tableName: tableName,
	}
}

func (r *OrderRepository) CreateOrder(ctx context.Context, order *domain.Order) error {
	// Order를 DynamoDB 아이템으로 변환
	av, err := attributevalue.MarshalMap(order)
	if err != nil {
		return fmt.Errorf("failed to marshal order: %w", err)
	}

	// PK, SK 추가 - OrderID는 int이므로 %d 사용
	av["PK"] = &types.AttributeValueMemberS{Value: fmt.Sprintf("ORDER#%d", order.OrderID)}
	av["SK"] = &types.AttributeValueMemberS{Value: "METADATA"}
	av["GSI1PK"] = &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s", order.UserID)}
	av["GSI1SK"] = &types.AttributeValueMemberS{Value: fmt.Sprintf("ORDER#%s", order.CreatedAt.Format("2006-01-02T15:04:05Z"))}

	// DynamoDB에 저장
	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item:      av,
	})

	if err != nil {
		return fmt.Errorf("failed to put item: %w", err)
	}

	return nil
}

func (r *OrderRepository) GetOrder(ctx context.Context, id int) (*domain.Order, error) {
	pk := fmt.Sprintf("ORDER#%d", id)

	out, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: pk},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
		ConsistentRead: aws.Bool(true),
	})
	if err != nil {
		return nil, err
	}
	if out.Item == nil || len(out.Item) == 0 {
		return nil, ErrOrderNotFound
	}

	var order domain.Order
	if err := attributevalue.UnmarshalMap(out.Item, &order); err != nil {
		return nil, err
	}
	return &order, nil
}

// GetOrdersByUser - 특정 사용자의 주문 목록 조회
func (r *OrderRepository) GetOrdersByUser(ctx context.Context, userID string, limit int32) ([]*domain.Order, error) {
	gsi1pk := fmt.Sprintf("USER#%s", userID)

	out, err := r.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		IndexName:              aws.String("GSI1"),
		KeyConditionExpression: aws.String("GSI1PK = :gsi1pk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":gsi1pk": &types.AttributeValueMemberS{Value: gsi1pk},
		},
		Limit:            aws.Int32(limit),
		ScanIndexForward: aws.Bool(false), // 최신 주문부터
	})

	if err != nil {
		return nil, fmt.Errorf("failed to query orders: %w", err)
	}

	orders := make([]*domain.Order, 0, len(out.Items))
	for _, item := range out.Items {
		var order domain.Order
		if err := attributevalue.UnmarshalMap(item, &order); err != nil {
			return nil, err
		}
		orders = append(orders, &order)
	}

	return orders, nil
}

var ErrOrderNotFound = errors.New("order not found")