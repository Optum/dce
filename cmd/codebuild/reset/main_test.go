package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	commonMocks "github.com/Optum/dce/pkg/common/mocks"
	"github.com/Optum/dce/pkg/db"
	"github.com/Optum/dce/pkg/db/mocks"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestResetPipeline(t *testing.T) {
	t.Run("updateDBPostReset", func(t *testing.T) {

		t.Run("Should change account status from NotReady to Ready", func(t *testing.T) {
			dbSvc := &mocks.DBer{}
			snsSvc := &commonMocks.Notificationer{}
			defer dbSvc.AssertExpectations(t)

			// Should change the Account Status
			dbSvc.
				On("TransitionAccountStatus", "111", db.NotReady, db.Ready).
				Return(&db.Account{}, nil)

			snsSvc.On("PublishMessage",
				mock.MatchedBy(func(arn *string) bool {
					return *arn == "Topic"
				}),
				mock.MatchedBy(func(message *string) bool {
					// Parse the message JSON
					messageObj := unmarshal(t, *message)
					// `default` and `body` and JSON embedded within the message JSON
					msgDefault := unmarshal(t, messageObj["default"].(string))
					msgBody := unmarshal(t, messageObj["Body"].(string))

					assert.Equal(t, msgDefault, msgBody, "SNS default/Body should  match")

					// Check that we're sending the account object
					assert.Equal(t, "", msgBody["Id"])

					return true
				}), true,
			).Return(aws.String("mock message"), nil)
			defer snsSvc.AssertExpectations(t)

			err := updateDBPostReset(dbSvc, snsSvc, "111", "Topic")
			dbSvc.AssertNumberOfCalls(t, "TransitionLeaseStatus", 0)
			dbSvc.AssertNumberOfCalls(t, "TransitionAccountStatus", 1)
			require.Nil(t, err)
		})

		t.Run("Should not change account status of Leased accounts", func(t *testing.T) {
			dbSvc := &mocks.DBer{}
			snsSvc := &commonMocks.Notificationer{}
			defer dbSvc.AssertExpectations(t)

			// Mock Account status change, so it returns an error
			// saying that the Account was not in "NotReady" state.
			// We'll just ignore this error, because it means
			// we don't want to change status at all.
			dbSvc.
				On("TransitionAccountStatus", "111", db.NotReady, db.Ready).
				Return(nil, &db.StatusTransitionError{})

			dbSvc.
				On("GetAccount", "111").
				Return(&db.Account{}, nil)

			snsSvc.On("PublishMessage",
				mock.MatchedBy(func(arn *string) bool {
					return *arn == "Topic"
				}),
				mock.MatchedBy(func(message *string) bool {
					// Parse the message JSON
					messageObj := unmarshal(t, *message)
					// `default` and `body` and JSON embedded within the message JSON
					msgDefault := unmarshal(t, messageObj["default"].(string))
					msgBody := unmarshal(t, messageObj["Body"].(string))

					assert.Equal(t, msgDefault, msgBody, "SNS default/Body should  match")

					// Check that we're sending the account object
					assert.Equal(t, "", msgBody["Id"])

					return true
				}), true,
			).Return(aws.String("mock message"), nil)
			defer snsSvc.AssertExpectations(t)

			err := updateDBPostReset(dbSvc, snsSvc, "111", "Topic")
			dbSvc.AssertNumberOfCalls(t, "TransitionLeaseStatus", 0)
			dbSvc.AssertNumberOfCalls(t, "TransitionAccountStatus", 1)
			require.Nil(t, err)
		})

		t.Run("Should handle DB errors (TransitionAccountStatus)", func(t *testing.T) {
			snsSvc := &commonMocks.Notificationer{}
			dbSvc := &mocks.DBer{}
			defer dbSvc.AssertExpectations(t)

			// Mock Account status change, so it returns an error
			// saying that the Account was not in "NotReady" state.
			// We'll just ignore this error, because it means
			// we don't want to change status at all.
			dbSvc.
				On("TransitionAccountStatus", "111", db.NotReady, db.Ready).
				Return(nil, errors.New("test error"))

			err := updateDBPostReset(dbSvc, snsSvc, "111", "Topic")
			dbSvc.AssertNumberOfCalls(t, "TransitionLeaseStatus", 0)
			dbSvc.AssertNumberOfCalls(t, "TransitionAccountStatus", 1)
			require.Equal(t, errors.New("test error"), err)
		})
	})

	t.Run("testNukeConfigGeneration", func(t *testing.T) {

		var b bytes.Buffer
		_config = &serviceConfig{
			accountID:                  "ABC123",
			accountAdminRoleName:       "AdminRole",
			nukeRegions:                []string{"us-east-1", "us-west-1"},
			accountPrincipalRoleName:   "PrincipalRole",
			accountPrincipalPolicyName: "PrincipalPolicy",
			nukeTemplateDefault:        "default-nuke-config-template.yml",
			nukeTemplateBucket:         "STUB",
			nukeTemplateKey:            "STUB",
		}
		svc := service{}

		err := generateNukeConfig(&svc, &b)
		assert.NoError(t, err)

		got := b.String()
		want := "regions:\n  - \"global\"\n  # DCE Principals roles are currently locked down\n  # to only access these two regions\n  # This significantly reduces the run time of nuke.\n  - \"us-east-1\"\n  - \"us-west-1\"\n\naccount-blacklist:\n  - \"999999999999\" # Arbitrary production account id\n\nresource-types:\n  excludes:\n    - S3Object # Let the S3Bucket delete all Objects instead of individual objects (optimization)\n\naccounts:\n  \"ABC123\": # Child Account\n    filters:\n      IAMPolicy:\n        - type: \"contains\"\n          value: \"PrincipalPolicy\"\n      IAMRole:\n        - \"AdminRole\"\n        - \"PrincipalRole\"\n      IAMRolePolicy:\n        - type: \"contains\"\n          value: \"AdminRole\"\n        - type: \"contains\"\n          value: \"PrincipalRole\"\n        - type: \"contains\"\n          value: \"PrincipalPolicy\"\n      IAMRolePolicyAttachment:\n        # Do not remove the policy from the principal user role\n        - \"PrincipalRole -> PrincipalPolicy\"\n"
		assert.Equal(t, got, want, "Template subsitition works")
	})
}

func unmarshal(t *testing.T, jsonStr string) map[string]interface{} {
	var data map[string]interface{}
	err := json.Unmarshal([]byte(jsonStr), &data)
	require.Nil(t, err,
		fmt.Sprintf("Failed to unmarshal JSON: %s; %s", jsonStr, err),
	)

	return data
}
