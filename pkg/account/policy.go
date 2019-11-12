package account

import ()

// UpdatePrincipalPolicyHash updates hash representing the
// current version of the Principal IAM Policy applied to the account
func (a *Account) UpdatePrincipalPolicyHash(nextHash string, input Writer) error {

	//policyManager, err := rolemanager.NewPolicyManager(a)
	//policyManager.MergePolicy(&rolemanager.MergePolicyInput{})

	a.data.PrincipalPolicyHash = "test"
	//conditionBuilder := expression.ConditionBuilder{}
	//conditionBuilder = expression.Name("PrincipalPolicyHash").Equal(expression.Value(a.PrincipalPolicyHash))

	err := a.Update(input)
	return err
}
