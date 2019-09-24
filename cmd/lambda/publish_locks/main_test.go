package main

import (
	"testing"

	"github.com/Optum/Dcs/pkg/db"
	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
)

func TestLeaseFromImageSuccess(t *testing.T) {

	// Arrange

	expectedOutput := db.DcsLease{
		AccountID:                "TestAccountID",
		PrincipalID:              "TestPrincipalID",
		LeaseStatus:              "FINANCELOCKED",
		CreatedOn:                1565723448,
		LastModifiedOn:           1565723448,
		BudgetAmount:             300,
		BudgetCurrency:           "USD",
		BudgetNotificationEmails: []string{"recipA@example.com", "recipB@example.com"},
		LeaseStatusModifiedOn:    1565723448,
	}

	email1 := events.NewStringAttribute("recipA@example.com")
	email2 := events.NewStringAttribute("recipB@example.com")
	budgetNotificationEmails := []events.DynamoDBAttributeValue{email1, email2}

	var input = map[string]events.DynamoDBAttributeValue{
		"accountId":                events.NewStringAttribute("TestAccountID"),
		"principalId":              events.NewStringAttribute("TestPrincipalID"),
		"leaseStatus":              events.NewStringAttribute("FINANCELOCKED"),
		"createdOn":                events.NewNumberAttribute("1565723448"),
		"lastModifiedOn":           events.NewNumberAttribute("1565723448"),
		"budgetAmount":             events.NewNumberAttribute("300.000"),
		"budgetCurrency":           events.NewStringAttribute("USD"),
		"budgetNotificationEmails": events.NewListAttribute(budgetNotificationEmails),
		"leaseStatusModifiedOn":    events.NewNumberAttribute("1565723448"),
	}

	actualOutput, err := leaseFromImage(input)

	assert.Nil(t, err)
	assert.Equal(t, actualOutput, &expectedOutput)
}
