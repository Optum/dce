package main

import (
	"encoding/json"
	"net/http"

	"github.com/Optum/dce/pkg/api/response"
	"github.com/Optum/dce/pkg/db"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/gorilla/mux"
)

// updateAccountRequest mirrors the db.Account object,
// except that it uses pointer types.
// This allows us to know whether the client omitted a field
// (ie. they don't want to update it)
// or, they explicitly set it to an empty value
type updateAccountRequest struct {
	ID                  *string                 `json:"id"`
	AccountStatus       *db.AccountStatus       `json:"accountStatus"`
	LastModifiedOn      *int64                  `json:"lastModifiedOn"`
	CreatedOn           *int64                  `json:"createdOn"`
	AdminRoleArn        *string                 `json:"adminRoleArn"`        // Assumed by the master account, to manage this user account
	PrincipalRoleArn    *string                 `json:"principalRoleArn"`    // Assumed by principal users
	PrincipalPolicyHash *string                 `json:"principalPolicyHash"` // The policy used by the PrincipalRoleArn
	Metadata            *map[string]interface{} `json:"metadata"`
}

func UpdateAccountByID(w http.ResponseWriter, r *http.Request) {
	accountID := mux.Vars(r)["accountId"]

	// Deserialize the request JSON as an request object
	var request *updateAccountRequest
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&request)
	if err != nil {
		ErrorHandler(w, err)
		return
	}
	request.ID = &accountID

	// If the request includes a new adminRoleArn,
	// validate that we can assume the ARN
	if request.AdminRoleArn != nil {
		_, err = TokenSvc.AssumeRole(&sts.AssumeRoleInput{
			RoleArn:         request.AdminRoleArn,
			RoleSessionName: aws.String("MasterAssumeRoleVerification"),
		})

		if err != nil {
			ErrorHandler(w, err)
			return
		}
	}

	// Figure out which DB fields we want to update
	// Note that not all fields are updatable
	// eg, we don't want clients to set their own timestamps,
	// 	or modify fields, like AccountStatus, which are  used for
	// 	internal bookkeeping.
	var fieldsToUpdate []string
	accountPartial := db.Account{
		ID: *request.ID,
	}
	if request.AdminRoleArn != nil {
		fieldsToUpdate = append(fieldsToUpdate, "AdminRoleArn")
		accountPartial.AdminRoleArn = *request.AdminRoleArn
	}
	if request.Metadata != nil {
		fieldsToUpdate = append(fieldsToUpdate, "Metadata")
		accountPartial.Metadata = *request.Metadata
	}
	if len(fieldsToUpdate) == 0 {
		ErrorHandler(w, err)
		return
	}

	// Update the DB record
	acct, err := Dao.UpdateAccount(accountPartial, fieldsToUpdate)
	if err != nil {
		// If the account doesn't exist, return a 404
		ErrorHandler(w, err)
		return
	}

	accountJSON, err := json.Marshal(response.AccountResponse(*acct))
	if err != nil {
		ErrorHandler(w, err)
		return
	}

	WriteAPIResponse(
		w,
		http.StatusOK,
		string(accountJSON),
	)
}
