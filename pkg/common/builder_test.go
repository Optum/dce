package common

import (
	"errors"
	"github.com/Optum/Dcs/pkg/common/commoniface/mocks"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/codebuild"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestBuilder(t *testing.T) {
	t.Run("should start a build", func(t *testing.T) {
		mockCodeBuild := &mocks.CodeBuildAPI{}
		builder := CodeBuild{
			Client: mockCodeBuild,
		}

		// Mock StartBuild call
		mockCodeBuild.On("StartBuild",
			&codebuild.StartBuildInput{
				ProjectName:                  aws.String("test-project"),
				EnvironmentVariablesOverride: []*codebuild.EnvironmentVariable{},
			},
		).Return(StartBuildOutput("build-1"), nil)

		res, err := builder.StartBuild(aws.String("test-project"), map[string]string{})
		require.Nil(t, err)
		require.Equal(t, res, "build-1")

		mockCodeBuild.AssertExpectations(t)
	})

	t.Run("should return build errors", func(t *testing.T) {
		mockCodeBuild := &mocks.CodeBuildAPI{}
		builder := CodeBuild{
			Client: mockCodeBuild,
		}

		// Mock StartBuild call
		mockCodeBuild.On("StartBuild",
			&codebuild.StartBuildInput{
				ProjectName:                  aws.String("test-project"),
				EnvironmentVariablesOverride: []*codebuild.EnvironmentVariable{},
			},
		).Return(nil, errors.New("mock error"))

		_, err := builder.StartBuild(aws.String("test-project"), map[string]string{})
		require.NotNil(t, err)
		require.Equal(t, errors.New("mock error"), err)
	})

	t.Run("should pass env vars to a build", func(t *testing.T) {
		mockCodeBuild := &mocks.CodeBuildAPI{}
		builder := CodeBuild{
			Client: mockCodeBuild,
		}

		// Mock StartBuild call
		mockCodeBuild.On("StartBuild",
			mock.MatchedBy(func(input *codebuild.StartBuildInput) bool {
				assert.Equal(t, "test-project", *input.ProjectName)
				assert.Len(t, input.EnvironmentVariablesOverride, 2)

				// Order of env vars is not guaranteed, need to handle in either order
				var fooVar, barVar *codebuild.EnvironmentVariable
				if *input.EnvironmentVariablesOverride[0].Name == "FOO" {
					fooVar = input.EnvironmentVariablesOverride[0]
					barVar = input.EnvironmentVariablesOverride[1]
				} else {
					fooVar = input.EnvironmentVariablesOverride[1]
					barVar = input.EnvironmentVariablesOverride[0]
				}

				assert.Equal(t, &codebuild.EnvironmentVariable{
					Name:  aws.String("FOO"),
					Value: aws.String("bar"),
					Type:  aws.String("PLAINTEXT"),
				}, fooVar)
				assert.Equal(t, &codebuild.EnvironmentVariable{
					Name:  aws.String("FAZ"),
					Value: aws.String("baz"),
					Type:  aws.String("PLAINTEXT"),
				}, barVar)

				return true
			}),
		).Return(StartBuildOutput("build-1"), nil)

		res, err := builder.StartBuild(aws.String("test-project"), map[string]string{
			"FOO": "bar",
			"FAZ": "baz",
		})
		require.Nil(t, err)
		require.Equal(t, res, "build-1")

		mockCodeBuild.AssertExpectations(t)
	})
}

func StartBuildOutput(buildID string) *codebuild.StartBuildOutput {
	return &codebuild.StartBuildOutput{
		Build: &codebuild.Build{
			Id: aws.String(buildID),
		},
	}
}
