package tests

import (
	"testing"

	"github.com/Optum/dce/tests/testutils"
)

func givenEmptySystem(t *testing.T) {
	truncateDBTables(t)
	testutils.GivenSqsIsEmpty(t, sqsSvc, sqsResetURL)
	testutils.GivenCodeBuildIsEmpty(t, codeBuildSvc, codeBuildResetName)
}
