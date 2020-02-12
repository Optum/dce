package data

import (
	"testing"

	"github.com/Optum/dce/pkg/account"
	"github.com/Optum/dce/pkg/arn"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
	"github.com/stretchr/testify/assert"
)

func ptrString(s string) *string {
	ptrS := s
	return &ptrS
}

func ptrInt64(i int64) *int64 {
	ptrI := i
	return &ptrI
}

func TestHelpersBuildFilter(t *testing.T) {

	tests := []struct {
		name   string
		query  string
		i      interface{}
		result expression.ConditionBuilder
		err    error
	}{
		{
			name: "buildfilter",
			i: &account.Account{
				ID: ptrString("1"),
			},
			result: expression.Name("Id").Equal(expression.Value("1")),
		},
		{
			name: "multipleFilters",
			i: &account.Account{
				ID:           ptrString("1"),
				AdminRoleArn: arn.New("aws", "iam", "", "123456789012", "role/AdminRoleArn"),
			},
			result: expression.And(
				expression.Name("Id").Equal(expression.Value("1")),
				expression.Name("AdminRoleArn").Equal(expression.Value("arn:aws:iam::123456789012:role/AdminRoleArn")),
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, o := getFiltersFromStruct(tt.i, nil)
			assert.Equal(t, &tt.result, o)
		})
	}

}
