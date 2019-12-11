package accountiface

import (
	"github.com/Optum/dce/pkg/account"
	"github.com/Optum/dce/pkg/model"
)

// Account Interface
type Account interface {
	ID() string
	Status() model.AccountStatus
	AdminRoleArn() string
	PrincipalRoleArn() string
	PrincipalPolicyHash() string
	Metadata() map[string]interface{}
	UpdateStatus(nextStatus model.AccountStatus, u account.Updater) error
	Update(d model.Account, u account.Updater, am account.Manager) error
	Delete(d account.Deleter) error
	OrphanAccount() error
	MarshalJSON() ([]byte, error)
}
