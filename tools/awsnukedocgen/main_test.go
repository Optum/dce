package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

var expectedSupportedMethods []string
var expectedServiceNames []string
var expectedUnsupportedServiceNames []string

func init() {

	expectedSupportedMethods = []string{
		"athena:DeleteWorkGroup",
		"sts:DeleteWorkGroup",
		"autoscaling:DeleteAutoScalingGroup",
		"autoscaling:DeleteLaunchConfiguration",
		"batch:DeleteJobQueue",
		"cloud9:DeleteEnvironment",
		"dynamodb:DeleteTable",
		"eks:DeleteFargateProfile",
		"eks:DeleteNodegroup",
		"elasticloadbalancing:DeleteLoadBalancer",
		"elasticloadbalancing:DeleteLoadBalancer",
		"elasticloadbalancing:DeleteTargetGroup",
		"elasticmapreduce:DeleteSecurityConfiguration",
		"firehose:DeleteDeliveryStream",
		"glue:DeleteJob",
		"iam:DeleteUser",
		"iam:DeleteVirtualMFADevice",
		"kinesis:DeleteStream",
		"lambda:DeleteFunction",
		"machinelearning:DeleteBatchPrediction",
		"mq:DeleteBroker",
		"kafka:DeleteCluster",
		"opsworks:DeleteInstance",
		"rds:DeleteDBCluster",
		"rds:DeleteDBInstance",
		"neptune:DeleteDBInstance",
		"redshift:DeleteCluster",
		"route53:DeleteHealthCheck",
		"route53:DeleteHostedZone",
		"s3:DeleteBucketPolicy",
		"s3:DeleteBucket",
		"s3:DeleteObject",
		"sns:DeleteEndpoint",
		"sns:DeletePlatformApplication",
		"sns:DeleteTopic",
		"sqs:DeleteQueue",
		"waf:DeleteRule",
		"ec2:TerminateInstances",
		"ec2:TerminateInstances",
		"elasticmapreduce:TerminateJobFlows",
	}

	expectedServiceNames = []string{
		"Amazon Athena",
		"Amazon EC2",
		"Amazon EC2 Auto Scaling",
		"AWS Batch",
		"AWS Cloud9",
		"Amazon DynamoDB",
		"Amazon Elastic Container Service for Kubernetes",
		"Elastic Load Balancing",
		"Elastic Load Balancing V2",
		"Amazon Elastic MapReduce",
		"Amazon Kinesis Firehose",
		"AWS Glue",
		"Identity And Access Management",
		"Amazon Kinesis",
		"AWS Lambda",
		"Amazon Machine Learning",
		"Amazon MQ",
		"Amazon Managed Streaming for Kafka",
		"AWS OpsWorks",
		"Amazon RDS",
		"Amazon Redshift",
		"Amazon Route 53",
		"Amazon S3",
		"Amazon SNS",
		"Amazon SQS",
		"AWS WAF",
	}

	expectedUnsupportedServiceNames = []string{
		"AWS Accounts",
		"AWS Amplify",
		"AWS App Mesh",
		"AWS App Mesh Preview",
		"AWS AppConfig",
		"AWS AppSync",
		"AWS Artifact",
		"AWS Auto Scaling",
		"AWS Backup",
		"AWS Backup storage",
		"AWS Billing",
		"AWS Budget Service",
		"AWS Certificate Manager",
		"AWS Certificate Manager Private Certificate Authority",
		"AWS Chatbot",
		"AWS Cloud Map",
		"AWS CloudFormation",
		"AWS CloudHSM",
		"AWS CloudTrail",
		"AWS Code Signing for Amazon FreeRTOS",
		"AWS CodeBuild",
		"AWS CodeCommit",
		"AWS CodeDeploy",
		"AWS CodePipeline",
		"AWS CodeStar",
		"AWS CodeStar Notifications",
		"AWS Config",
		"AWS Cost Explorer Service",
		"AWS Cost and Usage Report",
		"AWS Data Exchange",
		"AWS Database Migration Service",
		"AWS DeepLens",
		"AWS DeepRacer",
		"AWS Device Farm",
		"AWS Direct Connect",
		"AWS Directory Service",
		"AWS Elastic Beanstalk",
		"AWS Elemental MediaConnect",
		"AWS Elemental MediaConvert",
		"AWS Elemental MediaLive",
		"AWS Elemental MediaPackage",
		"AWS Elemental MediaPackage VOD",
		"AWS Elemental MediaStore",
		"AWS Elemental MediaTailor",
		"AWS Firewall Manager",
		"AWS Global Accelerator",
		"AWS Ground Station",
		"AWS Health APIs and Notifications",
		"AWS IQ",
		"AWS IQ Permissions",
		"AWS Import Export Disk Service",
		"AWS IoT",
		"AWS IoT 1-Click",
		"AWS IoT Analytics",
		"AWS IoT Device Tester",
		"AWS IoT Events",
		"AWS IoT Greengrass",
		"AWS IoT SiteWise",
		"AWS IoT Things Graph",
		"AWS Key Management Service",
		"AWS Lake Formation",
		"AWS License Manager",
		"AWS Managed Apache Cassandra Service",
		"AWS Marketplace",
		"AWS Marketplace Catalog",
		"AWS Marketplace Entitlement Service",
		"AWS Marketplace Image Building Service",
		"AWS Marketplace Management Portal",
		"AWS Marketplace Metering Service",
		"AWS Marketplace Procurement Systems Integration",
		"AWS Migration Hub",
		"AWS Mobile Hub",
		"AWS OpsWorks Configuration Management",
		"AWS Organizations",
		"AWS Outposts",
		"AWS Performance Insights",
		"AWS Price List",
		"AWS Private Marketplace",
		"AWS Resource Access Manager",
		"AWS Resource Groups",
		"AWS RoboMaker",
		"AWS SSO",
		"AWS SSO Directory",
		"AWS Savings Plans",
		"AWS Secrets Manager",
		"AWS Security Hub",
		"AWS Security Token Service",
		"AWS Server Migration Service",
		"AWS Serverless Application Repository",
		"AWS Service Catalog",
		"AWS Shield",
		"AWS Snowball",
		"AWS Step Functions",
		"AWS Support",
		"AWS Systems Manager",
		"AWS Transfer for SFTP",
		"AWS Trusted Advisor",
		"AWS WAF Regional",
		"AWS WAF V2",
		"AWS Well-Architected Tool",
		"AWS X-Ray",
		"Alexa for Business",
		"Amazon API Gateway",
		"Amazon AppStream 2.0",
		"Amazon Chime",
		"Amazon Cloud Directory",
		"Amazon CloudFront",
		"Amazon CloudSearch",
		"Amazon CloudWatch",
		"Amazon CloudWatch Logs",
		"Amazon CloudWatch Synthetics",
		"Amazon CodeGuru Profiler",
		"Amazon CodeGuru Reviewer",
		"Amazon Cognito Identity",
		"Amazon Cognito Sync",
		"Amazon Cognito User Pools",
		"Amazon Comprehend",
		"Amazon Connect",
		"Amazon Data Lifecycle Manager",
		"Amazon Detective",
		"Amazon DynamoDB Accelerator (DAX)",
		"Amazon EC2 Image Builder",
		"Amazon EC2 Instance Connect",
		"Amazon ElastiCache",
		"Amazon Elastic Block Store",
		"Amazon Elastic Container Registry",
		"Amazon Elastic Container Service",
		"Amazon Elastic File System",
		"Amazon Elastic Inference",
		"Amazon Elastic Transcoder",
		"Amazon Elasticsearch Service",
		"Amazon EventBridge",
		"Amazon EventBridge Schemas",
		"Amazon FSx",
		"Amazon Forecast",
		"Amazon Fraud Detector",
		"Amazon FreeRTOS",
		"Amazon GameLift",
		"Amazon Glacier",
		"Amazon GroundTruth Labeling",
		"Amazon GuardDuty",
		"Amazon Inspector",
		"Amazon Kendra",
		"Amazon Kinesis Analytics",
		"Amazon Kinesis Analytics V2",
		"Amazon Kinesis Video Streams",
		"Amazon Lex",
		"Amazon Lightsail",
		"Amazon Macie",
		"Amazon Managed Blockchain",
		"Amazon Mechanical Turk",
		"Amazon Message Delivery Service",
		"Amazon Mobile Analytics",
		"Amazon Neptune",
		"Amazon Personalize",
		"Amazon Pinpoint",
		"Amazon Pinpoint Email Service",
		"Amazon Pinpoint SMS and Voice Service",
		"Amazon Polly",
		"Amazon QLDB",
		"Amazon QuickSight",
		"Amazon RDS Data API",
		"Amazon RDS IAM Authentication",
		"Amazon Rekognition",
		"Amazon Resource Group Tagging API",
		"Amazon Route 53 Resolver",
		"Amazon Route53 Domains",
		"Amazon SES",
		"Amazon SageMaker",
		"Amazon Session Manager Message Gateway Service",
		"Amazon Simple Workflow Service",
		"Amazon SimpleDB",
		"Amazon Storage Gateway",
		"Amazon Sumerian",
		"Amazon Textract",
		"Amazon Transcribe",
		"Amazon Translate",
		"Amazon WorkDocs",
		"Amazon WorkLink",
		"Amazon WorkMail",
		"Amazon WorkMail Message Flow",
		"Amazon WorkSpaces",
		"Amazon WorkSpaces Application Manager",
		"Application Auto Scaling",
		"Application Discovery",
		"Application Discovery Arsenal",
		"CloudWatch Application Insights",
		"Comprehend Medical",
		"Compute Optimizer",
		"Data Pipeline",
		"DataSync",
		"Database Query Metadata Service",
		"IAM Access Analyzer",
		"Launch Wizard",
		"Manage Amazon API Gateway",
		"Network Manager",
		"Service Quotas",
	}
}

func TestNukeParser_GetDeleteMethods(t *testing.T) {
	parser := NewNukeParser("samples/nuke")
	actualMethods, err := parser.GetDeleteMethods()
	assert.Nil(t, err, "expected no error from parsing known folder")
	assert.Equal(t, len(expectedSupportedMethods), len(actualMethods))
	assertListsEqual(t, expectedSupportedMethods, actualMethods)
	assertListsEqual(t, actualMethods, expectedSupportedMethods)
}

func TestNukeParser_GetDeleteMethods_BadFolder(t *testing.T) {
	parser := NewNukeParser("bad/folder/no/exist")
	_, err := parser.GetDeleteMethods()
	assert.NotNil(t, err, "expected error from parsing non-existant folder")
}

func TestPoliciesParser_Parse(t *testing.T) {
	deleteMethods := expectedSupportedMethods
	parser := NewPoliciesParser("samples/policies-20200227.js", deleteMethods)
	err := parser.Parse()
	assert.Nil(t, err, "expected no errors")
	serviceInfo := parser.SupportedDeleteMethods()
	// Two is how many services should be filtered out--there is two duplicates
	// and there is one that does not exist
	assert.Equal(t, (len(deleteMethods) - 3), len(serviceInfo))
	for _, si := range serviceInfo {
		isFound := false
		for _, e := range deleteMethods {
			if fmt.Sprintf("%s:%s", si.ServicePrefix, si.MethodName) == e {
				isFound = true
				break
			}
		}
		assert.True(t, isFound, "expected service %s to be found in list.", si.ServiceName)
	}

	services := parser.SupportedServices()
	assert.Equal(t, len(expectedServiceNames), len(services))
	assertListsEqual(t, expectedServiceNames, services)
	assertListsEqual(t, services, expectedServiceNames)

	unservices := parser.UnsupportedServices()
	assert.Equal(t, len(expectedUnsupportedServiceNames), len(unservices))
	assertListsEqual(t, expectedUnsupportedServiceNames, unservices)
	assertListsEqual(t, unservices, expectedUnsupportedServiceNames)

}

func TestMarkdownGenerator_Generate(t *testing.T) {

	parser := NewPoliciesParser("samples/policies-20200227.js", expectedSupportedMethods)
	err := parser.Parse()
	assert.Nil(t, err, "expected no errors")
	expectedMarkdown, err := os.ReadFile("samples/supported.md")
	assert.Nil(t, err, "expected no error reading from file.")

	iam := NewIAMPolicyGenerator(true, parser.SupportedDeleteMethods(), []string{})

	generator := NewMarkdownGenerator(
		parser.SupportedServices(),
		parser.UnsupportedServices(),
		iam,
	)
	actualMarkdown, err := generator.Generate()

	assert.Nil(t, err, "expected to error from Generate()")
	assert.Equal(t, string(expectedMarkdown), actualMarkdown)
}

func TestIAMPolicyGenerator_GeneratePolicy(t *testing.T) {

}

func assertListsEqual(t *testing.T, expected []string, actual []string) {
	for _, e := range expected {
		isFound := false
		for _, a := range actual {
			if a == e {
				isFound = true
				break
			}
		}
		assert.True(t, isFound, "expected service %s to be found in list.", e)
	}
}
