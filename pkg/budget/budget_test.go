package budget

import (
	"github.com/Optum/dce/pkg/awsiface/mocks"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/costexplorer"
	"github.com/stretchr/testify/assert"
)

func TestCalculateTotalSpend(t *testing.T) {
	// Mock the CostExplorer SDK
	costExplorer := &mocks.CostExplorerAPI{}
	costExplorer.On("GetCostAndUsage", &costexplorer.GetCostAndUsageInput{
		Metrics:     []*string{aws.String("UnblendedCost")},
		Granularity: aws.String("DAILY"),
		TimePeriod: &costexplorer.DateInterval{
			Start: aws.String("1970-01-01"),
			End:   aws.String("1970-01-02"),
		},
	}).Return(&costexplorer.GetCostAndUsageOutput{
		ResultsByTime: []*costexplorer.ResultByTime{
			{
				Total: map[string]*costexplorer.MetricValue{
					"UnblendedCost": {
						Amount: aws.String("100"),
						Unit:   aws.String("USD"),
					},
				},
			},
			{
				Total: map[string]*costexplorer.MetricValue{
					"UnblendedCost": {
						Amount: aws.String("50"),
						Unit:   aws.String("USD"),
					},
				},
			},
		},
	}, nil)

	budgetSvc := AWSBudgetService{
		CostExplorer: costExplorer,
	}
	cost, err := budgetSvc.CalculateTotalSpend(
		time.Unix(0, 0),
		time.Unix(0, 0).Add(time.Hour*24),
	)
	assert.Nil(t, err, "There should be no errors")
	assert.Equal(t, cost, float64(150))
}
