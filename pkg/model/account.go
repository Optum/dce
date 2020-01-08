package model

// Account - Handles importing and exporting Accounts and non-exported Properties
type Account struct {
	ID                  *string                `json:"id,omitempty" dynamodbav:"Id"`                                             // AWS Account ID
	Status              *AccountStatus         `json:"accountStatus,omitempty" dynamodbav:"AccountStatus,omitempty"`             // Status of the AWS Account
	LastModifiedOn      *int64                 `json:"lastModifiedOn,omitempty" dynamodbav:"LastModifiedOn"`                     // Last Modified Epoch Timestamp
	CreatedOn           *int64                 `json:"createdOn,omitempty"  dynamodbav:"CreatedOn,omitempty"`                    // Account CreatedOn
	AdminRoleArn        *string                `json:"adminRoleArn,omitempty"  dynamodbav:"AdminRoleArn"`                        // Assumed by the master account, to manage this user account
	PrincipalRoleArn    *string                `json:"principalRoleArn,omitempty"  dynamodbav:"PrincipalRoleArn,omitempty"`      // Assumed by principal users
	PrincipalPolicyHash *string                `json:"principalPolicyHash,omitempty" dynamodbav:"PrincipalPolicyHash,omitempty"` // The the hash of the policy version deployed
	Metadata            map[string]interface{} `json:"metadata,omitempty"  dynamodbav:"Metadata,omitempty"`                      // Any org specific metadata pertaining to the account
}

// Accounts is multiple of Account
type Accounts []Account

// AccountStatus is an account status type
type AccountStatus string

const (
	// AccountStatusNone status
	AccountStatusNone AccountStatus = "None"
	// AccountStatusReady status
	AccountStatusReady AccountStatus = "Ready"
	// AccountStatusNotReady status
	AccountStatusNotReady AccountStatus = "NotReady"
	// AccountStatusLeased status
	AccountStatusLeased AccountStatus = "Leased"
	// AccountStatusOrphaned status
	AccountStatusOrphaned AccountStatus = "Orphaned"
)

// String returns the string value of AccountStatus
func (c AccountStatus) String() string {
	return string(c)
}

// StringPtr returns a pointer to the string value of AccountStatus
func (c AccountStatus) StringPtr() *string {
	v := string(c)
	return &v
}

// AccountStatusPtr returns a pointer to the string value of AccountStatus
func (c AccountStatus) AccountStatusPtr() *AccountStatus {
	v := c
	return &v
}
