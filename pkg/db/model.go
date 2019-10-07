package db

// RedboxAccount is a type corresponding to a RedboxAccount table record
type RedboxAccount struct {
	ID                  string                 `json:"Id"`             // AWS Account ID
	AccountStatus       AccountStatus          `json:"AccountStatus"`  // Status of the AWS Account
	LastModifiedOn      int64                  `json:"LastModifiedOn"` // Last Modified Epoch Timestamp
	CreatedOn           int64                  `json:"CreatedOn"`
	AdminRoleArn        string                 `json:"AdminRoleArn"`        // Assumed by the Redbox master account, to manage this user account
	PrincipalRoleArn    string                 `json:"PrincipalRoleArn"`    // Assumed by principal users of Redbox
	PrincipalPolicyHash string                 `json:"PrincipalPolicyHash"` // The the hash of the policy version deployed
	Metadata            map[string]interface{} `json:"Metadata"`            // Any org specific metadata pertaining to the account
}

// RedboxLease is a type corresponding to a RedboxLease
// table record
type RedboxLease struct {
	AccountID                string      `json:"AccountId"`                // AWS Account ID
	PrincipalID              string      `json:"PrincipalId"`              // Azure User Principal ID
	ID                       string      `json:"Id"`                       // Lease ID
	LeaseStatus              LeaseStatus `json:"LeaseStatus"`              // Status of the Lease
	LeaseStatusReason        string      `json:"LeaseStatusReason"`        // Reason for the status of the lease
	CreatedOn                int64       `json:"CreatedOn"`                // Created Epoch Timestamp
	LastModifiedOn           int64       `json:"LastModifiedOn"`           // Last Modified Epoch Timestamp
	BudgetAmount             float64     `json:"BudgetAmount"`             // Budget Amount allocated for this lease
	BudgetCurrency           string      `json:"BudgetCurrency"`           // Budget currency
	BudgetNotificationEmails []string    `json:"BudgetNotificationEmails"` // Budget notification emails
	LeaseStatusModifiedOn    int64       `json:"LeaseStatusModifiedOn"`    // Last Modified Epoch Timestamp
	RequestedLeaseStart      int64       `json:"RequestedLeaseStart"`      // Requested lease start time as an Epoch Timestamp
	ActualLeaseStart         int64       `json:"ActualLeaseStart"`         // Actual lease start time as an Epoch Timestamp
	RequestedLeaseEnd        int64       `json:"RequestedLeaseEnd"`        // Requested lease end time as an Epoch Timestamp
	ActualLeaseEnd           int64       `json:"ActualLeaseEnd"`           // Actual lease end time as an Epoch Timestamp
}

// Timestamp is a timestamp type for epoch format
type Timestamp int64

// Timestamped contains timestamp types
type Timestamped struct {
	CreatedOn      Timestamp
	LastModifiedOn Timestamp
}

// AccountStatus is a Redbox account status type
type AccountStatus string

const (
	// Ready status
	Ready AccountStatus = "Ready"
	// NotReady status
	NotReady AccountStatus = "NotReady"
	// Leased status
	Leased AccountStatus = "Leased"
)

// LeaseStatus is a Redbox account lease status type
type LeaseStatus string

const (
	// Active status
	Active LeaseStatus = "Active"
	// Inactive status
	Inactive LeaseStatus = "Inactive"
)
