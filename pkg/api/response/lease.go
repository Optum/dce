package response

import (
	"github.com/Optum/Dcs/pkg/db"
)

// CreateLeaseResponse creates an Lease Response based
// on the provided DcsLease
func CreateLeaseResponse(dcsLease *db.DcsLease) *LeaseResponse {
	return &LeaseResponse{
		AccountID:                dcsLease.AccountID,
		PrincipalID:              dcsLease.PrincipalID,
		LeaseStatus:              dcsLease.LeaseStatus,
		CreatedOn:                dcsLease.CreatedOn,
		LastModifiedOn:           dcsLease.LastModifiedOn,
		BudgetAmount:             dcsLease.BudgetAmount,
		BudgetCurrency:           dcsLease.BudgetCurrency,
		BudgetNotificationEmails: dcsLease.BudgetNotificationEmails,
		LeaseStatusModifiedOn:    dcsLease.LeaseStatusModifiedOn,
	}
}

// LeaseResponse is the structured JSON Response for an Lease
// to be returned for APIs
// {
// 	"accountId": "123",
// 	"principalId": "user",
// 	"leaseStatus": "Active",
// 	"createdOn": 56789,
// 	"lastModifiedOn": 56789,
// 	"budgetAmount": 300,
// 	"BudgetCurrency": "USD",
// 	"BudgetNotificationEmails": ["usermsid@test.com", "managersmsid@test.com"]
// }
type LeaseResponse struct {
	AccountID                string         `json:"accountId"`
	PrincipalID              string         `json:"principalId"`
	LeaseStatus              db.LeaseStatus `json:"leaseStatus"`
	CreatedOn                int64          `json:"createdOn"`
	LastModifiedOn           int64          `json:"lastModifiedOn"`
	BudgetAmount             float64        `json:"budgetAmount"`
	BudgetCurrency           string         `json:"budgetCurrency"`
	BudgetNotificationEmails []string       `json:"budgetNotificationEmails"`
	LeaseStatusModifiedOn    int64          `json:"leaseStatusModifiedOn"`
}
