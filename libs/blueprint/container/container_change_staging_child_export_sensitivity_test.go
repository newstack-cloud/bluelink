package container

import (
	"context"
	"os"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal/memstate"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/providerhelpers"
	"github.com/newstack-cloud/bluelink/libs/blueprint/refgraph"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/stretchr/testify/suite"
)

// A child blueprint resolves against its own sensitivity record, so the only thing that can
// mark a parent field reading one of its exports is the answer the child reported. That
// answer travels child staging -> the child's export change -> the parent's child export
// cache -> the parent's own change set, and a break anywhere along it is silent: the parent
// simply reports a secret as plain.
//
// Exercised through StageChanges rather than by seeding the cache, so the whole chain is
// covered rather than each half.
type ChildExportSensitivityTestSuite struct {
	blueprintContainer BlueprintContainer
	suite.Suite
}

func (s *ChildExportSensitivityTestSuite) SetupSuite() {
	stateContainer := memstate.NewMemoryStateContainer()
	providers := map[string]provider.Provider{
		"aws": newTestAWSProvider(
			/* alwaysStabilise */ false,
			/* skipRetryFailuresForLinkNames */ []string{},
			stateContainer,
		),
		"core": providerhelpers.NewCoreProvider(
			stateContainer.Links(),
			core.BlueprintInstanceIDFromContext,
			os.Getwd,
			provider.NewFileSourceRegistry(),
			core.SystemClock{},
		),
	}
	loader := NewDefaultLoader(
		providers,
		map[string]transform.SpecTransformer{},
		stateContainer,
		newFSChildResolver(),
		WithLoaderTransformSpec(false),
		WithLoaderValidateRuntimeValues(true),
		WithLoaderRefChainCollectorFactory(refgraph.NewRefChainCollector),
		WithLoaderLogger(core.NewNopLogger()),
	)

	blueprintContainer, err := loader.Load(
		context.Background(),
		"__testdata/container/change-staging/blueprint-child-export-sensitivity.yml",
		childExportSensitivityParams(),
	)
	s.Require().NoError(err)
	s.blueprintContainer = blueprintContainer
}

func (s *ChildExportSensitivityTestSuite) Test_a_parent_field_reading_a_childs_secret_export_is_marked() {
	parentChanges := s.stageParentChanges()

	tableName, hasTableName := parentChanges["spec.tableName"]
	s.Require().True(hasTableName, "spec.tableName is not among the parent's new fields")
	s.Assert().True(
		tableName.Sensitive,
		"the field reading the child's secret export was reported as plain",
	)

	region, hasRegion := parentChanges["spec.region"]
	s.Require().True(hasRegion, "spec.region is not among the parent's new fields")
	s.Assert().False(
		region.Sensitive,
		"the field reading the child's plain export was marked",
	)
}

func (s *ChildExportSensitivityTestSuite) stageParentChanges() map[string]provider.FieldChange {
	channels := createChangeStagingChannels()

	err := s.blueprintContainer.StageChanges(
		context.Background(),
		&StageChangesInput{InstanceID: "child-export-sensitivity-instance"},
		channels,
		childExportSensitivityParams(),
	)
	s.Require().NoError(err)

	fullChangeSet := (*changes.BlueprintChanges)(nil)
	for err == nil && fullChangeSet == nil {
		select {
		case <-channels.ResourceChangesChan:
		case <-channels.ChildChangesChan:
		case <-channels.LinkChangesChan:
		case changeSet := <-channels.CompleteChan:
			fullChangeSet = &changeSet
		case err = <-channels.ErrChan:
		}
	}
	s.Require().NoError(err)
	s.Require().NotNil(fullChangeSet)

	parentTable, hasParentTable := fullChangeSet.NewResources["parentTable"]
	s.Require().True(hasParentTable, "parentTable is not among the staged resources")

	byPath := map[string]provider.FieldChange{}
	for _, fieldChange := range parentTable.NewFields {
		byPath[fieldChange.FieldPath] = fieldChange
	}

	return byPath
}

func childExportSensitivityParams() core.BlueprintParams {
	region := "eu-west-2"

	return core.NewDefaultParams(
		map[string]map[string]*core.ScalarValue{},
		map[string]map[string]*core.ScalarValue{},
		map[string]*core.ScalarValue{},
		map[string]*core.ScalarValue{
			"region": {StringValue: &region},
		},
	)
}

func TestChildExportSensitivityTestSuite(t *testing.T) {
	suite.Run(t, new(ChildExportSensitivityTestSuite))
}
