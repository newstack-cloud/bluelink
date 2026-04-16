package links

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/resourcehelpers"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/newstack-cloud/bluelink/libs/common/testhelpers"

	. "gopkg.in/check.v1"
)

type DeclaredLinkGraphTestSuite struct {
	resourceRegistry  resourcehelpers.Registry
	blueprintFixtures map[string]*schema.Blueprint
}

var _ = Suite(&DeclaredLinkGraphTestSuite{})

func (s *DeclaredLinkGraphTestSuite) SetUpSuite(c *C) {
	awsProvider := newTestAWSProvider()
	resourceProviders := map[string]provider.Provider{
		"aws/sqs/queue":       awsProvider,
		"aws/lambda/function": awsProvider,
		"aws/dynamodb/table":  awsProvider,
	}
	celerityTransformer := newTestCelerityTransformer()
	transformers := map[string]transform.SpecTransformer{
		"celerity-2026-04-01": celerityTransformer,
	}
	s.resourceRegistry = resourcehelpers.NewRegistry(
		resourceProviders,
		transformers,
		10*time.Millisecond,
		/* stateContainer */ nil,
		/* params */ nil,
	)
	blueprintFixtures, err := s.loadBlueprintFixtures()
	if err != nil {
		c.Fatalf("failed to load blueprint fixtures: %v", err)
	}
	s.blueprintFixtures = blueprintFixtures
}

func (s *DeclaredLinkGraphTestSuite) loadBlueprintFixtures() (map[string]*schema.Blueprint, error) {
	fixtureFiles := map[string]string{
		"declared-link-graph-1": "__testdata/declared-link-graph-1.blueprint.yml",
	}

	blueprintFixtures := make(map[string]*schema.Blueprint)
	for name, filePath := range fixtureFiles {
		bp, err := schema.Load(filePath, schema.YAMLSpecFormat)
		if err != nil {
			return nil, err
		}
		blueprintFixtures[name] = bp
	}

	return blueprintFixtures, nil
}

func (s *DeclaredLinkGraphTestSuite) Test_enumerates_declared_links(c *C) {
	ctx := context.Background()
	linkGraph, err := EnumerateDeclaredLinks(
		ctx,
		s.blueprintFixtures["declared-link-graph-1"],
		s.resourceRegistry,
	)
	c.Assert(err, IsNil)

	edgesFromOrdersAPI := linkGraph.EdgesFrom("ordersApi")
	edgesFromOrdersQueue := linkGraph.EdgesFrom("ordersQueue")
	edgesFromOrphanHandler := linkGraph.EdgesFrom("orphanHandler")

	edgesToOrdersDatastore := linkGraph.EdgesTo("ordersDatastore")
	edgesToOrdersQueue := linkGraph.EdgesTo("ordersQueue")
	edgesToOrderWorker := linkGraph.EdgesTo("orderWorker")
	edgesToEmailQueue := linkGraph.EdgesTo("emailQueue")
	edgesToAuditTable := linkGraph.EdgesTo("auditTable")
	edgesToUnusedTable := linkGraph.EdgesTo("unusedTable")

	edgeGroups := map[string][]*ResolvedLink{
		"source::ordersApi":       edgesFromOrdersAPI,
		"source::ordersQueue":     edgesFromOrdersQueue,
		"source::orphanHandler":   edgesFromOrphanHandler,
		"target::ordersDatastore": edgesToOrdersDatastore,
		"target::ordersQueue":     edgesToOrdersQueue,
		"target::orderWorker":     edgesToOrderWorker,
		"target::emailQueue":      edgesToEmailQueue,
		"target::auditTable":      edgesToAuditTable,
		"target::unusedTable":     edgesToUnusedTable,
		"all":                     normaliseEdgesForSnapshot(linkGraph.Edges()),
	}
	err = testhelpers.Snapshot(normaliseEdgeGroupsForSnapshot(edgeGroups))
	if err != nil {
		c.Error(err)
	}
}

type groupedResolvedLinks struct {
	GroupName string
	Edges     []*ResolvedLink
}

func normaliseEdgeGroupsForSnapshot(edgeGroups map[string][]*ResolvedLink) []*groupedResolvedLinks {
	normalised := make([]*groupedResolvedLinks, 0, len(edgeGroups))
	for groupName, edgeGroup := range edgeGroups {
		normalised = append(normalised, &groupedResolvedLinks{
			GroupName: groupName,
			Edges:     normaliseEdgesForSnapshot(edgeGroup),
		})
	}

	slices.SortStableFunc(normalised, func(a *groupedResolvedLinks, b *groupedResolvedLinks) int {
		return strings.Compare(a.GroupName, b.GroupName)
	})

	return normalised
}

func normaliseEdgesForSnapshot(edges []*ResolvedLink) []*ResolvedLink {
	normalised := make([]*ResolvedLink, len(edges))

	for i, edge := range edges {
		slices.Sort(edge.SelectorKeys)
		normalised[i] = &ResolvedLink{
			Source:       edge.Source,
			Target:       edge.Target,
			SourceType:   edge.SourceType,
			TargetType:   edge.TargetType,
			SelectorKeys: edge.SelectorKeys,
		}
	}

	slices.SortStableFunc(normalised, func(a *ResolvedLink, b *ResolvedLink) int {
		aKey := fmt.Sprintf("%s->%s", a.Source, a.Target)
		bKey := fmt.Sprintf("%s->%s", b.Source, b.Target)
		return strings.Compare(aKey, bKey)
	})

	return normalised
}
