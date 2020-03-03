package main

import (
	"github.com/Optum/dce/pkg/config"
	"github.com/Optum/dce/pkg/data"
	"github.com/Optum/dce/pkg/lease"
	"github.com/Optum/dce/pkg/lease/leaseiface/mocks"
	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"strconv"
	"testing"
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
			retErr            error
		}{
			{
				name:      "new usage lease summary over lease budget is ended",
				eventName: "INSERT",
				sortKey:   data.UsageLeaseSkSummaryPrefix+"-123",
				costAmount: "100.0",
				budgetAmount: "99.0",
				expIsLeaseDeleted: true,
				retErr: nil,
			},
			{
				name:              "new usage lease summary under lease budget is not ended",
				eventName:         "INSERT",
				sortKey:           data.UsageLeaseSkSummaryPrefix+"-123",
				costAmount:        "99.0",
				budgetAmount:      "100.0",
				expIsLeaseDeleted: false,
				retErr: nil,
			},
			//{
			//	name:      "updated usage lease summary over lease budget is ended",
			//	eventName: "INSERT",
			//	sortKey:   data.UsageLeaseSkSummaryPrefix+"-123",
			//	costAmount: "100.0",
			//	budgetAmount: "99.0",
			//	expIsLeaseDeleted: true,
			//	retErr: nil,
			//},
			//{
			//	name:              "updated usage lease summary under lease budget is not ended",
			//	eventName:         "INSERT",
			//	sortKey:           data.UsageLeaseSkSummaryPrefix+"-123",
			//	costAmount:        "99.0",
			//	budgetAmount:      "100.0",
			//	expIsLeaseDeleted: false,
			//	retErr: nil,
			//},
			//{
			//	name:      "deleting usage lease summary over lease budget does not delete lease",
			//	eventName: "REMOVE",
			//	sortKey:   data.UsageLeaseSkSummaryPrefix+"-123",
			//	costAmount: "100.0",
			//	budgetAmount: "99.0",
			//	expIsLeaseDeleted: true,
			//	retErr: nil,
			//},
			//{
			//	name:              "deleting usage lease summary under lease budget lease does not try to delete lease",
			//	eventName:         "REMOVE",
			//	sortKey:           data.UsageLeaseSkSummaryPrefix+"-123",
			//	costAmount:        "99.0",
			//	budgetAmount:      "100.0",
			//	expIsLeaseDeleted: false,
			//	retErr: nil,
			//},
			//{
			//	name:              "delete lease returns error",
			//	eventName:         "INSERT",
			//	sortKey:           data.UsageLeaseSkSummaryPrefix+"-123",
			//	costAmount:        "99.0",
			//	budgetAmount:      "100.0",
			//	expIsLeaseDeleted: false,
			//	retDeleteErr: nil,
			//},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				event := events.DynamoDBEvent{
					Records: []events.DynamoDBEventRecord{
						{
							AWSRegion:      "",
							Change:         events.DynamoDBStreamRecord{
								NewImage:   map[string]events.DynamoDBAttributeValue{
									"SK": events.NewStringAttribute(tt.sortKey),
									"LeaseId": events.NewStringAttribute(""),
									"CostAmount": events.NewNumberAttribute(tt.costAmount),
									"BudgetAmount": events.NewNumberAttribute(tt.budgetAmount),
								},
							},
							EventName:      tt.eventName,
						},
					},
				}

				cfgBldr := &config.ConfigurationBuilder{}
				svcBldr := &config.ServiceBuilder{Config: cfgBldr}
				leaseSvc := mocks.Servicer{}
				if tt.expIsLeaseDeleted {
					leaseSvc.On("Delete", mock.Anything).Return(&lease.Lease{}, nil).Return(&lease.Lease{}, nil)
				}
				svcBldr.Config.WithService(&leaseSvc)
				_, err := svcBldr.Build()

				Services = svcBldr

				err = handler(nil, event)

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
			retDeleteErr      error
			retListErr      error
		}{
			{
				name:              "new usage principal summary over principal budget is ended",
				eventName:         "INSERT",
				sortKey:           data.UsagePrincipalSkPrefix+"-123",
				costAmount:        "100.0",
				budgetAmount:      "99.0",
				expIsLeaseDeleted: true,
				retDeleteErr:      nil,
				retListErr:      nil,
			},
			{
				name:              "new usage principal summary under principal budget is not ended",
				eventName:         "INSERT",
				sortKey:           data.UsagePrincipalSkPrefix+"-123",
				costAmount:        "99.0",
				budgetAmount:      "100.0",
				expIsLeaseDeleted: false,
				retDeleteErr:      nil,
				retListErr:        nil,
			},
			//{
			//	name:              "updated usage principal summary over principal budget is ended",
			//	eventName:         "INSERT",
			//	sortKey:           data.UsagePrincipalSkPrefix+"-123",
			//	costAmount:        "100.0",
			//	budgetAmount:      "99.0",
			//	expIsLeaseDeleted: true,
			//	retDeleteErr:      nil,
			//	retListErr:      nil,
			//},
			//{
			//	name:              "updated usage principal summary under principal budget is not ended",
			//	eventName:         "INSERT",
			//	sortKey:           data.UsagePrincipalSkPrefix+"-123",
			//	costAmount:        "99.0",
			//	budgetAmount:      "100.0",
			//	expIsLeaseDeleted: false,
			//	retDeleteErr:      nil,
			//	retListErr:        nil,
			//},
			//{
			//	name:              "deleting usage principal summary over lease budget does not delete lease",
			//	eventName:         "INSERT",
			//	sortKey:           data.UsagePrincipalSkPrefix+"-123",
			//	costAmount:        "100.0",
			//	budgetAmount:      "99.0",
			//	expIsLeaseDeleted: true,
			//	retDeleteErr:      nil,
			//	retListErr:      nil,
			//},
			//{
			//	name:              "deleting usage principal summary under lease budget does not delete lease",
			//	eventName:         "INSERT",
			//	sortKey:           data.UsagePrincipalSkPrefix+"-123",
			//	costAmount:        "99.0",
			//	budgetAmount:      "100.0",
			//	expIsLeaseDeleted: false,
			//	retDeleteErr:      nil,
			//	retListErr:        nil,
			//},
			//{
			//	name:              "list leases returns error,
			//	eventName:         "INSERT",
			//	sortKey:           data.UsagePrincipalSkPrefix+"-123",
			//	costAmount:        "99.0",
			//	budgetAmount:      "100.0",
			//	expIsLeaseDeleted: false,
			//	retDeleteErr:      nil,
			//	retListErr:        nil,
			//},
			//{
			//	name:              "delete lease returns error,
			//	eventName:         "INSERT",
			//	sortKey:           data.UsagePrincipalSkPrefix+"-123",
			//	costAmount:        "99.0",
			//	budgetAmount:      "100.0",
			//	expIsLeaseDeleted: false,
			//	retDeleteErr:      nil,
			//	retListErr:        nil,
			//},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				event := events.DynamoDBEvent{
					Records: []events.DynamoDBEventRecord{
						{
							AWSRegion:      "",
							Change:         events.DynamoDBStreamRecord{
								NewImage:   map[string]events.DynamoDBAttributeValue{
									"SK": events.NewStringAttribute(tt.sortKey),
									"PrincipalId": events.NewStringAttribute(""),
									"CostAmount": events.NewNumberAttribute(tt.costAmount),
								},
							},
							EventName:      tt.eventName,
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
					leaseSvc.On("ListPages", mock.Anything, mock.Anything).Return(&lease.Lease{}, nil).Return(&lease.Lease{}, nil)
					leaseSvc.On("Delete", mock.Anything).Return(&lease.Lease{}, nil).Return(nil)
				}
				svcBldr.Config.WithService(&leaseSvc)
				_, err = svcBldr.Build()

				Services = svcBldr

				err = handler(nil, event)

				leaseSvc.AssertExpectations(t)
				assert.Nil(t, err)
			})
		}
	})
}