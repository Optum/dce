package accountmanager

import (
	"testing"
	"time"

	"github.com/Optum/dce/internal/types"
	"github.com/Optum/dce/pkg/accountmanager/mocks"
	awsMocks "github.com/Optum/dce/pkg/awsiface/mocks"
	"github.com/Optum/dce/pkg/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/costexplorer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGettingUsage(t *testing.T) {
	now := time.Date(2020, 2, 23, 12, 0, 0, 0, time.UTC)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	yesterday := today.AddDate(0, 0, -1)
	// tomorrow := tomorrow.AddDate(0, 0, 1)

	type output struct {
		data types.Usages
		err  error
	}

	type ceOutput struct {
		data *costexplorer.GetCostAndUsageOutput
		err  awserr.Error
	}

	tests := []struct {
		name       string
		inputStart time.Time
		inputEnd   time.Time
		ceInput    *costexplorer.GetCostAndUsageInput
		ceOutput   ceOutput
		exp        output
	}{
		{
			name:       "should create role and policy and pass",
			inputStart: yesterday,
			inputEnd:   today,
			ceInput: &costexplorer.GetCostAndUsageInput{
				Granularity: aws.String("DAILY"),
				Metrics: []*string{
					aws.String("UnblendedCost"),
				},
				TimePeriod: &costexplorer.DateInterval{
					Start: aws.String("2020-02-22"),
					End:   aws.String("2020-02-24"),
				},
			},
			ceOutput: ceOutput{
				data: &costexplorer.GetCostAndUsageOutput{
					ResultsByTime: []*costexplorer.ResultByTime{
						{
							TimePeriod: &costexplorer.DateInterval{
								Start: aws.String("2020-02-22"),
								End:   aws.String("2020-02-22"),
							},
							Total: map[string]*costexplorer.MetricValue{
								"UnblendedCost": {
									Amount: aws.String("1.0"),
									Unit:   aws.String("USD"),
								},
							},
						},
					},
				},
			},
			exp: output{
				data: types.Usages{
					{
						TimePeriod: yesterday,
						Amount:     1.0,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ceSvc := &awsMocks.CostExplorerAPI{}
			ceSvc.On("GetCostAndUsage", tt.ceInput).
				Return(tt.ceOutput.data, tt.ceOutput.err)

			clientSvc := &mocks.Clienter{}
			clientSvc.On("IAM", mock.Anything).Return(ceSvc)

			usageSvc := usageService{
				ceSvc:  ceSvc,
				config: testConfig,
			}

			usgs, err := usageSvc.GetUsage(tt.inputStart, tt.inputEnd)
			assert.True(t, errors.Is(err, tt.exp.err), "actual error %+v doesn't match expected error %+v", err, tt.exp)
			if tt.exp.err == nil {
				assert.Equal(t, tt.exp.data, usgs)
			}
		})
	}
}
