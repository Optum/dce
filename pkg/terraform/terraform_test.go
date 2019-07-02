package terraform

import (
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"testing"
)

func TestTerraform(t *testing.T) {

	t.Run("InitAndApply, Output", func(t *testing.T) {

		testCases := []struct {
			name                      string
			hcl                       string
			expectedOutput            map[string]string
			expectedInitAndApplyError string
			tfvars                    map[string]string
			applyTimeout              float64
			initTimeout               float64
		}{
			{
				name: "should apply a terraform configuration, and get output",
				hcl: `
					resource "null_resource" "test" {
					  provisioner "local-exec" {
						command = "echo 'Hello world!'"
					  }
					}
					output "foo" {
						value = "bar"
					}`,
				expectedOutput: map[string]string{"foo": "bar"},
			},
			{
				name: "should not require outputs",
				hcl: `
					resource "null_resource" "test" {
					  provisioner "local-exec" {
						command = "echo 'Hello world!'"
					  }
					}`,
				expectedOutput: map[string]string{},
			},
			{
				name: "should pass vars to the module",
				hcl: `
					variable "foo" {}
					variable "faz" {}
					output "foo_out" { value = "${var.foo}_out" }	
					output "faz_out" { value = "${var.faz}_out" }	
				`,
				tfvars: map[string]string{
					"foo": "bar",
					"faz": "baz",
				},
				expectedOutput: map[string]string{
					"foo_out": "bar_out",
					"faz_out": "baz_out",
				},
			},
			{
				name:                      "should fail for invalid HCL",
				hcl:                       `not valid HCL`,
				expectedInitAndApplyError: "The Terraform configuration must be valid before initialization",
			},
			{
				name: "should fail apply after timeout",
				// Execute slow running HCL
				hcl: `
					resource "null_resource" "test" {
					  provisioner "local-exec" {
						command = "sleep 5"
					  }
					}`,
				expectedInitAndApplyError: "Command timed out: terraform apply",
				applyTimeout:              0.05,
			},
			{
				name:                      "should fail init after timeout",
				expectedInitAndApplyError: "Command timed out: terraform init",
				// Just set a really low timeout, so this always fails
				initTimeout: 0.001,
			},
		}

		for _, test := range testCases {
			// See https://gist.github.com/posener/92a55c4cd441fc5e5e85f27bca008721#how-to-solve-this
			test := test

			t.Run(test.name, func(t *testing.T) {
				// Tests all run in temp dir, so we can safely run in parallel
				// This saves 10-20 seconds in test run time
				t.Parallel()

				// Create a Terraform client, pointing at a temp dir
				tf, cleanup, err := terraformTemp(test.hcl)
				require.Nil(t, err)
				defer cleanup()

				// Apply TF Config
				err = tf.InitAndApply(&InitAndApplyInput{
					Vars:         test.tfvars,
					ApplyTimeout: test.applyTimeout,
					InitTimeout:  test.initTimeout,
				})
				// Check for expected error
				if test.expectedInitAndApplyError != "" {
					require.NotNil(t, err)
					require.Contains(t, err.Error(), test.expectedInitAndApplyError)
					return
				}
				require.Nil(t, err, "InitAndApply")

				// Grab outputs
				outputs, err := tf.Output()
				require.Nil(t, err, "Output")
				require.NotNil(t, outputs)

				require.Equal(t, test.expectedOutput, outputs)
			})

		}

	})
}

// Create a Terraform client, pointing at a temp dir
func terraformTemp(hcl string) (*Terraform, func() error /*cleanup*/, error) {
	// Create a temp dir for module testing
	moduleDir, err := ioutil.TempDir("", "go-test-terraform")
	if err != nil {
		return nil, nil, err
	}
	// Create a "callback" function,
	// so the caller can cleanup the temp dir
	cleanup := func() error {
		return os.RemoveAll(moduleDir)
	}

	// Write some HCL to our module
	err = ioutil.WriteFile(moduleDir+"/main.tf", []byte(hcl), 0644)
	if err != nil {
		return nil, nil, err
	}

	// Create the Terraform client
	tf := &Terraform{
		ModuleDir:       moduleDir,
		TerraformBinary: "terraform",
	}

	return tf, cleanup, nil
}
