package main

import (
	"github.com/Optum/Redbox/pkg/awsiface"
	"github.com/Optum/Redbox/pkg/budget"
	"github.com/Optum/Redbox/pkg/common"
	"github.com/Optum/Redbox/pkg/db"
	"github.com/aws/aws-sdk-go/service/costexplorer"
	"github.com/pkg/errors"
	"log"
	"time"
)

type calculateSpendInput struct {
	account    *db.RedboxAccount
	lease      *db.RedboxLease
	tokenSvc   common.TokenService
	budgetSvc  budget.Service
	awsSession awsiface.AwsSession
}

func calculateSpend(input *calculateSpendInput) (float64, error) {
	adminRoleArn := input.account.AdminRoleArn
	log.Printf("Assuming role %s for budget check", adminRoleArn)
	assumedSession, err := input.tokenSvc.NewSession(input.awsSession, adminRoleArn)
	if err != nil {
		return 0, errors.Wrapf(err, "Failed to assume role %s", adminRoleArn)
	}

	// Configure the CostExplorer SDK for the Service
	input.budgetSvc.SetCostExplorer(
		costexplorer.New(assumedSession),
	)

	// Budget period starts last time the lease was reset.
	// We can look at the `leaseStatusModifiedOn` to know
	// when the lease status changed from `ResetLock` --> `Active`
	budgetStartTime := time.Unix(input.lease.LeaseStatusModifiedOn, 0)
	// CostExplorer's `endTime` arg is exclusive_. So if we want
	// the budget to include today's spend, we need the end time to be tomorrow.
	budgetEndTime := time.Now().Add(time.Hour * 24)

	log.Printf("Calculating spend for lease %s @ %s for period %s to %s...",
		input.lease.PrincipalID, input.lease.AccountID,
		budgetStartTime.Format("2006-01-02"), budgetEndTime.Format("2006-01-02"),
	)

	// Query CostExplorer, and calculate total spend in the account
	spend, err := input.budgetSvc.CalculateTotalSpend(budgetStartTime, budgetEndTime)
	if err != nil {
		return 0, errors.Wrapf(err, "Failed to calculate spend for account %s", input.lease.AccountID)
	}

	log.Printf("Lease for %s @ %s has spent $%.2f of their $%.2f budget",
		input.lease.PrincipalID, input.lease.AccountID, spend, input.lease.BudgetAmount)

	return spend, nil
}
