package model

// AccountData - Handles importing and exporting Accounts and non-exported Properties
type Account struct {
	ID                  string                 `json:"id" dynamodbav:"Id"`                                   // AWS Account ID
	Status              AccountStatus          `json:"accountStatus" dynamodbav:"AccountStatus"`             // Status of the AWS Account
	LastModifiedOn      int64                  `json:"lastModifiedOn" dynamodbav:"LastModifiedOn"`           // Last Modified Epoch Timestamp
	CreatedOn           int64                  `json:"createdOn"  dynamodbav:"CreatedOn"`                    // Account CreatedOn
	AdminRoleArn        string                 `json:"adminRoleArn"  dynamodbav:"AdminRoleArn"`              // Assumed by the master account, to manage this user account
	PrincipalRoleArn    string                 `json:"principalRoleArn"  dynamodbav:"PrincipalRoleArn"`      // Assumed by principal users
	PrincipalPolicyHash string                 `json:"principalPolicyHash" dynamodbav:"PrincipalPolicyHash"` // The the hash of the policy version deployed
	Metadata            map[string]interface{} `json:"metadata"  dynamodbav:"Metadata"`                      // Any org specific metadata pertaining to the account
}

type Accounts []Account

// Status is an account status type
type AccountStatus string

const (
	// None status
	None AccountStatus = "None"
	// Ready status
	Ready AccountStatus = "Ready"
	// NotReady status
	NotReady AccountStatus = "NotReady"
	// Leased status
	Leased AccountStatus = "Leased"
	// Orphaned status
	Orphaned AccountStatus = "Orphaned"
)
