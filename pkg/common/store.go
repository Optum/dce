package common

import "github.com/aws/aws-sdk-go/service/ssm"

// Storer interface requires a method to retrieve values from a Parameter Store
type Storer interface {
	GetParameter(*string) (*string, error)
}

// SSM implements the Storer interface using AWS SSM Client
type SSM struct {
	Client *ssm.SSM
}

// GetParameter returns the value in the Parameter Store, provided by the key
func (manager SSM) GetParameter(input *string) (*string, error) {
	// Get the Parameter with SSM
	decrypt := true
	getParameterInput := ssm.GetParameterInput{
		Name:           input,
		WithDecryption: &decrypt,
	}
	output, err := manager.Client.GetParameter(&getParameterInput)
	if err != nil {
		return nil, err
	}

	// Return the output's Parameter value
	return output.Parameter.Value, nil
}
