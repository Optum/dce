package main

import (
	"github.com/Optum/dce/pkg/common"
	"github.com/Optum/dce/pkg/config"
	"github.com/Optum/dce/pkg/db"
	"github.com/aws/aws-sdk-go/aws/session"
)

type resetConfig struct {
	// Env Vars
	AccountID             string   `env:"RESET_ACCOUNT"`
	AdminRoleName         string   `env:"RESET_ACCOUNT_ADMIN_ROLE_NAME"`
	PrincipalRoleName     string   `env:"RESET_ACCOUNT_PRINCIPAL_ROLE_NAME"`
	PrincipalPolicyName   string   `env:"RESET_ACCOUNT_PRINCIPAL_POLICY_NAME"`
	IsNukeEnabled         bool     `env:"RESET_NUKE_TOGGLE"`
	NukeTemplateDefault   string   `env:"RESET_NUKE_TEMPLATE_DEFAULT"`
	NukeTemplateBucket    string   `env:"RESET_NUKE_TEMPLATE_BUCKET"`
	NukeTemplateKey       string   `env:"RESET_NUKE_TEMPLATE_KEY"`
	ResetCompleteTopicARN string   `env:"RESET_COMPLETE_TOPIC_ARN"`
	NukeRegions           []string `env:"RESET_NUKE_REGIONS"`

	Session      *session.Session
	TokenService common.TokenService
	S3           common.Storager
	DB           db.DBer
	SNS          common.Notificationer
}

func (c *resetConfig) AdminRoleARN() string {
	return "arn:aws:iam::" + c.AccountID + ":role/" + c.AdminRoleName
}

func initConfig() (*resetConfig, error) {
	var conf resetConfig
	cfgBldr := &config.ServiceBuilder{}
	_, err := cfgBldr.
		WithTokenService().
		WithStorager().
		WithDB().
		WithNotificationer().
		Build()
	if err != nil {
		return nil, err
	}
	err = cfgBldr.Unmarshal(&conf)
	return &conf, err
}
