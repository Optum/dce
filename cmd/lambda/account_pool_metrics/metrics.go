package main

import (
	"github.com/Optum/dce/pkg/account"
	"github.com/Optum/dce/pkg/account/accountiface"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"log"
)

type MetricCount struct {
	status account.Status
	count  int
}

type AccountPoolMetrics struct {
	accountSvc accountiface.Servicer
	Ready      MetricCount
	NotReady   MetricCount
	Leased     MetricCount
	Orphaned   MetricCount
}

// Hardcoding until pagination function is implemented
var QUERY_LIMIT int64 = 20000

func (m *AccountPoolMetrics) Refresh() {

	readyQuery := account.Account{
		ID:                  nil,
		Status:              &m.Ready.status,
		LastModifiedOn:      nil,
		CreatedOn:           nil,
		AdminRoleArn:        nil,
		PrincipalRoleArn:    nil,
		PrincipalPolicyHash: nil,
		Metadata:            nil,
		Limit:               &QUERY_LIMIT,
		NextID:              nil,
		PrincipalPolicyArn:  nil,
	}
	readyAccounts, err := m.accountSvc.List(&readyQuery)
	if err != nil {
		log.Fatal("failed to query accounts by status, ", err)
	}
	m.Ready.count = len(*readyAccounts)

	notReadyQuery := account.Account{
		ID:                  nil,
		Status:              &m.NotReady.status,
		LastModifiedOn:      nil,
		CreatedOn:           nil,
		AdminRoleArn:        nil,
		PrincipalRoleArn:    nil,
		PrincipalPolicyHash: nil,
		Metadata:            nil,
		Limit:               &QUERY_LIMIT,
		NextID:              nil,
		PrincipalPolicyArn:  nil,
	}
	notReadyAccounts, err := m.accountSvc.List(&notReadyQuery)
	if err != nil {
		log.Fatal("failed to query accounts by status, ", err)
	}
	m.NotReady.count = len(*notReadyAccounts)

	leasedQuery := account.Account{
		ID:                  nil,
		Status:              &m.Leased.status,
		LastModifiedOn:      nil,
		CreatedOn:           nil,
		AdminRoleArn:        nil,
		PrincipalRoleArn:    nil,
		PrincipalPolicyHash: nil,
		Metadata:            nil,
		Limit:               &QUERY_LIMIT,
		NextID:              nil,
		PrincipalPolicyArn:  nil,
	}
	leasedAccounts, err := m.accountSvc.List(&leasedQuery)
	if err != nil {
		log.Fatal("failed to query accounts by status, ", err)
	}
	m.Leased.count = len(*leasedAccounts)

	orphanedQuery := account.Account{
		ID:                  nil,
		Status:              &m.Orphaned.status,
		LastModifiedOn:      nil,
		CreatedOn:           nil,
		AdminRoleArn:        nil,
		PrincipalRoleArn:    nil,
		PrincipalPolicyHash: nil,
		Metadata:            nil,
		Limit:               &QUERY_LIMIT,
		NextID:              nil,
		PrincipalPolicyArn:  nil,
	}
	orphanedAccounts, err := m.accountSvc.List(&orphanedQuery)
	if err != nil {
		log.Fatal("failed to query accounts by status, ", err)
	}
	m.Orphaned.count = len(*orphanedAccounts)

	log.Println("Found ", m.Ready.count, m.Ready.status, " accounts")
	log.Println("Found ", m.NotReady.count, m.NotReady.status, " accounts")
	log.Println("Found ", m.Leased.count, m.Leased.status, " accounts")
	log.Println("Found ", m.Orphaned.count, m.Orphaned.status, " accounts")
}

func (m *AccountPoolMetrics) Publish() {
	log.Println("Publishing metrics to cloudwatch")

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	svc := cloudwatch.New(sess)

	log.Println("Publishing ReadyAccount Metric: ", float64(m.Ready.count))
	log.Println("Publishing NotReadyAccounts Metric: ", float64(m.NotReady.count))
	log.Println("Publishing LeasedAccounts Metric: ", float64(m.Leased.count))
	log.Println("Publishing OrphanedAccounts Metric: ", float64(m.Orphaned.count))

	_, err := svc.PutMetricData(&cloudwatch.PutMetricDataInput{
		Namespace: aws.String("DCE/AccountPool"),
		MetricData: []*cloudwatch.MetricDatum{
			{
				MetricName: aws.String("ReadyAccounts"),
				Unit:       aws.String("Count"),
				Value:      aws.Float64(float64(m.Ready.count)),
			},
			{
				MetricName: aws.String("NotReadyAccounts"),
				Unit:       aws.String("Count"),
				Value:      aws.Float64(float64(m.NotReady.count)),
			},
			{
				MetricName: aws.String("LeasedAccounts"),
				Unit:       aws.String("Count"),
				Value:      aws.Float64(float64(m.Leased.count)),
			},
			{
				MetricName: aws.String("OrphanedAccounts"),
				Unit:       aws.String("Count"),
				Value:      aws.Float64(float64(m.Orphaned.count)),
			},
		},
	})
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("Published metrics")
}

func NewAccountPoolMetrics(accountSvc accountiface.Servicer) *AccountPoolMetrics {
	return &AccountPoolMetrics{
		accountSvc: accountSvc,
		Ready:      MetricCount{status: account.StatusReady, count: 0},
		NotReady:   MetricCount{status: account.StatusNotReady, count: 0},
		Leased:     MetricCount{status: account.StatusLeased, count: 0},
		Orphaned:   MetricCount{status: account.StatusOrphaned, count: 0}}
}
