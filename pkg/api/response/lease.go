package response

import (
	"github.com/Optum/Redbox/pkg/db"
)

// CreateLeaseResponse creates an Lease Response based
// on the provided RedboxLease
func CreateLeaseResponse(redboxLease *db.RedboxLease) *LeaseResponse {
	return &LeaseResponse{
		AccountID:                redboxLease.AccountID,
		PrincipalID:              redboxLease.PrincipalID,
		ID:                       redboxLease.ID,
		LeaseStatus:              redboxLease.LeaseStatus,
		CreatedOn:                redboxLease.CreatedOn,
		LastModifiedOn:           redboxLease.LastModifiedOn,
		BudgetAmount:             redboxLease.BudgetAmount,
		BudgetCurrency:           redboxLease.BudgetCurrency,
		BudgetNotificationEmails: redboxLease.BudgetNotificationEmails,
		LeaseStatusModifiedOn:    redboxLease.LeaseStatusModifiedOn,
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
	ID                       string         `json:"id"`
	LeaseStatus              db.LeaseStatus `json:"leaseStatus"`
	LeaseStatusReason        string         `json:"leaseStatusReturn"`
	CreatedOn                int64          `json:"createdOn"`
	LastModifiedOn           int64          `json:"lastModifiedOn"`
	BudgetAmount             float64        `json:"budgetAmount"`
	BudgetCurrency           string         `json:"budgetCurrency"`
	BudgetNotificationEmails []string       `json:"budgetNotificationEmails"`
	LeaseStatusModifiedOn    int64          `json:"leaseStatusModifiedOn"`
	RequestedLeaseStart      int64          `json:"requestedLeaseStart"`
	ActualLeaseStart         int64          `json:"actualLeaseStart"`
	RequestedLeaseEnd        int64          `json:"requestedLeaseEnd"`
	ActualLeaseEnd           int64          `json:"actualLeaseEnd"`
}
