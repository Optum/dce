package main

import (
	"context"
	"strconv"
	"testing"

	"github.com/Optum/dce/pkg/config"
	"github.com/Optum/dce/pkg/data"
	"github.com/Optum/dce/pkg/lease"
	"github.com/Optum/dce/pkg/lease/leaseiface/mocks"
	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestEndLeaseOverBudget(t *testing.T) {

	t.Run("end lease over lease budget", func(t *testing.T) {
		tests := []struct {
			name              string
			eventName         string
			sortKey           string
			costAmount        string
			budgetAmount      string
			expIsLeaseDeleted bool
		}{
			{
				name:              "new usage lease summary over lease budget ends lease",
				eventName:         "INSERT",
				sortKey:           data.UsageLeaseSkSummaryPrefix + "-123",
				costAmount:        "100.0",
				budgetAmount:      "99.0",
				expIsLeaseDeleted: true,
			},
			{
				name:              "new usage lease summary under lease budget does not end lease",
				eventName:         "INSERT",
				sortKey:           data.UsageLeaseSkSummaryPrefix + "-123",
				costAmount:        "99.0",
				budgetAmount:      "100.0",
				expIsLeaseDeleted: false,
			},
			{
				name:              "updated usage lease summary over lease budget ends lease",
				eventName:         "INSERT",
				sortKey:           data.UsageLeaseSkSummaryPrefix + "-123",
				costAmount:        "100.0",
				budgetAmount:      "99.0",
				expIsLeaseDeleted: true,
			},
			{
				name:              "updated usage lease summary under lease budget does not end lease",
				eventName:         "INSERT",
				sortKey:           data.UsageLeaseSkSummaryPrefix + "-123",
				costAmount:        "99.0",
				budgetAmount:      "100.0",
				expIsLeaseDeleted: false,
			},
			{
				name:              "deleting usage lease summary over lease budget does not end lease",
				eventName:         "REMOVE",
				sortKey:           data.UsageLeaseSkSummaryPrefix + "-123",
				costAmount:        "100.0",
				budgetAmount:      "99.0",
				expIsLeaseDeleted: false,
			},
			{
				name:              "deleting usage lease summary under lease budget does not end lease",
				eventName:         "REMOVE",
				sortKey:           data.UsageLeaseSkSummaryPrefix + "-123",
				costAmount:        "99.0",
				budgetAmount:      "100.0",
				expIsLeaseDeleted: false,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				event := events.DynamoDBEvent{
					Records: []events.DynamoDBEventRecord{
						{
							AWSRegion: "",
							Change: events.DynamoDBStreamRecord{
								NewImage: map[string]events.DynamoDBAttributeValue{
									"SK":           events.NewStringAttribute(tt.sortKey),
									"LeaseId":      events.NewStringAttribute(""),
									"CostAmount":   events.NewNumberAttribute(tt.costAmount),
									"BudgetAmount": events.NewNumberAttribute(tt.budgetAmount),
								},
							},
							EventName: tt.eventName,
						},
					},
				}

				cfgBldr := &config.ConfigurationBuilder{}
				svcBldr := &config.ServiceBuilder{Config: cfgBldr}
				leaseSvc := mocks.Servicer{}
				if tt.expIsLeaseDeleted {
					leaseSvc.On("Delete", mock.Anything).Return(&lease.Lease{}, nil)
				}
				svcBldr.Config.WithService(&leaseSvc)
				_, _ = svcBldr.Build()

				Services = svcBldr

				err := handler(context.TODO(), event)

				leaseSvc.AssertExpectations(t)
				assert.Nil(t, err)
			})
		}
	})

	t.Run("end lease over principal budget", func(t *testing.T) {
		tests := []struct {
			name              string
			eventName         string
			sortKey           string
			costAmount        string
			budgetAmount      string
			expIsLeaseDeleted bool
		}{
			{
				name:              "new usage principal summary over principal budget ends lease",
				eventName:         "INSERT",
				sortKey:           data.UsagePrincipalSkPrefix + "-123",
				costAmount:        "100.0",
				budgetAmount:      "99.0",
				expIsLeaseDeleted: true,
			},
			{
				name:              "new usage principal summary under principal budget does not end lease",
				eventName:         "INSERT",
				sortKey:           data.UsagePrincipalSkPrefix + "-123",
				costAmount:        "99.0",
				budgetAmount:      "100.0",
				expIsLeaseDeleted: false,
			},
			{
				name:              "updated usage principal summary over principal budget ends lease",
				eventName:         "INSERT",
				sortKey:           data.UsagePrincipalSkPrefix + "-123",
				costAmount:        "100.0",
				budgetAmount:      "99.0",
				expIsLeaseDeleted: true,
			},
			{
				name:              "updated usage principal summary under principal budget does not end lease",
				eventName:         "INSERT",
				sortKey:           data.UsagePrincipalSkPrefix + "-123",
				costAmount:        "99.0",
				budgetAmount:      "100.0",
				expIsLeaseDeleted: false,
			},
			{
				name:              "deleting usage principal summary over lease budget does not end lease",
				eventName:         "REMOVE",
				sortKey:           data.UsagePrincipalSkPrefix + "-123",
				costAmount:        "100.0",
				budgetAmount:      "99.0",
				expIsLeaseDeleted: false,
			},
			{
				name:              "deleting usage principal summary under lease budget does not end lease",
				eventName:         "REMOVE",
				sortKey:           data.UsagePrincipalSkPrefix + "-123",
				costAmount:        "99.0",
				budgetAmount:      "100.0",
				expIsLeaseDeleted: false,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				event := events.DynamoDBEvent{
					Records: []events.DynamoDBEventRecord{
						{
							AWSRegion: "",
							Change: events.DynamoDBStreamRecord{
								NewImage: map[string]events.DynamoDBAttributeValue{
									"SK":          events.NewStringAttribute(tt.sortKey),
									"PrincipalId": events.NewStringAttribute(""),
									"CostAmount":  events.NewNumberAttribute(tt.costAmount),
								},
							},
							EventName: tt.eventName,
						},
					},
				}

				principalBudgetFlt, err := strconv.ParseFloat(tt.budgetAmount, 64)
				assert.Nil(t, err)
				Settings = &lambdaConfig{
					PrincipalBudgetAmount: principalBudgetFlt,
				}

				cfgBldr := &config.ConfigurationBuilder{}
				svcBldr := &config.ServiceBuilder{Config: cfgBldr}
				leaseSvc := mocks.Servicer{}
				if tt.expIsLeaseDeleted {
					leaseSvc.On("ListPages", mock.Anything, mock.Anything).Return(nil)
				}
				svcBldr.Config.WithService(&leaseSvc)
				_, _ = svcBldr.Build()

				Services = svcBldr

				err = handler(context.TODO(), event)

				leaseSvc.AssertExpectations(t)
				assert.Nil(t, err)
			})
		}
	})
}
