package config

import (
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Port             string `envconfig:"PORT" default:"8080"`
	AWSRegion        string `envconfig:"AWS_REGION" default:"ap-northeast-2"`
	OrderTableName   string `envconfig:"ORDER_TABLE_NAME" default:"orders"`
	KafkaBrokers     string `envconfig:"KAFKA_BROKERS" default:"localhost:9092"`
	LogLevel         string `envconfig:"LOG_LEVEL" default:"info"`
	DynamoDBEndpoint string `envconfig:"DYNAMODB_ENDPOINT" default:""` // DynamoDB Local 엔드포인트
}

func Load() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}