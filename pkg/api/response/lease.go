package response

import (
	"github.com/Optum/dce/pkg/db"
)

// CreateLeaseResponse creates an Lease Response based
// on the provided Lease
func CreateLeaseResponse(lease *db.Lease) *LeaseResponse {
	response := LeaseResponse(*lease)
	return &response
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
	AccountID                string                 `json:"accountId"`
	PrincipalID              string                 `json:"principalId"`
	ID                       string                 `json:"id"`
	LeaseStatus              db.LeaseStatus         `json:"leaseStatus"`
	LeaseStatusReason        db.LeaseStatusReason   `json:"leaseStatusReason"`
	CreatedOn                int64                  `json:"createdOn"`
	LastModifiedOn           int64                  `json:"lastModifiedOn"`
	BudgetAmount             float64                `json:"budgetAmount"`
	BudgetCurrency           string                 `json:"budgetCurrency"`
	BudgetNotificationEmails []string               `json:"budgetNotificationEmails"`
	LeaseStatusModifiedOn    int64                  `json:"leaseStatusModifiedOn"`
	ExpiresOn                int64                  `json:"expiresOn"`
	Metadata                 map[string]interface{} `json:"metadata"`
}
