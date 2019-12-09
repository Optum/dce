package main

import (
	"encoding/json"
	"net/http"

	"github.com/Optum/dce/pkg/account"
	"github.com/Optum/dce/pkg/data"
)

// CreateAccount - Function to validate the account request to add into the pool and
// publish the account creation to its respective client
func CreateAccount(w http.ResponseWriter, r *http.Request) {

	// Marshal the request JSON into a CreateRequest object
	request := &CreateRequest{}
	var err error
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&request)

	if err != nil {
		ErrorHandler(w, err)
		return
	}

	// Set default metadata={}
	if request.Metadata == nil {
		request.Metadata = map[string]interface{}{}
	}

	// Validate the request body
	//isValid, validationRes := request.Validate()
	//if !isValid {
	//	err = fmt.Errorf("%s: %w", *validationRes, errors.ErrValidation)
	//	ErrorHandler(w, err)
	//	return
	//}

	var dao *data.Account
	if err := Services.Config.GetService(dao); err != nil {
		ErrorHandler(w, err)
		return
	}

	a, err := account.CreateAccount(request.ID, request.AdminRoleArn, request.Metadata, dao)
	if err != nil {
		ErrorHandler(w, err)
		return
	}
	WriteAPIResponse(w, http.StatusCreated, a)
}

type CreateRequest struct {
	ID           string                 `json:"id"`
	AdminRoleArn string                 `json:"adminRoleArn"`
	Metadata     map[string]interface{} `json:"metadata"`
}

//
//// Validate - Checks if the Account Request has the provided id and adminRoleArn
//// fields
//func (req *CreateRequest) Validate() (bool, *string) {
//	isValid := true
//	var validationErrors []error
//	if req.ID == "" {
//		isValid = false
//		validationErrors = append(validationErrors, fmt.Errorf("missing required field \"id\": %w", errors.ErrValidation))
//
//	}
//	if req.AdminRoleArn == "" {
//		isValid = false
//		validationErrors = append(validationErrors, fmt.Errorf("missing required field \"adminRoleArn\": %w", errors.ErrValidation))
//	}
//
//	if !isValid {
//		errMsgs := []string{}
//		for _, verr := range validationErrors {
//			errMsgs = append(errMsgs, verr.Error())
//		}
//		msg := strings.Join(errMsgs, "; ")
//		return false, &msg
//	}
//
//	return true, nil
//}
//
//func createPrincipalRole(childAccount db.Account, masterAccountID string) (*rolemanager.CreateRoleWithPolicyOutput, string, error) {
//	// Create an assume role policy,
//	// to let principals from the same account assume the role.
//	//
//	// Consumers of open source DCE may modify and customize
//	// this as need (eg. to integrate with SSO/SAML)
//	// by responding to the "account-created" SNS topic
//	assumeRolePolicy := strings.TrimSpace(fmt.Sprintf(`
//		{
//			"Version": "2012-10-17",
//			"Statement": [
//				{
//					"Effect": "Allow",
//					"Principal": {
//						"AWS": "arn:aws:iam::%s:root"
//					},
//					"Action": "sts:AssumeRole",
//					"Condition": {}
//				}
//			]
//		}
//	`, masterAccountID))
//
//	// Render the default policy for the principal
//
//	policy, policyHash, err := StorageSvc.GetTemplateObject(artifactsBucket, principalPolicyS3Key,
//		principalPolicyInput{
//			PrincipalPolicyArn:   fmt.Sprintf("arn:aws:iam::%s:policy/%s", childAccount.ID, policyName),
//			PrincipalRoleArn:     fmt.Sprintf("arn:aws:iam::%s:role/%s", childAccount.ID, principalRoleName),
//			PrincipalIAMDenyTags: principalIAMDenyTags,
//			AdminRoleArn:         childAccount.AdminRoleArn,
//			Regions:              allowedRegions,
//		})
//	if err != nil {
//		return nil, "", err
//	}
//
//	// Assume role into the new account
//	accountSession, err := TokenSvc.NewSession(AWSSession, childAccount.AdminRoleArn)
//	if err != nil {
//		return nil, "", err
//	}
//	iamClient := iam.New(accountSession)
//
//	// Create the Role + Policy
//	RoleManager.SetIAMClient(iamClient)
//	createRoleOutput := &rolemanager.CreateRoleWithPolicyOutput{}
//	createRoleOutput, err = RoleManager.CreateRoleWithPolicy(&rolemanager.CreateRoleWithPolicyInput{
//		RoleName:                 principalRoleName,
//		RoleDescription:          "Role to be assumed by principal users of DCE",
//		AssumeRolePolicyDocument: assumeRolePolicy,
//		MaxSessionDuration:       principalMaxSessionDuration,
//		PolicyName:               policyName,
//		PolicyDocument:           policy,
//		PolicyDescription:        "Policy for principal users of DCE",
//		Tags: append(tags,
//			&iam.Tag{Key: aws.String("Name"), Value: aws.String("DCEPrincipal")},
//		),
//		IgnoreAlreadyExistsErrors: true,
//	})
//	return createRoleOutput, policyHash, err
//}
//
//type principalPolicyInput struct {
//	PrincipalPolicyArn   string
//	PrincipalRoleArn     string
//	PrincipalIAMDenyTags []string
//	AdminRoleArn         string
//	Regions              []string
//}
//
