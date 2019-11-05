package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/Optum/dce/pkg/api/response"
	"github.com/Optum/dce/pkg/awsiface"
	"github.com/Optum/dce/pkg/common"
	"github.com/Optum/dce/pkg/db"
	"github.com/Optum/dce/pkg/rolemanager"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/sts"
)

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, snsEvent events.SNSEvent) error {
	dbSvc, err := db.NewFromEnv()
	awsSession := session.Must(session.NewSession())
	tokenSvc := common.STS{Client: sts.New(awsSession)}
	s3Svc := common.S3{
		Client:  s3.New(awsSession),
		Manager: s3manager.NewDownloader(awsSession),
	}
	roleManagerSvc := &rolemanager.IAMPolicyManager{}

	if err != nil {
		log.Printf("Unable to setup DB Service: %s", err.Error())
		return err
	}

	for _, record := range snsEvent.Records {
		snsRecord := record.SNS

		var lease response.LeaseResponse
		err := json.Unmarshal([]byte(snsRecord.Message), &lease)
		if err != nil {
			log.Printf("Failed to read SNS message %s: %s", snsRecord.Message, err.Error())
			return err
		}

		err = processRecord(processRecordInput{
			AccountID:            lease.AccountID,
			DbSvc:                dbSvc,
			StoragerSvc:          s3Svc,
			TokenSvc:             tokenSvc,
			AwsSession:           awsSession,
			RoleManager:          roleManagerSvc,
			PrincipalRoleName:    common.RequireEnv("PRINCIPAL_ROLE_NAME"),
			PrincipalPolicyName:  common.RequireEnv("PRINCIPAL_POLICY_NAME"),
			PrincipalIAMDenyTags: strings.Split(common.RequireEnv("PRINCIPAL_IAM_DENY_TAGS"), ","),
			PolicyBucket:         common.RequireEnv("ARTIFACTS_BUCKET"),
			PolicyBucketKey:      common.RequireEnv("PRINCIPAL_POLICY_S3_KEY"),
		})
	}
	return nil
}

type processRecordInput struct {
	AccountID            string
	DbSvc                db.DBer
	StoragerSvc          common.Storager
	TokenSvc             common.TokenService
	AwsSession           awsiface.AwsSession
	RoleManager          rolemanager.PolicyManager
	PrincipalRoleName    string
	PrincipalPolicyName  string
	PrincipalIAMDenyTags []string
	PolicyBucket         string
	PolicyBucketKey      string
}

func processRecord(input processRecordInput) error {

	account, err := input.DbSvc.GetAccount(input.AccountID)
	if err != nil {
		log.Printf("Failed to get account %s: %s",
			input.AccountID, err.Error())
		return err
	}

	accountRes := response.AccountResponse(*account)

	principalPolicyArn, err := arn.Parse(fmt.Sprintf("arn:aws:iam::%s:policy/%s", input.AccountID, input.PrincipalPolicyName))
	if err != nil {
		log.Printf("Failed to parse ARN 'arn:aws:iam::%s:policy/%s': %s", input.AccountID, input.PrincipalPolicyName, err.Error())
		return err
	}

	policy, policyHash, err := input.StoragerSvc.GetTemplateObject(input.PolicyBucket, input.PolicyBucketKey, getPolicyInput{
		PrincipalPolicyArn:   principalPolicyArn.String(),
		PrincipalRoleArn:     fmt.Sprintf("arn:aws:iam::%s:role/%s", input.AccountID, input.PrincipalRoleName),
		PrincipalIAMDenyTags: input.PrincipalIAMDenyTags,
		AdminRoleArn:         accountRes.AdminRoleArn,
	})

	if policyHash == accountRes.PrincipalPolicyHash {
		log.Printf("Policy already matches.  Not updating '%s'", principalPolicyArn.String())
		return nil
	}
	// Assume role into the new child account
	accountSession, err := input.TokenSvc.NewSession(input.AwsSession, accountRes.AdminRoleArn)
	if err != nil {
		log.Printf("Failed to assume role '%s': %s", accountRes.AdminRoleArn, err.Error())
		return err
	}
	iamSvc := iam.New(accountSession)

	// Update the Policy
	input.RoleManager.SetIAMClient(iamSvc)
	log.Printf("Update policy '%s' to hash '%s' from '%s'.", principalPolicyArn.String(), accountRes.PrincipalPolicyHash, policyHash)
	err = input.RoleManager.MergePolicy(&rolemanager.MergePolicyInput{
		PolicyName:     input.PrincipalPolicyName,
		PolicyArn:      principalPolicyArn,
		PolicyDocument: policy,
	})
	if err != nil {
		log.Printf("Failed updating the policy '%s': %s", principalPolicyArn.String(), err)
		return err
	}

	log.Printf("Update account '%s' resource record.  Policy Hash from '%s' to '%s'", input.AccountID, accountRes.PrincipalPolicyHash, policyHash)
	_, err = input.DbSvc.UpdateAccountPrincipalPolicyHash(input.AccountID, accountRes.PrincipalPolicyHash, policyHash)
	if err != nil {
		log.Printf("Failed to update account '%s' resource record.  Policy Hash from '%s' to '%s': %s",
			input.AccountID, accountRes.PrincipalPolicyHash, policyHash, err)
	}
	return err
}

type getPolicyInput struct {
	PrincipalPolicyArn   string
	PrincipalRoleArn     string
	PrincipalIAMDenyTags []string
	AdminRoleArn         string
}
