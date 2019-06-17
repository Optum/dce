package reset

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestGenerateConfig tests the GenerateConfig function to verify
// an example template file can have it's values substituted.
func TestGenerateConfig(t *testing.T) {
	subs := map[string]string{
		"{{id}}":      "123456789012",
		"{{service}}": "CloudFormationStack",
	}
	result, _ := GenerateConfig("testdata/test-config-template.yml", subs)
	expected, _ := ioutil.ReadFile("testdata/test-config-result.yml")

	strResult := string(result)
	strExpected := string(expected)
	assert.Equal(t, strResult, strExpected, "Result was: \n%s\nExpected was: \n%s\n", strResult, strExpected)
}

// TestGenerateConfig tests the SubstituteConfig function to verify
// an example []byte can have it's values substituted.
func TestSubstituteConfig(t *testing.T) {
	subs := map[string]string{
		"this":  "that",
		"first": "last",
	}
	template := []byte("this is my first name")
	result := SubstituteConfig(template, subs)
	expected := []byte("that is my last name")

	strResult := string(result)
	strExpected := string(expected)
	assert.Equal(t, strResult, strExpected, "Result was: \n%s\nExpected was: \n%s\n", strResult, strExpected)
}
