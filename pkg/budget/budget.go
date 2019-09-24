package budget

import (
	"github.com/Optum/Dcs/pkg/awsiface"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/costexplorer"
)

// Define a Service, so we can mock this service from other components
// (eg, if I'm testing a Lambda controller that uses this Service)
//go:generate mockery -name Service
type Service interface {
	CalculateTotalSpend(startDate time.Time, endDate time.Time) (float64, error)
	SetCostExplorer(costExplorer awsiface.CostExplorerAPI)
}

// Define a concrete implementation of the Service interface
type AWSBudgetService struct {
	CostExplorer awsiface.CostExplorerAPI
}

func (budgetSvc *AWSBudgetService) SetCostExplorer(costExplorer awsiface.CostExplorerAPI) {
	budgetSvc.CostExplorer = costExplorer
}

// Implement the CalculateTotalSpend method of the Service interface
func (budgetSvc *AWSBudgetService) CalculateTotalSpend(startDate time.Time, endDate time.Time) (float64, error) {

	// CostExplorer uses strings for dates, in the format
	// of "2017-01-01"
	// Golang time formatting syntax is confusing as can be.
	// See https://stackoverflow.com/a/20234207
	timeFormat := "2006-01-02"
	timePeriod := costexplorer.DateInterval{
		Start: aws.String(startDate.UTC().Format(timeFormat)),
		End:   aws.String(endDate.UTC().Format(timeFormat)),
	}

	metrics := []*string{aws.String("UnblendedCost")}
	granularity := aws.String("DAILY")

	getCostAndUsageInput := costexplorer.GetCostAndUsageInput{
		Metrics:     metrics,
		TimePeriod:  &timePeriod,
		Granularity: granularity,
	}

	output, err := budgetSvc.CostExplorer.GetCostAndUsage(&getCostAndUsageInput)
	if err != nil {
		return 0, err
	}

	var totalCost float64

	for _, result := range output.ResultsByTime {
		cost, err := strconv.ParseFloat(*result.Total["UnblendedCost"].Amount, 64)
		if err != nil {
			return 0, err
		}

		totalCost = totalCost + cost

	}
	return totalCost, nil
}
