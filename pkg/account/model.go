package account

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/Optum/dce/pkg/arn"
	"github.com/Optum/dce/pkg/errors"
	validation "github.com/go-ozzo/ozzo-validation"
)

var (
	// PrincipalRoleName default principal role Name
	PrincipalRoleName string
	// PrincipalPolicyName default principal policy Name
	PrincipalPolicyName string
	// ValidStatuses has the valid status options
	ValidStatuses []Status
)

func init() {
	var ok bool
	PrincipalRoleName, ok = os.LookupEnv("PRINCIPAL_ROLE_NAME")
	if !ok {
		PrincipalRoleName = "DCEPrincipal"
	}
	PrincipalPolicyName, ok = os.LookupEnv("PRINCIPAL_POLICY_NAME")
	if !ok {
		PrincipalPolicyName = "DCEPrincipalDefaultPolicy"
	}
	ValidStatuses = []Status{
		StatusNone,
		StatusLeased,
		StatusNotReady,
		StatusOrphaned,
		StatusReady,
	}
}

// Account - Handles importing and exporting Accounts and non-exported Properties
type Account struct {
	ID                  *string                `json:"id,omitempty" dynamodbav:"Id" schema:"id,omitempty"`                                                              // AWS Account ID
	Status              *Status                `json:"accountStatus,omitempty" dynamodbav:"AccountStatus,omitempty" schema:"status,omitempty"`                          // Status of the AWS Account
	LastModifiedOn      *int64                 `json:"lastModifiedOn,omitempty" dynamodbav:"LastModifiedOn" schema:"lastModifiedOn,omitempty"`                          // Last Modified Epoch Timestamp
	CreatedOn           *int64                 `json:"createdOn,omitempty"  dynamodbav:"CreatedOn,omitempty" schema:"createdOn,omitempty"`                              // Account CreatedOn
	AdminRoleArn        *arn.ARN               `json:"adminRoleArn,omitempty"  dynamodbav:"AdminRoleArn" schema:"adminRoleArn,omitempty"`                               // Assumed by the master account, to manage this user account
	PrincipalRoleArn    *arn.ARN               `json:"principalRoleArn,omitempty"  dynamodbav:"PrincipalRoleArn,omitempty" schema:"principalRoleArn,omitempty"`         // Assumed by principal users
	PrincipalPolicyHash *string                `json:"principalPolicyHash,omitempty" dynamodbav:"PrincipalPolicyHash,omitempty" schema:"principalPolicyHash,omitempty"` // The the hash of the policy version deployed
	Metadata            map[string]interface{} `json:"metadata,omitempty"  dynamodbav:"Metadata,omitempty" schema:"-"`                                                  // Any org specific metadata pertaining to the account
	Limit               *int64                 `json:"-" dynamodbav:"-" schema:"limit,omitempty"`
	NextID              *string                `json:"-" dynamodbav:"-" schema:"nextId,omitempty"`
	PrincipalRoleName   *string                `json:"-"  dynamodbav:"-" schema:"-"`
	PrincipalPolicyArn  *arn.ARN               `json:"-"  dynamodbav:"-" schema:"-"`
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

// UnmarshalJSON helps with custom unmarshalling needs
func (a *Account) UnmarshalJSON(data []byte) error {
	type Alias Account

	var alias Alias
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}

	a.ID = alias.ID
	a.Status = alias.Status
	a.LastModifiedOn = alias.LastModifiedOn
	a.CreatedOn = alias.CreatedOn
	a.PrincipalRoleArn = alias.PrincipalRoleArn
	a.AdminRoleArn = alias.AdminRoleArn
	a.Metadata = alias.Metadata
	a.PrincipalPolicyHash = alias.PrincipalPolicyHash

	if alias.ID != nil {
		principalPolicyArn := arn.New("aws", "iam", "", *alias.ID, fmt.Sprintf("policy/%s", PrincipalPolicyName))
		a.PrincipalPolicyArn = principalPolicyArn
	}

	if alias.PrincipalRoleArn != nil {
		roleName := strings.Split(alias.PrincipalRoleArn.Resource, "/")[1]
		a.PrincipalRoleName = &roleName
	}

	return nil
}

// NewAccount creates a new instance of account
func NewAccount(accountID string, adminRoleArn arn.ARN, metadata map[string]interface{}) (*Account, error) {

	// we don't set last modified on and created on.  That will get set when there is a save
	policyArn := arn.New("aws", "iam", "", accountID, fmt.Sprintf("policy/%s", PrincipalPolicyName))
	roleArn := arn.New("aws", "iam", "", accountID, fmt.Sprintf("role/%s", PrincipalRoleName))

	return &Account{
		ID:                 &accountID,
		AdminRoleArn:       &adminRoleArn,
		PrincipalRoleName:  &PrincipalRoleName,
		PrincipalRoleArn:   roleArn,
		PrincipalPolicyArn: policyArn,
		Metadata:           metadata,
		Status:             StatusNotReady.StatusPtr(),
	}, nil
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
