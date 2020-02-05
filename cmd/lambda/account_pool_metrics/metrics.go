package main

import (
	"github.com/Optum/dce/pkg/account"
	"github.com/Optum/dce/pkg/account/accountiface"
	"log"
)

type AccountPoolMetrics struct {
	accountSvc   accountiface.Servicer
	Ready        int
	NotReady     int
	Leased       int
	Orphaned     int
}

func (m AccountPoolMetrics) Refresh() {

	type metricCount struct {
		status    account.Status
		count     int
	}
	metricsArr := []*metricCount{
		&metricCount{
			status: account.StatusReady,
			count: 0,
		},
		&metricCount{
			status: account.StatusNotReady,
			count: 0,
		},
		&metricCount{
			status: account.StatusLeased,
			count: 0,
		},
		&metricCount{
			status: account.StatusOrphaned,
			count: 0,
		},
	}
	for _, metric := range metricsArr {
		query := account.Account{
			ID:                  nil,
			Status:              &metric.status,
			LastModifiedOn:      nil,
			CreatedOn:           nil,
			AdminRoleArn:        nil,
			PrincipalRoleArn:    nil,
			PrincipalPolicyHash: nil,
			Metadata:            nil,
			Limit:               nil,
			NextID:              nil,
			PrincipalPolicyArn:  nil,
		}
		accounts, err := m.accountSvc.List(&query)
		if err != nil {
			log.Fatal("failed to query accounts by status", err)
		}
		metric.count = len(*accounts)
	}
}

func (m AccountPoolMetrics) Publish() {
	log.Println("Publish(): TODO")

}

func NewAccountPoolMetrics(accountSvc accountiface.Servicer) *AccountPoolMetrics {
	return &AccountPoolMetrics{accountSvc: accountSvc}
}