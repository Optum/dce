package accountmanager

import (
	"fmt"
	"testing"
	"time"

	"github.com/Optum/dce/pkg/awsiface/mocks"
	"github.com/Optum/dce/pkg/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAssumeRole(t *testing.T) {

	tests := []struct {
		name         string
		manager      AccountManager
		adminRoleArn string
		assumeResp   *sts.AssumeRoleOutput
		assumeErr    error
		expErr       error
	}{
		{
			name:         "normal",
			adminRoleArn: "arn:aws:iam::123456789012:role/test",
			assumeResp: &sts.AssumeRoleOutput{
				Credentials: &sts.Credentials{
					Expiration:      aws.Time(time.Now()),
					AccessKeyId:     aws.String("AKID"),
					SecretAccessKey: aws.String("SECRET"),
					SessionToken:    aws.String("SESSION"),
				},
			},
			assumeErr: nil,
		},
		{
			name:         "assume failure",
			adminRoleArn: "arn:aws:iam::123456789012:role/test",
			assumeResp:   nil,
			assumeErr:    fmt.Errorf("failure"),
			expErr:       errors.NewValidation("admin role arn", fmt.Errorf("failure")),
		},
		{
			name:         "parse admin role arn",
			adminRoleArn: "arn:aws:iam:role/test",
			assumeResp:   nil,
			assumeErr:    nil,
			expErr:       errors.NewValidation("admin role arn", fmt.Errorf("arn: not enough sections")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			manager, err := New(NewInput{"testRole", "testPolicy"})
			assert.Nil(t, err)

			stsMock := &mocks.STSAPI{}
			stsMock.On("AssumeRole", mock.MatchedBy(func(input *sts.AssumeRoleInput) bool {
				return *input.DurationSeconds == 900 && *input.RoleArn == tt.adminRoleArn
			})).Return(tt.assumeResp, tt.assumeErr)
			manager.sts = stsMock

			err = manager.Setup(tt.adminRoleArn)

			assert.True(t, errors.Is(err, tt.expErr))
		})
	}
}
