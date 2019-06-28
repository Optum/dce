package common

import (
	"github.com/aws/aws-sdk-go/service/codebuild"
)

// Builder interface requires a method to start a Build
type Builder interface {
	StartBuild(projectName *string,
		environmentVariables map[string]string) (string, error)
}

// CodeBuild implements the Builder interface using the AWS CodeBuild
// Service.
type CodeBuild struct {
	Client *codebuild.CodeBuild
}

// StartBuild method starts a CodeBuild Build based on the provided
// Project Name and Environment Variable to add or Overwrite
func (build *CodeBuild) StartBuild(projectName *string,
	environmentVariables map[string]string) (string, error) {
	// Construct the EnvironmentVariablesOverride for the input
	override := []*codebuild.EnvironmentVariable{}
	envType := "PLAINTEXT"
	for key, value := range environmentVariables {
		override = append(override, &codebuild.EnvironmentVariable{
			Name:  &key,
			Value: &value,
			Type:  &envType,
		})
	}

	// Construct the CodeBuild Input
	input := &codebuild.StartBuildInput{
		ProjectName:                  projectName,
		EnvironmentVariablesOverride: override,
	}

	// Start the CodeBuild
	output, err := build.Client.StartBuild(input)
	if err != nil {
		return "", err
	}

	// Return the Build's ID
	return *output.Build.Id, nil
}
