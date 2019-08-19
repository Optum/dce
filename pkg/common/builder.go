package common

import (
	"github.com/aws/aws-sdk-go/service/codebuild"
	"github.com/aws/aws-sdk-go/service/codebuild/codebuildiface"
)

// Builder interface requires a method to start a Build
type Builder interface {
	StartBuild(projectName *string,
		environmentVariables map[string]string) (string, error)
}

// CodeBuild implements the Builder interface using the AWS CodeBuild
// Service.
type CodeBuild struct {
	Client codebuildiface.CodeBuildAPI
}

// StartBuild method starts a CodeBuild Build based on the provided
// Project Name and Environment Variable to add or Overwrite
func (build *CodeBuild) StartBuild(projectName *string,
	environmentVariables map[string]string) (string, error) {
	// Construct the EnvironmentVariablesOverride for the input
	envVars := []*codebuild.EnvironmentVariable{}
	for key, value := range environmentVariables {
		envType := "PLAINTEXT"
		overrideKey := key // Need to copy so each environment variable isn't referencing the same pointer in key, value
		overrideValue := value
		envVars = append(envVars, &codebuild.EnvironmentVariable{
			Name:  &overrideKey,
			Value: &overrideValue,
			Type:  &envType,
		})
	}

	// Construct the CodeBuild Input
	input := &codebuild.StartBuildInput{
		ProjectName:                  projectName,
		EnvironmentVariablesOverride: envVars,
	}

	// Start the CodeBuild
	output, err := build.Client.StartBuild(input)
	if err != nil {
		return "", err
	}

	// Return the Build's ID
	return *output.Build.Id, nil
}
