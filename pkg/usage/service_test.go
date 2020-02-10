package usage_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/Optum/dce/pkg/errors"
	"github.com/Optum/dce/pkg/usage"
	"github.com/Optum/dce/pkg/usage/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var now = time.Now()
var nowPlusOneNow = now.AddDate(0, 0, 1)
var startDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).Unix()
var startDatePlusOne = time.Date(nowPlusOneNow.Year(), nowPlusOneNow.Month(), nowPlusOneNow.Day(), 0, 0, 0, 0, time.UTC).Unix()
var endDate = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, time.UTC).Unix()
var endDatePlusOne = time.Date(nowPlusOneNow.Year(), nowPlusOneNow.Month(), nowPlusOneNow.Day(), 0, 0, 0, 0, time.UTC).Unix()
var principalID = "principal"
var accountID = "123456789012"

func TestGetUsage(t *testing.T) {

	type response struct {
		data *usage.Usage
		err  error
	}

	tests := []struct {
		name        string
		ID          string
		StartDate   int64
		PrincipalID string
		ret         response
		exp         response
	}{
		{
			name:        "should get usage by start date and principal ID",
			StartDate:   startDate,
			PrincipalID: principalID,
			ret: response{
				data: &usage.Usage{
					StartDate:   &startDate,
					PrincipalID: &principalID,
				},
				err: nil,
			},
			exp: response{
				data: &usage.Usage{
					StartDate:   &startDate,
					PrincipalID: &principalID,
				},
				err: nil,
			},
		},
		{
			name: "should get failure",
			ret: response{
				data: nil,
				err:  errors.NewInternalServer("failure", fmt.Errorf("original failure")),
			},
			exp: response{
				data: nil,
				err:  errors.NewInternalServer("failure", fmt.Errorf("original failure")),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mocksRwd := &mocks.ReaderWriter{}

			mocksRwd.On("Get", tt.StartDate, tt.PrincipalID).Return(tt.ret.data, tt.ret.err)

			usageSvc := usage.NewService(usage.NewServiceInput{
				DataSvc: mocksRwd,
			})

			usg, err := usageSvc.Get(tt.StartDate, tt.PrincipalID)
			assert.True(t, errors.Is(err, tt.exp.err), "actual error %q doesn't match expected error %q", err, tt.exp.err)

			assert.Equal(t, tt.exp.data, usg)
		})
	}
}

func TestList(t *testing.T) {

	type response struct {
		data *usage.Usages
		err  error
	}

	tests := []struct {
		name      string
		inputData usage.Usage
		ret       response
		exp       response
	}{
		{
			name: "standard",
			inputData: usage.Usage{
				AccountID: &accountID,
			},
			ret: response{
				data: &usage.Usages{
					usage.Usage{
						StartDate:   &startDate,
						PrincipalID: &principalID,
						EndDate:     &endDate,
					},
					usage.Usage{
						StartDate:   &startDatePlusOne,
						PrincipalID: &principalID,
						EndDate:     &endDatePlusOne,
					},
				},
				err: nil,
			},
			exp: response{
				data: &usage.Usages{
					usage.Usage{
						StartDate:   &startDate,
						PrincipalID: &principalID,
						EndDate:     &endDate,
					},
					usage.Usage{
						StartDate:   &startDatePlusOne,
						PrincipalID: &principalID,
						EndDate:     &endDatePlusOne,
					},
				},
				err: nil,
			},
		},
		{
			name: "internal error",
			inputData: usage.Usage{
				AccountID: &accountID,
			},
			ret: response{
				data: nil,
				err:  errors.NewInternalServer("failure", fmt.Errorf("original error")),
			},
			exp: response{
				data: nil,
				err:  errors.NewInternalServer("failure", fmt.Errorf("original error")),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mocksRWD := &mocks.ReaderWriter{}
			mocksRWD.On("List", mock.AnythingOfType("*usage.Usage")).Return(tt.ret.data, tt.ret.err)

			usageSvc := usage.NewService(
				usage.NewServiceInput{
					DataSvc: mocksRWD,
				},
			)

			usgs, err := usageSvc.List(&tt.inputData)
			assert.True(t, errors.Is(err, tt.exp.err), "actual error %q doesn't match expected error %q", err, tt.exp.err)
			assert.Equal(t, tt.exp.data, usgs)
		})
	}

}

func TestCreate(t *testing.T) {

	var costAmount float64 = 100.00
	var costCurrency string = "USD"
	var timeToLive int64 = time.Now().AddDate(0, 0, 30).Unix()
	type response struct {
		data *usage.Usage
		err  error
	}

	tests := []struct {
		name        string
		req         *usage.Usage
		exp         response
		getResponse response
		writeErr    error
	}{
		{
			name: "should create",
			req: &usage.Usage{
				StartDate:    &startDate,
				PrincipalID:  &principalID,
				AccountID:    &accountID,
				EndDate:      &endDate,
				CostCurrency: &costCurrency,
				CostAmount:   &costAmount,
				TimeToLive:   &timeToLive,
			},
			exp: response{
				data: &usage.Usage{
					StartDate:    &startDate,
					PrincipalID:  &principalID,
					AccountID:    &accountID,
					EndDate:      &endDate,
					CostCurrency: &costCurrency,
					CostAmount:   &costAmount,
					TimeToLive:   &timeToLive,
				},
				err: nil,
			},
			getResponse: response{
				data: nil,
				err:  errors.NewNotFound("usage", "1581033600-principal"),
			},
			writeErr: nil,
		},
		{
			name: "should fail on usages already exists",
			req: &usage.Usage{
				StartDate:    &startDate,
				PrincipalID:  &principalID,
				AccountID:    &accountID,
				EndDate:      &endDate,
				CostCurrency: &costCurrency,
				CostAmount:   &costAmount,
				TimeToLive:   &timeToLive,
			},
			exp: response{
				data: nil,
				err:  errors.NewAlreadyExists("usage", "1581033600-principal"),
			},
			getResponse: response{
				data: &usage.Usage{
					StartDate:   &startDate,
					PrincipalID: &principalID,
				},
				err: nil,
			},
		},
		{
			name: "should fail on get error",
			req: &usage.Usage{
				StartDate:    &startDate,
				PrincipalID:  &principalID,
				AccountID:    &accountID,
				EndDate:      &endDate,
				CostCurrency: &costCurrency,
				CostAmount:   &costAmount,
				TimeToLive:   &timeToLive,
			},
			exp: response{
				data: nil,
				err:  errors.NewInternalServer("error", nil),
			},
			getResponse: response{
				data: nil,
				err:  errors.NewInternalServer("error", nil),
			},
		},
		{
			name: "should fail on save",
			req: &usage.Usage{
				StartDate:    &startDate,
				PrincipalID:  &principalID,
				AccountID:    &accountID,
				EndDate:      &endDate,
				CostCurrency: &costCurrency,
				CostAmount:   &costAmount,
				TimeToLive:   &timeToLive,
			},
			exp: response{
				data: nil,
				err:  errors.NewInternalServer("error", nil),
			},
			getResponse: response{
				data: nil,
				err:  errors.NewNotFound("usage", "1581033600-principal"),
			},
			writeErr: errors.NewInternalServer("error", nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mocksRwd := &mocks.ReaderWriter{}

			mocksRwd.On("Get", *tt.req.StartDate, *tt.req.PrincipalID).Return(tt.getResponse.data, tt.getResponse.err)
			mocksRwd.On("Write", mock.AnythingOfType("*usage.Usage")).Return(tt.writeErr)

			usageSvc := usage.NewService(
				usage.NewServiceInput{
					DataSvc: mocksRwd,
				},
			)

			result, err := usageSvc.Create(tt.req)

			assert.Truef(t, errors.Is(err, tt.exp.err), "actual error %q doesn't match expected error %q", err, tt.exp.err)
			assert.Equal(t, tt.exp.data, result)

		})
	}
}
