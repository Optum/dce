package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/Optum/dce/pkg/api/response"
	"github.com/Optum/dce/pkg/db"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/aws/aws-sdk-go/service/sts/stsiface"
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
		response.WriteAPIErrorResponse(w, http.StatusBadRequest, "ClientError", "invalid request parameters")
		return
	}
	request.ID = &accountID

	var tokenSvc stsiface.STSAPI
	if err := services.Config.GetService(&tokenSvc); err != nil {
		response.WriteServerErrorWithResponse(w, "Could not create token service")
		return
	}

	// If the request includes a new adminRoleArn,
	// validate that we can assume the ARN
	if request.AdminRoleArn != nil {
		_, err = tokenSvc.AssumeRole(&sts.AssumeRoleInput{
			RoleArn:         request.AdminRoleArn,
			RoleSessionName: aws.String("MasterAssumeRoleVerification"),
		})

		if err != nil {
			response.WriteRequestValidationError(
				w,
				fmt.Sprintf("Unable to update account %s: "+
					"admin role is not assumable by the master account",
					accountID),
			)
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
		response.WriteRequestValidationError(
			w,
			fmt.Sprintf("Unable to update account %s: "+
				"no updatable fields provided",
				accountID),
		)
		return
	}

	var dao db.DBer
	if err := services.Config.GetService(&dao); err != nil {
		response.WriteServerErrorWithResponse(w, "Could not create data service")
		return
	}

	// Update the DB record
	acct, err := dao.UpdateAccount(accountPartial, fieldsToUpdate)
	if err != nil {
		// If the account doesn't exist, return a 404
		if _, ok := err.(*db.NotFoundError); ok {
			response.WriteNotFoundError(w)
			return
		}
		// Other DB errors return a 500
		log.Printf("ERROR: Failed to update account %s: %s", *request.ID, err)
		response.WriteServerErrorWithResponse(w, "Internal Server Error")
		return
	}

	accountJSON, err := json.Marshal(response.AccountResponse(*acct))
	if err != nil {
		log.Printf("ERROR: Failed to marshal account response for %s: %s", *request.ID, err)
		response.WriteServerErrorWithResponse(w, "Internal server error")
		return
	}

	response.WriteAPIResponse(
		w,
		http.StatusOK,
		string(accountJSON),
	)
}
