package response

import (
	"github.com/Optum/Redbox/pkg/db"
)

// AccountResponse is the serialized JSON Response for a RedboxAccount
// to be returned for APIs
// {
// 	"id": "123",
// 	"status": "Active",
// 	"lastModifiedOn": 56789,
//	"createOn": 12345,
//	"adminRoleArn": " arn:aws:iam::1234567890:role/adminRole
// }
//
// Converting from a db.RedboxAccount can be done via type casting:
//	dbAccount := db.RedboxAccount{...}
//	accountRes := response.AccountResponse(dbAccount)
type AccountResponse struct {
	ID                  string                 `json:"id"`
	AccountStatus       db.AccountStatus       `json:"accountStatus"`
	LastModifiedOn      int64                  `json:"lastModifiedOn"`
	CreatedOn           int64                  `json:"createdOn"`
	AdminRoleArn        string                 `json:"adminRoleArn"`        // Assumed by the Redbox master account, to manage this user account
	PrincipalRoleArn    string                 `json:"principalRoleArn"`    // Assumed by principal users of Redbox
	PrincipalPolicyHash string                 `json:"principalPolicyHash"` // The policy used by the PrincipalRoleArn
	Metadata            map[string]interface{} `json:"metadata"`
}
