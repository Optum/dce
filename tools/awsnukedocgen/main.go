package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"

	"gopkg.in/yaml.v2"
)

var logger *log.Logger

const configFile string = "awsnukedocgen.yaml"
const iamTemplate string = `
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "DoNotModifySelf",
            "Effect": "Deny",
            "NotAction": [
                "iam:GetPolicy",
                "iam:GetPolicyVersion",
                "iam:GetRole",
                "iam:GetRolePolicy",
                "iam:ListRoles",
                "iam:ListRolePolicies",
                "iam:ListAttachedRolePolicies",
                "iam:ListRoleTags",
                "iam:ListPoliciesGrantingServiceAccess",
                "iam:ListEntitiesForPolicy",
                "iam:ListPolicyVersions",
                "iam:GenerateServiceLastAccessedDetails"
            ],
            "Resource": [
                "arn:aws:iam::123456789012:policy/DCEPrincipalDefaultPolicy",
                "arn:aws:iam::123456789012:role/DCEPrincipal",
                "arn:aws:iam::123456789012:role/OrganizationAccountAccessRole"
            ]
        },
        {
            "Sid": "DenyTaggedResourcesAWS",
            "Effect": "Deny",
            "Action": "*",
            "Resource": "*",
            "Condition": {
                "StringEquals": {
                    "aws:ResourceTag/AppName": [
                        "DCE"
                    ]
                }
            }
        },
        {
            "Sid": "DenyIAM",
            "Effect": "Deny",
            "Action": [
                "iam:DeactivateMFADevice",
                "iam:CreateSAMLProvider",
                "iam:UpdateAccountPasswordPolicy",
                "iam:DeleteVirtualMFADevice",
                "iam:EnableMFADevice",
                "iam:CreateAccountAlias",
                "iam:DeleteAccountAlias",
                "iam:UpdateSAMLProvider",
                "iam:DeleteSAMLProvider"
            ],
            "Resource": "*"
        },
        {
            "Sid": "AllowedServices",
            "Effect": "Allow",
            "Action": [
{{- range .ServiceShortNames}}
                "{{.}}:*",
{{- end}}
            ],
            "Resource": "*",
            "Condition": {
                "StringEquals": {
                    "aws:RequestedRegion": [
                        "us-east-1",
                        "us-west-1"
                    ]
                }
            }
        }
    ]
}
`
const markdownTemplate string = `# Account Cleanup with AWS Nuke

DCE uses a [fork of AWS Nuke](https://github.com/Optum/aws-nuke)
to facilitate account cleanup. Shown here is a list of services that are
supported and those that are not supported for account cleanup.

## Supported Services

{{range .SupportedServices -}}
* {{.}}
{{end}}

## Unsupported Services

{{range .UnsupportedServices -}}
* {{.}}
{{end}}

`

// ToolConfig stories the configuration for the services
type ToolConfig struct {
	AdditionalServices []string          `yaml:"additionalServices"`
	ServiceAliases     map[string]string `yaml:"serviceAliases"`
}

var configuration *ToolConfig

// init initializes the configuration by reading it from the
// yaml file.
func init() {
	logger = log.New(os.Stderr, "", 0)
	configuration = &ToolConfig{}

	yamlStr, err := ioutil.ReadFile(configFile)
	if err != nil {
		logger.Fatalln("error loading configuration", err)
	}

	err = yaml.Unmarshal(yamlStr, configuration)

	if err != nil {
		logger.Fatalln("error while trying to parse configuration", err)
	}
}

// GetUniqueValues returns a list of unique strings from the
// given values
func GetUniqueValues(from []string) []string {
	keys := make(map[string]bool)
	result := []string{}
	for _, entry := range from {
		if _, val := keys[entry]; !val {
			keys[entry] = true
			result = append(result, entry)
		}
	}
	return result
}

// exitWithErr exits the program with an error
func exitWithErr(err error) {
	logger.Fatalf("error: %v", err)
}

// fileExists returns true if the file with the given name exists.``
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func dirExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
}

// NukeParser parses the /resources folder in the AWS Nuke source to pull
// out the .\Delete methods.
type NukeParser struct {
	nukeSourceDir              string
	deleteMethodExpression     *regexp.Regexp
	serviceReferenceExpression *regexp.Regexp
}

// GetDeleteMethods returns a list of all of the delete methods used
// in the /resources folder of AWS Nukes
func (np *NukeParser) GetDeleteMethods() ([]string, error) {
	return np.scanForSupportedDeleteMethods(
		np.serviceReferenceExpression,
		np.deleteMethodExpression,
	)
}

// Close closes the resource because file.Close() returns an error,
// which is flagged by gosec.
func (np *NukeParser) Close(f *os.File) {
	err := f.Close()
	if err != nil {
		logger.Printf("could not close file: %s", f.Name())
	}
}

// scanForExpression scans for the given expression in the directory
func (np *NukeParser) scanForSupportedDeleteMethods(
	refExpr *regexp.Regexp,
	methodExpr *regexp.Regexp,
) ([]string, error) {
	results := make([]string, 0)

	dir := path.Join(np.nukeSourceDir, "resources")
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return results, fmt.Errorf("error while trying to get files: %v", err)
	}

	for _, f := range files {
		logger.Printf("scanning file: %s", f.Name())
		methods := make([]string, 0)
		services := make([]string, 0)
		file, err := os.Open(filepath.Clean(path.Join(dir, f.Name())))
		if err != nil {
			logger.Printf("error opening file \"%s\": %v", f.Name(), err)
			continue
		}
		defer np.Close(file)
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			refGroup := refExpr.FindAllStringSubmatch(scanner.Text(), -1)
			if len(refGroup) > 0 {
				val := refGroup[0][1]
				// look up the name of the service in the map, and if it's
				// found in the overrides, add in the overridden value...
				override, ok := configuration.ServiceAliases[val]
				if ok {
					services = append(services, override)
				} else {
					services = append(services, val)
				}
			}

			methodGroup := methodExpr.FindAllStringSubmatch(scanner.Text(), -1)
			if len(methodGroup) == 0 {
				continue
			}
			val := methodGroup[0][1]
			methods = append(methods, val)
		}

		for _, svc := range services {
			for _, method := range methods {
				results = append(results, fmt.Sprintf("%s:%s", svc, method))
			}
		}
	}

	return results, nil
}

// NewNukeParser creates a new instance of the NukeParser
func NewNukeParser(sourceDir string) *NukeParser {
	return &NukeParser{
		nukeSourceDir:              sourceDir,
		deleteMethodExpression:     regexp.MustCompile(`^.*\.(Delete[^(]+|Terminate[^(]+)\(.*`),
		serviceReferenceExpression: regexp.MustCompile(`^.*aws-sdk-go\/service\/([^\/]+)\"\s*$`),
	}
}

// SupportedDelete contains the short name and the supported delete
// method that was found in AWS Nuke.
type SupportedDelete struct {
	ServiceName   string
	ServicePrefix string
	MethodName    string
}

// ServiceInfo contains the short name and available actions for the
// service.
type ServiceInfo struct {
	ServiceName *string   `json:"StringPrefix"`
	Actions     []*string `json:"Actions"`
}

// PoliciesConfig contains the policy configuration
type PoliciesConfig struct {
	ServiceMap map[string]ServiceInfo `json:"serviceMap"`
}

// PoliciesParser parses the policies file to get the name of the service
// and the actions supported by the services
// The JSON looks like this:
//    "serviceMap": {
//        "Amazon Comprehend": {
//            "StringPrefix": "comprehend",
//            "Actions": [
// ...
type PoliciesParser struct {
	policiesJSFile      string
	deleteMethods       []string
	supportedDeletes    []*SupportedDelete
	supportedServices   []string
	unsupportedServices []string
}

// SupportedServices returns a list of supported services with "nice"
// descriptions.
func (pp *PoliciesParser) SupportedServices() []string {
	return pp.supportedServices
}

// UnsupportedServices returns a list of unsupported services with "nice"
// descriptions
func (pp *PoliciesParser) UnsupportedServices() []string {
	return pp.unsupportedServices
}

// SupportedDeleteMethods returns a list of supported delete operations
func (pp *PoliciesParser) SupportedDeleteMethods() []*SupportedDelete {
	return pp.supportedDeletes
}

// Parse parses the policies file and returns a list of supported
// services
func (pp *PoliciesParser) Parse() error {
	deletes := make([]*SupportedDelete, 0)
	supportedServices := make([]string, 0)
	unsupportedServices := make([]string, 0)

	if !fileExists(pp.policiesJSFile) {
		return fmt.Errorf("file %s does not exist", pp.policiesJSFile)
	}

	policies := &PoliciesConfig{}
	file, err := os.Open(pp.policiesJSFile)

	if err != nil {
		log.Fatalf("error opening file: %s", err.Error())
	}

	bytes, _ := ioutil.ReadAll(file)
	err = json.Unmarshal(bytes, policies)

	if err != nil {
		log.Fatalf("error while unmarshaling JSON file: %s", err.Error())
	}

	// now iterate through the service maps, and for each of those
	// iterate through the actions. If the action is a supported method,
	// we'll
	for svc, info := range policies.ServiceMap {
		isFound := false
		for _, action := range info.Actions {
			if pp.isSupported(fmt.Sprintf("%s:%s", *info.ServiceName, *action)) {
				deletes = append(deletes, &SupportedDelete{
					ServiceName:   svc,
					ServicePrefix: *info.ServiceName,
					MethodName:    *action,
				})
				isFound = true
				supportedServices = append(supportedServices, svc)
			}
		}
		if !isFound {
			unsupportedServices = append(unsupportedServices, svc)
		}
	}

	// now, as a double check, we'll get the list of all services that are
	// referenced by the source code. If it's a service that's not referenced
	// at all we'll remove it from the list.
	pp.supportedDeletes = deletes
	pp.supportedServices = GetUniqueValues(supportedServices)
	pp.unsupportedServices = GetUniqueValues(unsupportedServices)
	return nil
}

func (pp *PoliciesParser) isSupported(method string) bool {
	for _, m := range pp.deleteMethods {
		if m == method {
			return true
		}
	}
	return false
}

// NewPoliciesParser creates a news instance of *PoliciesParser
func NewPoliciesParser(policiesJS string, methods []string) *PoliciesParser {
	return &PoliciesParser{
		policiesJSFile:      policiesJS,
		deleteMethods:       methods,
		supportedDeletes:    make([]*SupportedDelete, 0),
		supportedServices:   make([]string, 0),
		unsupportedServices: make([]string, 0),
	}
}

// MarkdownGenerator generates Markdown documentation that identifies
// which services are supported by AWS Nuke and which service are NOT
// supported by AWS Nuke
type MarkdownGenerator struct {
	SupportedServices   []string
	UnsupportedServices []string
	IAMExample          string
}

// Generate returns a string of text in Markdown format that
// documents what is supported and what AWS services are not supported
// for use in DCE
func (mg *MarkdownGenerator) Generate() (string, error) {
	sort.Strings(mg.SupportedServices)
	sort.Strings(mg.UnsupportedServices)
	buf := new(bytes.Buffer)
	tplate := template.Must(template.New("md").Parse(markdownTemplate))
	err := tplate.Execute(buf, mg)
	return buf.String(), err
}

// NewMarkdownGenerator creates a new instance of the MarkdownGenerator
func NewMarkdownGenerator(supported []string, unsupported []string, ipg *IAMPolicyGenerator) *MarkdownGenerator {
	policy, err := ipg.GeneratePolicy()
	if err != nil {
		policy = ""
	}
	return &MarkdownGenerator{
		SupportedServices:   supported,
		UnsupportedServices: unsupported,
		IAMExample:          policy,
	}
}

// IAMPolicyGenerator combines the functions used by AWS nuke and all
// of those supported by AWS to create a policy.
type IAMPolicyGenerator struct {
	usePermissive     bool
	SupportedDeletes  []*SupportedDelete
	ServiceShortNames []string
}

// GeneratePolicy generates the IAM policy
func (ipg *IAMPolicyGenerator) GeneratePolicy() (string, error) {
	buf := new(bytes.Buffer)
	tplate := template.Must(template.New("pol").Parse(iamTemplate))
	err := tplate.Execute(buf, ipg)
	return buf.String(), err
}

// NewIAMPolicyGenerator creates a new instance of *IAMPolicyGenerator
func NewIAMPolicyGenerator(permissive bool, methods []*SupportedDelete, overrides []string) *IAMPolicyGenerator {
	services := make([]string, 0)
	for _, sd := range methods {
		services = append(services, sd.ServicePrefix)
	}
	// Add in the overrides for additional services that should be included but
	// are parsed incorrectly or not at all.
	services = append(services, overrides...)
	services = GetUniqueValues(services)
	sort.Strings(services)
	return &IAMPolicyGenerator{
		usePermissive:     permissive,
		SupportedDeletes:  methods,
		ServiceShortNames: services,
	}
}

// main method for the iampolgen tool
func main() {

	logger.Println("Starting now...")

	policiesJSFile := flag.String("policies-js-file", "./policies.js", "name of the file containing IAM policy configuration.")
	generateMarkdown := flag.Bool("generate-markdown", false, "if true, tells the generator to generate a Markdown file documenting the gaps")
	nukeSourceDir := flag.String("nuke-source-dir", "", "location of the AWS nuke source")
	iamUsePermissive := flag.Bool("iam-use-permissive", true, "if set to true, will generate permissive polcies using wildcards")

	flag.Parse()

	// Do some basic checking to make sure the args supplied are good.
	if !fileExists(*policiesJSFile) {
		exitWithErr(fmt.Errorf("specified policy file \"%s\" does not exist", *policiesJSFile))
	}

	if len(*nukeSourceDir) == 0 {
		exitWithErr(fmt.Errorf("must specify a valid AWS nuke directory"))
	}

	if !dirExists(*nukeSourceDir) {
		exitWithErr(fmt.Errorf("specified AWS nuke source directory \"%s\" does not exist", *nukeSourceDir))
	}

	// get the list of delete commands that are used by AWS Nuke
	nuke := NewNukeParser(*nukeSourceDir)
	methods, err := nuke.GetDeleteMethods()

	if err != nil {
		exitWithErr(fmt.Errorf("error whilst trying to get methods: %v", err))
	}

	// get the list of the AWS services and their delete permissions
	policy := NewPoliciesParser(*policiesJSFile, methods)
	err = policy.Parse()

	if err != nil {
		exitWithErr(fmt.Errorf("error parsing policies file: %v", err))
	}

	// now hand these off to the generator, which will compare the two
	// and generate an IAM policy using awesomeness.
	polgen := NewIAMPolicyGenerator(*iamUsePermissive, policy.SupportedDeleteMethods(), configuration.AdditionalServices)

	if *generateMarkdown {
		logger.Printf("generating Markdown file now...")
		markdown := NewMarkdownGenerator(
			policy.SupportedServices(),
			policy.UnsupportedServices(),
			polgen,
		)
		md, err := markdown.Generate()
		if err != nil {
			exitWithErr(fmt.Errorf("error while generating markdown: %v", err))
		}
		fmt.Println(md)
		return
	}

	samplePolicy, err := polgen.GeneratePolicy()

	if err != nil {
		exitWithErr(fmt.Errorf("error while creating sample policy: %v", err))
	}

	fmt.Println(samplePolicy)

}
