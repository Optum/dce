package accountmanager

import (
	"strconv"
	"time"

	"github.com/Optum/dce/internal/types"
	"github.com/Optum/dce/pkg/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/costexplorer"
	"github.com/aws/aws-sdk-go/service/costexplorer/costexploreriface"
)

const (
	layoutISO = "2006-01-02"
)

type usageService struct {
	ceSvc  costexploreriface.CostExplorerAPI
	config ServiceConfig
}

func (p *usageService) GetUsage(startDate time.Time, endDate time.Time) (types.Usages, error) {

	usageOutputs, err := p.ceSvc.GetCostAndUsage(
		&costexplorer.GetCostAndUsageInput{
			Metrics:     []*string{aws.String("UnblendedCost")},
			Granularity: aws.String("DAILY"),
			TimePeriod: &costexplorer.DateInterval{
				Start: aws.String(startDate.Format(layoutISO)),
				// add a date because CE is exlusive and we are designed for inclusive
				End: aws.String(endDate.AddDate(0, 0, 1).Format(layoutISO)),
			},
		},
	)
	if err != nil {
		return nil, errors.NewInternalServer("error getting usage information", err)
	}

	result := types.Usages{}
	for _, usageOutput := range usageOutputs.ResultsByTime {
		cost, err := strconv.ParseFloat(*usageOutput.Total["UnblendedCost"].Amount, 64)
		if err != nil {
			return nil, err
		}

		period, err := time.Parse("2006-01-02", *usageOutput.TimePeriod.Start)
		if err != nil {
			return nil, err
		}
		result = append(result, types.Usage{
			TimePeriod: period,
			Amount:     cost,
		})

	}

	return result, nil
}
