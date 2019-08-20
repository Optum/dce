package reset

import (
	"log"

	"github.com/aws/aws-sdk-go/service/athena"
	"github.com/aws/aws-sdk-go/service/athena/athenaiface"
)

// AthenaService interface
type AthenaService interface {
	ListWorkGroups(input *athena.ListWorkGroupsInput) (*athena.ListWorkGroupsOutput, error)
	ListNamedQueries(input *athena.ListNamedQueriesInput) (*athena.ListNamedQueriesOutput, error)
	DeleteNamedQuery(input *athena.DeleteNamedQueryInput) (*athena.DeleteNamedQueryOutput, error)
	DeleteWorkGroup(input *athena.DeleteWorkGroupInput) (*athena.DeleteWorkGroupOutput, error)
}

// AthenaReset defines a concrete implementation of the above Service interface
type AthenaReset struct {
	Client athenaiface.AthenaAPI
}

// ListWorkGroups implemenation
func (athenaReset AthenaReset) ListWorkGroups(input *athena.ListWorkGroupsInput) (*athena.ListWorkGroupsOutput, error) {
	return athenaReset.Client.ListWorkGroups(input)
}

// ListNamedQueries implemenation
func (athenaReset AthenaReset) ListNamedQueries(input *athena.ListNamedQueriesInput) (*athena.ListNamedQueriesOutput, error) {
	return athenaReset.Client.ListNamedQueries(input)
}

// DeleteWorkGroup implemenation
func (athenaReset AthenaReset) DeleteWorkGroup(input *athena.DeleteWorkGroupInput) (*athena.DeleteWorkGroupOutput, error) {
	return athenaReset.Client.DeleteWorkGroup(input)
}

// DeleteNamedQuery implemenation
func (athenaReset AthenaReset) DeleteNamedQuery(input *athena.DeleteNamedQueryInput) (*athena.DeleteNamedQueryOutput, error) {
	return athenaReset.Client.DeleteNamedQuery(input)
}

// DeleteAthenaResources deletes all aethna resources in the current aws session
func DeleteAthenaResources(athenaSvc AthenaService) error {

	var maxResult int64 = 50
	// Delete all workgroups
	listWorkGroupsInput := &athena.ListWorkGroupsInput{
		MaxResults: &maxResult,
	}
	listWorkGroupsOutput, err := athenaSvc.ListWorkGroups(listWorkGroupsInput)
	if err != nil {
		return err
	}

	for _, workGroup := range listWorkGroupsOutput.WorkGroups {
		isDelete := true
		log.Printf("Starting Athena workgroup list %v", workGroup)
		if *workGroup.Name == "primary" {
			continue
		}
		deleteWorkGroupInput := &athena.DeleteWorkGroupInput{
			RecursiveDeleteOption: &isDelete,
			WorkGroup:             workGroup.Name,
		}
		log.Printf("Starting Athena workgroup delete %v", workGroup)
		_, err := athenaSvc.DeleteWorkGroup(deleteWorkGroupInput)
		if err != nil {
			log.Printf("Athena workgroup delete error: %v", err)
			return err
		}
	}

	// Delete all namedqueries
	listNamedQueriesInput := &athena.ListNamedQueriesInput{}
	listNamedQueriesOutput, err := athenaSvc.ListNamedQueries(listNamedQueriesInput)
	if err != nil {
		return err
	}

	for _, namedQuery := range listNamedQueriesOutput.NamedQueryIds {
		log.Printf("Starting Athena namedquery delete %v", *namedQuery)
		deleteNamedQueryInput := &athena.DeleteNamedQueryInput{
			NamedQueryId: namedQuery,
		}
		_, err := athenaSvc.DeleteNamedQuery(deleteNamedQueryInput)
		if err != nil {
			return err
		}
	}

	return nil
}
