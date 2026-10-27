package objectstore_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/statestore"
	"github.com/stretchr/testify/suite"
)

type CleanupOperationsSuite struct {
	objectstoreSuite
}

// Each short-lived engine records one operation, so the window only holds if every engine
// sees the operations the others stored.
func (s *CleanupOperationsSuite) Test_the_rolling_window_holds_across_engines() {
	ctx := context.Background()
	total := statestore.MaxCleanupOperationsPerType + 5
	for i := range total {
		err := s.newContainer().CleanupOperations().Save(ctx, &manage.CleanupOperation{
			ID:          fmt.Sprintf("cleanup-op-%03d", i),
			CleanupType: manage.CleanupTypeEvents,
			Status:      manage.CleanupOperationStatusCompleted,
			StartedAt:   int64(1760000000 + i),
		})
		s.Require().NoError(err)
	}

	s.Assert().Len(s.objectKeys(testPrefix+"cleanup_operations/"), statestore.MaxCleanupOperationsPerType)

	latest, err := s.newContainer().CleanupOperations().GetLatestByType(ctx, manage.CleanupTypeEvents)
	s.Require().NoError(err)
	s.Assert().Equal(fmt.Sprintf("cleanup-op-%03d", total-1), latest.ID)
}

func TestCleanupOperationsSuite(t *testing.T) {
	suite.Run(t, new(CleanupOperationsSuite))
}
