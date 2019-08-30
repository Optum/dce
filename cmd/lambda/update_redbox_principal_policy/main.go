package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/Optum/Redbox/pkg/api/response"
	"github.com/Optum/Redbox/pkg/common"
	"github.com/Optum/Redbox/pkg/db"
	"github.com/Optum/Redbox/pkg/rolemanager"
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
	awsSession := newAWSSession()
	tokenSvc := common.STS{Client: sts.New(awsSession)}
	s3Svc := common.S3{
		Client:  s3.New(awsSession),
		Manager: s3manager.NewDownloader(awsSession),
	}

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
		account, err := dbSvc.GetAccount(lease.AccountID)
		if err != nil {
			log.Printf("Failed to get the redbox account %s: %s",
				lease.AccountID, err.Error())
			return err
		}

		accountRes := response.AccountResponse(*account)
		fmt.Printf("AccountID = %s \nRoleToAssume = %s\n", lease.AccountID, accountRes.AdminRoleArn)
		principalIAMDenyTags := strings.Split(common.RequireEnv("PRINCIPAL_IAM_DENY_TAGS"), ",")
		principalRoleName := common.RequireEnv("PRINCIPAL_ROLE_NAME")
		policyName := common.RequireEnv("PRINCIPAL_POLICY_NAME")
		principalPolicyArn, err := arn.Parse(fmt.Sprintf("arn:aws:iam::%s:policy/%s", lease.AccountID, policyName))
		if err != nil {
			log.Printf("Failed to parse ARN 'arn:aws:iam::%s:policy/%s': %s", lease.AccountID, policyName, err.Error())
			return err
		}

		policy, err := getPolicy(s3Svc, getPolicyInput{
			PrincipalPolicyArn:   principalPolicyArn.String(),
			PrincipalRoleArn:     fmt.Sprintf("arn:aws:iam::%s:role/%s", lease.AccountID, principalRoleName),
			PrincipalIAMDenyTags: principalIAMDenyTags,
			AdminRoleArn:         accountRes.AdminRoleArn,
		})

		// Assume role into the new Redbox account
		accountSession, err := tokenSvc.NewSession(awsSession, accountRes.AdminRoleArn)
		if err != nil {
			log.Printf("Failed to assume role '%s': %s", accountRes.AdminRoleArn, err.Error())
			return err
		}
		iamSvc := iam.New(accountSession)

		// Create the Role + Policy
		roleManager := &rolemanager.IAMPolicyManager{}
		roleManager.SetIAMClient(iamSvc)
		return roleManager.MergePolicy(&rolemanager.MergePolicyInput{
			PolicyName:        policyName,
			PolicyArn:         principalPolicyArn,
			PolicyDocument:    policy,
			PolicyDescription: "", // Policy should already exist so this will be ignored
		})
	}
	return nil
}

type getPolicyInput struct {
	PrincipalPolicyArn   string
	PrincipalRoleArn     string
	PrincipalIAMDenyTags []string
	AdminRoleArn         string
}

func getPolicy(storage common.S3, input getPolicyInput) (string, error) {
	bucket := common.RequireEnv("ARTIFACTS_BUCKET")
	key := common.RequireEnv("PRINCIPAL_POLICY_S3_KEY")
	policy, err := storage.GetTemplateObject(bucket, key, input)
	return policy, err
}

func newAWSSession() *session.Session {
	awsSession, err := session.NewSession()
	if err != nil {
		errorMessage := fmt.Sprintf("Failed to create AWS session: %s", err)
		log.Fatal(errorMessage)
	}
	return awsSession
}
