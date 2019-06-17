package db

// RedboxAccount is a type corresponding to a RedboxAccount table record
type RedboxAccount struct {
	ID             string        `json:"Id"`             // AWS Account ID
	GroupID        string        `json:"GroupId"`        // Azure AD Group ID associated with the Account
	AccountStatus  AccountStatus `json:"AccountStatus"`  // Status of the AWS Account
	LastModifiedOn int64         `json:"LastModifiedOn"` // Last Modified Epoch Timestamp
}

// RedboxAccountAssignment is a type corresponding to a RedboxAccountAssignment
// table record
type RedboxAccountAssignment struct {
	AccountID        string           `json:"AccountId"`        // AWS Account ID
	UserID           string           `json:"UserId"`           // Azure User ID
	AssignmentStatus AssignmentStatus `json:"AssignmentStatus"` // Status of the Assignment
	CreatedOn        int64            `json:"CreatedOn"`        // Created Epoch Timestamp
	LastModifiedOn   int64            `json:"LastModifiedOn"`   // Last Modified Epoch Timestamp
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
	// Assigned status
	Assigned AccountStatus = "Assigned"
)

// AssignmentStatus is a Redbox account assignment status type
type AssignmentStatus string

const (
	// Active status
	Active AssignmentStatus = "Active"
	// Decommissioned status
	Decommissioned AssignmentStatus = "Decommissioned"
	// FinanceLock status
	FinanceLock AssignmentStatus = "FinanceLock"
	// ResetLock status
	ResetLock AssignmentStatus = "ResetLock"
	// ResetFinanceLock status
	// Same as ResetLock but the account's status was FinanceLock beforehand
	// and should be FinanceLock after a Reset has been applied
	ResetFinanceLock AssignmentStatus = "ResetFinanceLock"
)
