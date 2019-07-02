package response

import (
	"github.com/Optum/Redbox/pkg/db"
)

// CreateAccountAssignmentResponse creates an AccountAssignment Response based
// on the provided RedboxAccountAssignment
func CreateAccountAssignmentResponse(assgn *db.RedboxAccountAssignment) *AccountAssignmentResponse {
	return &AccountAssignmentResponse{
		AccountID:        assgn.AccountID,
		UserID:           assgn.UserID,
		AssignmentStatus: assgn.AssignmentStatus,
		CreatedOn:        assgn.CreatedOn,
		LastModifiedOn:   assgn.LastModifiedOn,
	}
}

// AccountAssignmentResponse is the structured JSON Response for an Assignment
// to be returned for APIs
// {
// 	"accountId": "123",
// 	"userId": "user",
// 	"assignmentStatus": "Active",
// 	"createdOn": 56789,
// 	"lastModifiedOn": 56789
// }
type AccountAssignmentResponse struct {
	AccountID        string              `json:"accountId"`
	UserID           string              `json:"userId"`
	AssignmentStatus db.AssignmentStatus `json:"assignmentStatus"`
	CreatedOn        int64               `json:"createdOn"`
	LastModifiedOn   int64               `json:"lastModifiedOn"`
}
