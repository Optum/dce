package main

import (
	"bytes"
	"github.com/Optum/Dce/pkg/db"
	"github.com/Optum/Dce/pkg/email"
	"html/template"
	"log"
	"sort"
	"strings"
)

type sendBudgetNotificationEmailInput struct {
	lease                                  *db.DceLease
	emailSvc                               email.Service
	budgetNotificationFromEmail            string
	budgetNotificationBCCEmails            []string
	budgetNotificationTemplateHTML         string
	budgetNotificationTemplateText         string
	budgetNotificationTemplateSubject      string
	budgetNotificationThresholdPercentiles []float64
	actualSpend                            float64
}

func sendBudgetNotificationEmail(input *sendBudgetNotificationEmailInput) error {
	// Determine the highest budget threshold passed
	thresholdPercentile := determineThresholdPercentile(&determineThresholdPercentileInput{
		thresholdPercentiles: input.budgetNotificationThresholdPercentiles,
		budgetAmount:         input.lease.BudgetAmount,
		actualSpend:          input.actualSpend,
	})
	if thresholdPercentile == 0 {
		return nil
	}

	if len(input.lease.BudgetNotificationEmails)+len(input.budgetNotificationBCCEmails) == 0 {
		log.Printf("Skipping budget notification emails: "+
			"no notification emails addressses were provided for lease %s @ %s",
			input.lease.PrincipalID, input.lease.AccountID)
		return nil
	}

	log.Printf("Budget notification threshold hit at %.0f%%", thresholdPercentile)
	log.Printf("Sending budget notification emails for lease %s @ %s to %s",
		input.lease.PrincipalID, input.lease.AccountID, strings.Join(input.lease.BudgetNotificationEmails, ","))

	// Render email templates
	templateData := struct {
		Lease               db.DceLease
		ActualSpend         float64
		IsOverBudget        bool
		ThresholdPercentile int
	}{
		Lease:               *input.lease,
		ActualSpend:         input.actualSpend,
		IsOverBudget:        input.actualSpend >= input.lease.BudgetAmount,
		ThresholdPercentile: int(thresholdPercentile),
	}
	bodyHTML, err := renderTemplate("htmlEmail", input.budgetNotificationTemplateHTML, templateData)
	if err != nil {
		return err
	}
	bodyText, err := renderTemplate("textEmail", input.budgetNotificationTemplateText, templateData)
	if err != nil {
		return err
	}

	subject, err := renderTemplate("emailSubject", input.budgetNotificationTemplateSubject, templateData)
	if err != nil {
		return err
	}

	return input.emailSvc.SendEmail(&email.SendEmailInput{
		FromAddress:  input.budgetNotificationFromEmail,
		ToAddresses:  input.lease.BudgetNotificationEmails,
		BCCAddresses: input.budgetNotificationBCCEmails,
		BodyHTML:     bodyHTML,
		BodyText:     bodyText,
		Subject:      subject,
	})
}

func renderTemplate(id string, templateStr string, data interface{}) (string, error) {
	tmpl, err := template.New(id).Parse(templateStr)
	if err != nil {
		return "", err
	}

	buf := &bytes.Buffer{}
	err = tmpl.Execute(buf, data)

	return strings.TrimSpace(buf.String()), err
}

type determineThresholdPercentileInput struct {
	thresholdPercentiles []float64
	budgetAmount         float64
	actualSpend          float64
}

func determineThresholdPercentile(input *determineThresholdPercentileInput) float64 {
	// Sort threshold percentiles in increasing order
	sort.Float64s(input.thresholdPercentiles)

	// Find the highest threshold that we've reached
	var thresholdPassed float64
	for _, thresholdPercentile := range input.thresholdPercentiles {
		thresholdAmount := input.budgetAmount * (thresholdPercentile / 100)
		if input.actualSpend >= thresholdAmount {
			thresholdPassed = thresholdPercentile
		}
	}

	return thresholdPassed
}
