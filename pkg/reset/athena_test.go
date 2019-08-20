package reset

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/athena"
	"github.com/aws/aws-sdk-go/service/athena/athenaiface"
	"github.com/stretchr/testify/assert"
)

type mockAthenaReset struct {
	athenaiface.AthenaAPI
}

// ListWorkGroups implemenation
func (athenaReset mockAthenaReset) ListWorkGroups(input *athena.ListWorkGroupsInput) (*athena.ListWorkGroupsOutput, error) {
	wg1 := &athena.WorkGroupSummary{
		Name: aws.String("wg1"),
	}
	mockOutput := &athena.ListWorkGroupsOutput{
		WorkGroups: []*athena.WorkGroupSummary{wg1},
	}
	return mockOutput, nil
}

// ListNamedQueries implemenation
func (athenaReset mockAthenaReset) ListNamedQueries(input *athena.ListNamedQueriesInput) (*athena.ListNamedQueriesOutput, error) {
	mockOutput := &athena.ListNamedQueriesOutput{
		NamedQueryIds: []*string{aws.String("test-query-1"), aws.String("test-query-2")},
	}
	return mockOutput, nil
}

// DeleteWorkGroup implemenation
func (athenaReset mockAthenaReset) DeleteWorkGroup(input *athena.DeleteWorkGroupInput) (*athena.DeleteWorkGroupOutput, error) {
	mockOutput := &athena.DeleteWorkGroupOutput{}
	return mockOutput, nil
}

// DeleteNamedQuery implemenation
func (athenaReset mockAthenaReset) DeleteNamedQuery(input *athena.DeleteNamedQueryInput) (*athena.DeleteNamedQueryOutput, error) {
	mockOutput := &athena.DeleteNamedQueryOutput{}
	return mockOutput, nil
}

func TestDeleteAthenaResources(t *testing.T) {
	mockAthena := new(mockAthenaReset)
	err := DeleteAthenaResources(mockAthena)
	assert.Nil(t, err, "There should be no errors")
}
