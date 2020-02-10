package main

import (
	"github.com/Optum/dce/pkg/account"
	accountMocks "github.com/Optum/dce/pkg/account/accountiface/mocks"
	awsMocks "github.com/Optum/dce/pkg/awsiface/mocks"
	"github.com/Optum/dce/pkg/config"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

func TestRefreshMetrics(t *testing.T) {

	tests := []struct {
		name   string
		status account.Status
	}{
		{
			name:   "get ready accounts",
			status: account.StatusReady,
		},
		{
			name:   "get not ready accounts",
			status: account.StatusNotReady,
		},
		{
			name:   "get leased accounts",
			status: account.StatusLeased,
		},
		{
			name:   "get orphaned accounts",
			status: account.StatusOrphaned,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// arrange
			accountSvc := accountMocks.Servicer{}
			accountSvc.On("List", mock.MatchedBy(func(query *account.Account) bool {
				// assert
				return *query.Status == tt.status &&
					*query.Limit == QueryLimit
			})).Return(
				&account.Accounts{}, nil,
			)
			cfgBldr := &config.ConfigurationBuilder{}
			svcBldr := &config.ServiceBuilder{Config: cfgBldr}
			svcBldr.Config.WithService(&accountSvc)
			_, err := svcBldr.Build()
			assert.Nil(t, err)
			if err == nil {
				Services = svcBldr
			}

			// act
			getMetric(tt.status)
		})
	}
}

func TestPublishMetrics(t *testing.T) {
	t.Run("publish metrics", func(t *testing.T) {
		// arrange
		namespace := "testNamespace"
		countMetric1 := CountMetric{
			name:  "testmetric1",
			count: 1,
		}

		cloudwatchSvc := awsMocks.CloudWatchAPI{}
		cloudwatchSvc.On("PutMetricData", mock.MatchedBy(func(input *cloudwatch.PutMetricDataInput) bool {
			// assert
			namespaceEqual := *input.Namespace == namespace

			firstMetricEqual := *input.MetricData[0].MetricName == countMetric1.name+"Accounts" &&
				*input.MetricData[0].Value == float64(countMetric1.count)

			return namespaceEqual &&
				firstMetricEqual
		})).Return(nil, nil)

		cfgBldr := &config.ConfigurationBuilder{}
		svcBldr := &config.ServiceBuilder{Config: cfgBldr}
		svcBldr.Config.WithService(&cloudwatchSvc)
		_, err := svcBldr.Build()
		assert.Nil(t, err)
		if err == nil {
			Services = svcBldr
		}

		// act
		publishMetrics(namespace, countMetric1)
	})
}
