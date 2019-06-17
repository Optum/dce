package reset

import (
	"bytes"
	"io/ioutil"
)

// GenerateConfig will create a new Config off a template with
// phrases to be interpolated from.
// Returns the []byte of the interpolated config and any errors.
func GenerateConfig(templatePath string, accountSubs map[string]string) ([]byte, error) {
	var err error
	var nukeTemplate []byte

	// Read the template file
	nukeTemplate, err = ioutil.ReadFile(templatePath)
	if err != nil {
		return nil, err
	}

	// Replace all instances of phrases/keys within accountSubs with its respective values
	// and return the response.
	return SubstituteConfig(nukeTemplate, accountSubs), nil
}

// SubstituteConfig replaces all instances of phrases/keys in the template with the
// respected values in the provided map.
// Returns a []byte of the Config with the substituted values.
func SubstituteConfig(template []byte, subs map[string]string) []byte {
	config := template
	for toSub, withSub := range subs {
		config = bytes.Replace(config, []byte(toSub), []byte(withSub), -1)
	}
	return config
}
