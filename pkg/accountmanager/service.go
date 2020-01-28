package accountmanager

import (
	"github.com/Optum/dce/pkg/account"
	"github.com/Optum/dce/pkg/arn"
	"github.com/Optum/dce/pkg/common"
	"github.com/Optum/dce/pkg/errors"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts/stsiface"
	validation "github.com/go-ozzo/ozzo-validation"
)

// Service manages account resources
type Service struct {
	client   clienter
	storager common.Storager
}

// ValidateAccess creates a new Account instance
func (s *Service) ValidateAccess(role *arn.ARN) error {
	err := validation.Validate(role,
		validation.NotNil,
		validation.By(isAssumable(s.client)))
	if err != nil {
		return errors.NewValidation("account", err)
	}
	return nil
}

// MergePrincipalAccess creates roles, policies and update them as needed
func (s *Service) MergePrincipalAccess(account *account.Account) error {
	err := validation.ValidateStruct(account,
		validation.Field(&account.AdminRoleArn, validation.NotNil),
		validation.Field(&account.PrincipalRoleArn, validation.NotNil),
	)
	if err != nil {
		return errors.NewValidation("account", err)
	}

	iamSvc := s.client.IAM(account.AdminRoleArn)

	principalSvc := principalService{
		iamSvc:   iamSvc,
		storager: s.storager,
		account:  account,
	}

	err = principalSvc.MergeRole()
	if err != nil {
		return err
	}
	err = principalSvc.MergePolicy()
	if err != nil {
		return err
	}

	err = principalSvc.AttachRoleWithPolicy()
	if err != nil {
		return err
	}

	return nil
}

// NewServiceInput are the items needed to create a new service
type NewServiceInput struct {
	Session  *session.Session
	Sts      stsiface.STSAPI
	Storager common.Storager
}

// NewService creates a new account manager server
func NewService(input NewServiceInput) (*Service, error) {

	return &Service{
		client: &client{
			session: input.Session,
			sts:     input.Sts,
		},
		storager: input.Storager,
	}, nil

}
