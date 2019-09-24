package terraform

import (
	"encoding/json"
	"github.com/Optum/Dcs/pkg/shell"
	"github.com/pkg/errors"
)

// Terraform implements the Terraformer interface
type Terraform struct {
	ModuleDir        string
	BackendBucket    string
	BackendLockTable string
	TerraformBinary  string
}

// Terraformer is a Go wrapper around the Terraform CLI
type Terraformer interface {
	InitAndApply(input InitAndApplyInput) error
	Output() map[string]string
}

func (tf *Terraform) bin() string {
	if tf.TerraformBinary == "" {
		return "terraform"
	}
	return tf.TerraformBinary
}

// InitAndApplyInput contains parameters for the Terraformer.InitAndApply operation
type InitAndApplyInput struct {
	Vars         map[string]string
	InitTimeout  float64
	ApplyTimeout float64
}

// InitAndApply initializes a terraform module,
// and applies the terraform configuration
func (tf *Terraform) InitAndApply(input *InitAndApplyInput) error {
	// Configure backend
	var backendArgs []string
	if tf.BackendBucket != "" {
		backendArgs = append(backendArgs,
			[]string{"-backend-config", "bucket=" + tf.BackendBucket}...,
		)
	}
	if tf.BackendLockTable != "" {
		backendArgs = append(backendArgs,
			[]string{"-backend-config", "dynamodb_table=" + tf.BackendLockTable}...,
		)
	}

	// Prepare `-var` cli flags
	var varFlags []string
	for key, value := range input.Vars {
		varFlags = append(varFlags, "-var", key+"="+value)
	}

	// Terraform init
	_, err := shell.Exec(&shell.ExecInput{
		Name: tf.bin(),
		Args: append([]string{
			"init",
			"-no-color",
		}, backendArgs...),
		Dir:     tf.ModuleDir,
		Timeout: input.InitTimeout,
	})
	if err != nil {
		return err
	}

	// Terraform apply
	_, err = shell.Exec(&shell.ExecInput{
		Name: tf.bin(),
		Args: append([]string{
			"apply",
			"-no-color",
			"-auto-approve",
			"-input=false",
		}, varFlags...),
		Dir:     tf.ModuleDir,
		Timeout: input.ApplyTimeout,
	})

	return err
}

// Output returns a key/value map of terraform outputs.
// NOTE: this method only supports string outputs,
// 	and will ignore any non-string types
func (tf *Terraform) Output() (map[string]string, error) {
	// Run `terraform output` command
	shOut, err := shell.Exec(&shell.ExecInput{
		Name: tf.bin(),
		Args: []string{
			"output",
			"-no-color",
			"-json",
		},
		Dir:     tf.ModuleDir,
		Timeout: 30,
	})
	if err != nil {
		return nil, err
	}

	// Parse output JSON
	var outputMap map[string]map[string]interface{}
	if err := json.Unmarshal([]byte(shOut), &outputMap); err != nil {
		return nil, err
	}

	// Grab values from output JSON
	output := map[string]string{}
	for key, valObj := range outputMap {
		if oType, ok := valObj["type"]; ok && oType == "string" {
			if _, ok = valObj["value"]; !ok {
				return nil, errors.New("Unexpected terraform output JSON: " + shOut)
			}
			output[key] = valObj["value"].(string)
		}
	}

	return output, nil
}
