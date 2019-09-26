package response

import (
	"github.com/Optum/Dce/pkg/db"
)

// CreateLeaseResponse creates an Lease Response based
// on the provided DceLease
func CreateLeaseResponse(dceLease *db.DceLease) *LeaseResponse {
	return &LeaseResponse{
		AccountID:                dceLease.AccountID,
		PrincipalID:              dceLease.PrincipalID,
		LeaseStatus:              dceLease.LeaseStatus,
		CreatedOn:                dceLease.CreatedOn,
		LastModifiedOn:           dceLease.LastModifiedOn,
		BudgetAmount:             dceLease.BudgetAmount,
		BudgetCurrency:           dceLease.BudgetCurrency,
		BudgetNotificationEmails: dceLease.BudgetNotificationEmails,
		LeaseStatusModifiedOn:    dceLease.LeaseStatusModifiedOn,
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
