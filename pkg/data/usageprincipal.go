package data

import (
	"github.com/Optum/dce/pkg/usage"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
)

const usagePrincipalSkPrefix string = "Usage-Principal"

type usagePrincipalData struct {
	usage.Principal
	SK         string `json:"-" dynamodbav:"SK" schema:"-"`
	TimeToLive int64  `json:"timeToLive" dynamodbav:"TimeToLive,omitempty" schema:"timeToLive,omitempty"` // ttl attribute
}

// UsagePrincipal - Data Layer Struct
type UsagePrincipal struct {
	DynamoDB       dynamodbiface.DynamoDBAPI
	TableName      string `env:"PRINCIPAL_DB"`
	ConsistentRead bool   `env:"USE_CONSISTENT_READS" envDefault:"false"`
	Limit          int64  `env:"LIMIT" envDefault:"25"`
	BudgetPeriod   string `env:"PRINCIPAL_BUDGET_PERIOD" envDefault:"WEEKLY"`
	TimeToLive     int    `env:"USAGE_TTL" envDefault:"30"`
}
