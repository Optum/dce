package tests

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/gruntwork-io/terratest/modules/terraform"
)

func TestTerraformOutputs(t *testing.T) {
	tfOpts := &terraform.Options{
		TerraformDir: "../../modules",
	}

	tfOut := terraform.OutputAll(t, tfOpts)

	assert.Regexp(t,
		regexp.MustCompile("^Accounts"),
		tfOut["redbox_account_db_table_name"].(string),
		"redbox_account_db_table_name",
	)
	assert.Regexp(t,
		regexp.MustCompile(`^Leases`),
		tfOut["redbox_lease_db_table_name"].(string),
		"redbox_lease_db_table_name",
	)
	assert.Regexp(t,
		regexp.MustCompile(`^https:\/\/sqs\.us-east-1\.amazonaws\.com\/[0-9]+\/redbox-account-reset-`),
		tfOut["sqs_reset_queue_url"].(string),
		"sqs_reset_queue_url",
	)
	assert.Regexp(t,
		regexp.MustCompile(`^arn:aws:sqs:us-east-1:[0-9]+:redbox-account-reset-`),
		tfOut["sqs_reset_queue_arn"].(string),
		"sqs_reset_queue_arn",
	)

}
