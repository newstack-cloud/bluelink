package objectstore_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/memfile"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/objectstore"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/objectstore/internal/mockservice"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/suite"
)

type equivalenceContainer interface {
	state.Container
	Changesets() manage.Changesets
	Validation() manage.Validation
	ReconciliationResults() manage.ReconciliationResults
	CleanupOperations() manage.CleanupOperations
}

// A backend opens a fresh engine's view of one persisted store. memfile reloads everything
// at startup, so a freshly opened memfile container is the reference for what a freshly
// opened objectstore container, which loads on demand, must return.
type equivalenceBackend interface {
	open() (equivalenceContainer, error)
}

type memfileBackend struct {
	fs afero.Fs
}

func (b *memfileBackend) open() (equivalenceContainer, error) {
	return memfile.LoadStateContainer("state", b.fs, core.NewNopLogger())
}

type objectstoreBackend struct {
	svc *mockservice.Service
}

func (b *objectstoreBackend) open() (equivalenceContainer, error) {
	return objectstore.LoadStateContainer(context.Background(), b.svc, testPrefix, core.NewNopLogger())
}

type equivalenceMutation struct {
	name   string
	mutate func(ctx context.Context, c equivalenceContainer) error
}

// Checks that an engine reading the bucket lazily sees what an engine that loaded all of
// it eagerly sees, and that mutations made by a fresh engine persist the same state.
//
// Each step opens a new container, so nothing a previous step cached can mask a missing
// load, every read and every mutation starts from what reached storage.
type LazyEquivalenceSuite struct {
	suite.Suite
	reference *memfileBackend
	lazy      *objectstoreBackend
}

func (s *LazyEquivalenceSuite) SetupTest() {
	s.reference = &memfileBackend{fs: afero.NewMemMapFs()}
	s.Require().NoError(s.reference.fs.MkdirAll("state", 0755))
	s.lazy = &objectstoreBackend{svc: mockservice.New()}

	s.seed(s.reference)
	s.seed(s.lazy)
}

// Each mutation runs against freshly seeded backends.
func (s *LazyEquivalenceSuite) SetupSubTest() {
	s.SetupTest()
}

func (s *LazyEquivalenceSuite) Test_a_fresh_engine_reads_the_same_state() {
	s.assertEquivalent()
}

func (s *LazyEquivalenceSuite) Test_a_fresh_engine_persists_the_same_instance_mutations() {
	s.runMutations(instanceMutations())
}

func (s *LazyEquivalenceSuite) Test_a_fresh_engine_persists_the_same_resource_mutations() {
	s.runMutations(resourceMutations())
}

func (s *LazyEquivalenceSuite) Test_a_fresh_engine_persists_the_same_link_mutations() {
	s.runMutations(linkMutations())
}

func (s *LazyEquivalenceSuite) Test_a_fresh_engine_persists_the_same_child_mutations() {
	s.runMutations(childMutations())
}

func (s *LazyEquivalenceSuite) Test_a_fresh_engine_persists_the_same_metadata_and_export_mutations() {
	s.runMutations(metadataAndExportMutations())
}

func (s *LazyEquivalenceSuite) Test_a_fresh_engine_persists_the_same_cleanup_operation_mutations() {
	s.runMutations(cleanupOperationMutations())
}

// Equivalence can't catch what both backends get wrong, so this is asserted directly, an
// instance saved with its resources and links in place must serve them from the same engine.
func (s *LazyEquivalenceSuite) Test_an_instance_saved_whole_serves_its_resources_and_links() {
	for _, backend := range []equivalenceBackend{s.reference, s.lazy} {
		c, err := backend.open()
		s.Require().NoError(err)
		ctx := context.Background()

		instance := eqInstance(
			eqNewChildID, eqNewChildName,
			[]*state.ResourceState{
				eqResource(eqNewChildID, eqNewResourceID, eqNewResource, "aws/sqs/queue"),
			},
			map[string]*state.InstanceState{},
		)
		instance.Links[eqNewLinkName] = &state.LinkState{
			LinkID:     eqNewLinkID,
			Name:       eqNewLinkName,
			InstanceID: eqNewChildID,
		}
		s.Require().NoError(c.Instances().Save(ctx, *instance))

		_, err = c.Resources().Get(ctx, eqNewResourceID)
		s.Assert().NoError(err, "%T resources.Get", backend)
		_, err = c.Links().Get(ctx, eqNewLinkID)
		s.Assert().NoError(err, "%T links.Get", backend)
	}
}

func (s *LazyEquivalenceSuite) Test_saving_an_existing_instance_under_a_new_name_is_rejected() {
	for _, backend := range []equivalenceBackend{s.reference, s.lazy} {
		c, err := backend.open()
		s.Require().NoError(err)
		ctx := context.Background()

		instance, err := c.Instances().Get(ctx, eqParentID)
		s.Require().NoError(err)
		instance.InstanceName = "RenamedParent"
		err = c.Instances().Save(ctx, instance)
		s.Assert().ErrorIs(err, state.ErrInstanceNameImmutable, "%T", backend)

		fresh, err := backend.open()
		s.Require().NoError(err)
		id, err := fresh.Instances().LookupIDByName(ctx, eqParentName)
		s.Require().NoError(err, "%T", backend)
		s.Assert().Equal(eqParentID, id, "%T", backend)
	}
}

func (s *LazyEquivalenceSuite) runMutations(mutations []equivalenceMutation) {
	for _, mutation := range mutations {
		s.Run(mutation.name, func() {
			referenceErr := s.mutate(s.reference, mutation)
			lazyErr := s.mutate(s.lazy, mutation)
			s.Require().Equal(errorText(referenceErr), errorText(lazyErr), "mutation result")
			s.assertEquivalent()
		})
	}
}

func (s *LazyEquivalenceSuite) seed(backend equivalenceBackend) {
	c, err := backend.open()
	s.Require().NoError(err)
	s.Require().NoError(seedEquivalenceState(context.Background(), c))
}

func (s *LazyEquivalenceSuite) mutate(backend equivalenceBackend, mutation equivalenceMutation) error {
	c, err := backend.open()
	s.Require().NoError(err)
	return mutation.mutate(context.Background(), c)
}

func (s *LazyEquivalenceSuite) assertEquivalent() {
	reference := s.view(s.reference)
	lazy := s.view(s.lazy)

	for key, expected := range reference {
		s.Assert().Equal(expected, lazy[key], key)
	}
}

// Rendered results of every read, keyed by the read. Errors are kept as text so that "not
// found" on one backend and a value on the other is a difference too.
//
// Each read opens its own container, because one read loading an instance would otherwise
// hide a later read that fails to load it.
func (s *LazyEquivalenceSuite) view(backend equivalenceBackend) map[string]string {
	view := map[string]string{}
	for _, read := range equivalenceReads() {
		c, err := backend.open()
		s.Require().NoError(err)
		value, err := read.read(context.Background(), c)
		view[read.key] = renderRead(value, err)
	}
	return view
}

func renderRead(value any, err error) string {
	if err != nil {
		return "error: " + err.Error()
	}
	data, err := json.Marshal(value)
	if err != nil {
		return "marshal error: " + err.Error()
	}
	return string(data)
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func TestLazyEquivalenceSuite(t *testing.T) {
	suite.Run(t, new(LazyEquivalenceSuite))
}
