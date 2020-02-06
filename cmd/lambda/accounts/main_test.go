package main

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	os.Setenv("ACCOUNT_CREATED_TOPIC_ARN", "mock-account-created-topic")
	os.Setenv("PRINCIPAL_ROLE_NAME", "DCEPrincipal")
	os.Setenv("RESET_SQS_URL", "mock.queue.url")
	os.Setenv("PRINCIPAL_MAX_SESSION_DURATION", "100")
	os.Setenv("PRINCIPAL_POLICY_NAME", "DCEPrincipalDefaultPolicy")
	os.Setenv("PRINCIPAL_IAM_DENY_TAGS", "DCE,CantTouchThis")
	os.Setenv("ACCOUNT_DELETED_TOPIC_ARN", "test:arn")
	os.Exit(m.Run())
}
