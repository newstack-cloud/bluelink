package core_test

import (
	"testing"
	"time"

	"github.com/newstack-cloud/bluelink/apps/deploy-engine/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/objectstore"
	"github.com/stretchr/testify/suite"
)

type ObjectStoreConfigSuite struct {
	suite.Suite
}

// The engine states its default as a literal, so this keeps it from drifting from the
// library default it is meant to match.
func (s *ObjectStoreConfigSuite) Test_the_default_cache_max_age_matches_the_library_default() {
	config, err := core.LoadConfig()
	s.Require().NoError(err)

	s.Assert().Equal(
		objectstore.DefaultCacheMaxAge,
		time.Duration(config.State.ObjectStore.CacheMaxAgeMS)*time.Millisecond,
	)
	s.Assert().False(config.State.ObjectStore.DisableCacheRevalidation)
}

func (s *ObjectStoreConfigSuite) Test_cache_revalidation_is_configured_from_the_environment() {
	s.T().Setenv("BLUELINK_DEPLOY_ENGINE_STATE_OBJECTSTORE_CACHE_MAX_AGE_MS", "5000")
	s.T().Setenv("BLUELINK_DEPLOY_ENGINE_STATE_OBJECTSTORE_DISABLE_CACHE_REVALIDATION", "true")

	config, err := core.LoadConfig()
	s.Require().NoError(err)

	s.Assert().Equal(5000, config.State.ObjectStore.CacheMaxAgeMS)
	s.Assert().True(config.State.ObjectStore.DisableCacheRevalidation)
}

func TestObjectStoreConfigSuite(t *testing.T) {
	suite.Run(t, new(ObjectStoreConfigSuite))
}
