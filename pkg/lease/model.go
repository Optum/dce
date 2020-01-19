package lease

// Lease is a type corresponding to a Lease
// table record
type Lease struct {
	AccountID                *string                `json:"accountId,omitempty" dynamodbav:"AccountId"`                                         // AWS Account ID
	PrincipalID              *string                `json:"principalId,omitempty" dynamodbav:"PrincipalId"`                                     // Azure User Principal ID
	ID                       *string                `json:"id,omitempty" dynamodbav:"Id,omitempty"`                                             // Lease ID
	Status                   *Status                `json:"leaseStatus,omitempty" dynamodbav:"LeaseStatus,omitempty"`                           // Status of the Lease
	StatusReason             *StatusReason          `json:"leaseStatusReason,omitempty" dynamodbav:"LeaseStatusReason,omitempty"`               // Reason for the status of the lease
	CreatedOn                *int64                 `json:"createdOn,omitempty" dynamodbav:"CreatedOn,omitempty"`                               // Created Epoch Timestamp
	LastModifiedOn           *int64                 `json:"lastModifiedOn,omitempty" dynamodbav:"LastModifiedOn,omitempty"`                     // Last Modified Epoch Timestamp
	BudgetAmount             *float64               `json:"budgetAmount,omitempty" dynamodbav:"BudgetAmount,omitempty"`                         // Budget Amount allocated for this lease
	BudgetCurrency           *string                `json:"budgetCurrency,omitempty" dynamodbav:"BudgetCurrency,omitempty"`                     // Budget currency
	BudgetNotificationEmails *[]string              `json:"budgetNotificationEmails,omitempty" dynamodbav:"BudgetNotificationEmails,omitempty"` // Budget notification emails
	StatusModifiedOn         *int64                 `json:"leaseStatusModifiedOn,omitempty" dynamodbav:"LeaseStatusModifiedOn,omitempty"`       // Last Modified Epoch Timestamp
	ExpiresOn                *int64                 `json:"expiresOn,omitempty" dynamodbav:"ExpiresOn,omitempty"`                               // Lease expiration time as Epoch
	Metadata                 map[string]interface{} `json:"metadata,omitempty" dynamodbav:"Metadata,omitempty"`                                 // Arbitrary key-value metadata to store with lease object
}

// Leases is a list of type Account
type Leases []Lease
