package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
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
	awsCfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(cfg.AWSRegion),
	)
	if err != nil {
		return nil, err
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

	// PK, SK 추가
	av["PK"] = &types.AttributeValueMemberS{Value: fmt.Sprintf("ORDER#%s", order.OrderID)}
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

var ErrOrderNotFound = errors.New("order not found")
