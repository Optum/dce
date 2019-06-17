// Package common contains interfaces and implementations for services and
package common

import (
	"github.com/aws/aws-sdk-go/service/codepipeline"
)

// Pipeline interface requires a method to start a CodePipline Pipeline.
type Pipeline interface {
	StartPipeline(*codepipeline.StartPipelineExecutionInput) (
		*codepipeline.StartPipelineExecutionOutput, error)
}

// CodePipeline implments the Pipeline interface using the AWS CodePipeline
// Service.
type CodePipeline struct {
	Client *codepipeline.CodePipeline
}

// StartPipeline method starts a CodePipeline Pipeline based on the provided
// CodePipeline Execution Input.
func (pipeline CodePipeline) StartPipeline(
	input *codepipeline.StartPipelineExecutionInput) (
	*codepipeline.StartPipelineExecutionOutput, error) {
	return pipeline.Client.StartPipelineExecution(input)
}
