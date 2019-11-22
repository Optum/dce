package tests

import (
	"testing"

	"github.com/Optum/dce/pkg/config"
	"github.com/stretchr/testify/assert"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
)

func TestS3Client_FromConfig(t *testing.T) {
	// TODO: Not sure if I need anything from Terraform here...
	// Load Terraform outputs
	// tfOpts := &terraform.Options{
	// 	TerraformDir: "../../modules",
	// }
	// tfOut := terraform.OutputAll(t, tfOpts)
	// artifactsBucket := tfOut["artifacts_bucket_name"].(string)

	cfgBldr := &config.DCEConfigBuilder{}
	svcBldr := &config.AWSServiceBuilder{Config: cfgBldr}

	_, err := svcBldr.WithS3().Build()

	if err != nil {
		var s3Client s3iface.S3API
		err = svcBldr.Config.GetService(&s3Client)

		// try to use the client to list the bucket
		result, err := s3Client.ListBuckets(&s3.ListBucketsInput{})
		assert.Nil(t, err)
		assert.NotNil(t, result)
	}
}
