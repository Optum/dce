package account

import (
	"github.com/Optum/dce/pkg/errors"
	validation "github.com/go-ozzo/ozzo-validation"
)

// Account - Handles importing and exporting Accounts and non-exported Properties
type Account struct {
	ID                  *string                `json:"id,omitempty" dynamodbav:"Id"`                                             // AWS Account ID
	Status              *Status                `json:"accountStatus,omitempty" dynamodbav:"AccountStatus,omitempty"`             // Status of the AWS Account
	LastModifiedOn      *int64                 `json:"lastModifiedOn,omitempty" dynamodbav:"LastModifiedOn"`                     // Last Modified Epoch Timestamp
	CreatedOn           *int64                 `json:"createdOn,omitempty"  dynamodbav:"CreatedOn,omitempty"`                    // Account CreatedOn
	AdminRoleArn        *string                `json:"adminRoleArn,omitempty"  dynamodbav:"AdminRoleArn"`                        // Assumed by the master account, to manage this user account
	PrincipalRoleArn    *string                `json:"principalRoleArn,omitempty"  dynamodbav:"PrincipalRoleArn,omitempty"`      // Assumed by principal users
	PrincipalPolicyHash *string                `json:"principalPolicyHash,omitempty" dynamodbav:"PrincipalPolicyHash,omitempty"` // The the hash of the policy version deployed
	Metadata            map[string]interface{} `json:"metadata,omitempty"  dynamodbav:"Metadata,omitempty"`                      // Any org specific metadata pertaining to the account
}

// Validate the account data
func (a *Account) Validate() error {
	err := validation.ValidateStruct(a,
		validation.Field(&a.AdminRoleArn, validateAdminRoleArn...),
		validation.Field(&a.ID, validateID...),
		validation.Field(&a.LastModifiedOn, validateInt64...),
		validation.Field(&a.Status, validateStatus...),
		validation.Field(&a.CreatedOn, validateInt64...),
		validation.Field(&a.PrincipalRoleArn, validatePrincipalRoleArn...),
		validation.Field(&a.PrincipalPolicyHash, validatePrincipalPolicyHash...),
	)
	if err != nil {
		return errors.NewValidation("account", err)
	}
	return nil
}

// Accounts is a list of type Account
type Accounts []Account

// Status is an account status type
type Status string

const (
	// StatusNone status
	StatusNone Status = "None"
	// StatusReady status
	StatusReady Status = "Ready"
	// StatusNotReady status
	StatusNotReady Status = "NotReady"
	// StatusLeased status
	StatusLeased Status = "Leased"
	// StatusOrphaned status
	StatusOrphaned Status = "Orphaned"
)

// String returns the string value of AccountStatus
func (c Status) String() string {
	return string(c)
}

// StringPtr returns a pointer to the string value of AccountStatus
func (c Status) StringPtr() *string {
	v := string(c)
	return &v
}

// StatusPtr returns a pointer to the string value of AccountStatus
func (c Status) StatusPtr() *Status {
	v := c
	return &v
}
