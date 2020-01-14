package main

import (
	"github.com/Optum/dce/pkg/common"
	"github.com/Optum/dce/pkg/config"
	"github.com/Optum/dce/pkg/db"
	"github.com/Optum/dce/pkg/usage"
)

type leasesConfig struct {
	LeaseAddedTopicARN       string  `env:"LEASE_ADDED_TOPIC" envDefault:"DCEDefaultProvisionTopic"`
	PrincipalBudgetAmount    float64 `env:"PRINCIPAL_BUDGET_AMOUNT" envDefault:"1000"`
	PrincipalBudgetPeriod    string  `env:"PRINCIPAL_BUDGET_PERIOD" envDefault:"WEEKLY"`
	MaxLeaseBudgetAmount     float64 `env:"MAX_LEASE_BUDGET_AMOUNT" envDefault:"1000"`
	MaxLeasePeriod           int64   `env:"MAX_LEASE_PERIOD" envDefault:"704800"`
	DefaultLeaseLengthInDays int     `env:"DEFAULT_LEASE_LENGTH_IN_DAYS" envDefault:"7"`

	SNS   common.Notificationer
	DB    db.DBer
	Usage usage.Service
}

func initConfig() (*leasesConfig, error) {
	var conf leasesConfig
	cfgBldr := &config.ServiceBuilder{}
	_, err := cfgBldr.
		WithNotificationer().
		WithDB().
		WithUsageService().
		Build()
	if err != nil {
		return nil, err
	}
	err = cfgBldr.Unmarshal(&conf)
	return &conf, err
}
