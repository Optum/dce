package commoniface

import "github.com/aws/aws-sdk-go/service/codebuild/codebuildiface"

type CodeBuildAPI interface {
	codebuildiface.CodeBuildAPI
}
