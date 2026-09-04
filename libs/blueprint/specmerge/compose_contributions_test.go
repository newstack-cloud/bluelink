package specmerge

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/stretchr/testify/suite"
)

type ComposeContributionsTestSuite struct {
	suite.Suite
}

// The case the rebuild exists for. Deploying one link has to leave every other link's
// contribution to the same resource in place, or the update states a spec without them
// and a converging provider takes them away.
func (s *ComposeContributionsTestSuite) Test_keeps_the_contributions_of_a_link_that_did_not_run() {
	result, err := ComposeResourceContributions(
		core.MappingNodeFields(),
		"ordersRole",
		&ContributionInputs{
			Produced: []LinkResourceContribution{
				s.appended("saveOrderFunction::ordersTable", "spec.policies", "dynamodb:PutItem"),
			},
			StoredLinks: []state.LinkState{
				s.storedAppend("archiveFunction::appQueue", "spec.policies", "sqs:SendMessage"),
			},
		},
	)

	s.Require().NoError(err)
	s.Assert().Empty(result.Unresolved)
	s.Assert().Equal(
		[]string{"sqs:SendMessage", "dynamodb:PutItem"},
		s.itemsAt(result.Spec, "$.policies"),
		"the link that did not run keeps its statement",
	)
}

// A link that ran states everything it needs. A field it contributed last time and did not
// contribute now has been withdrawn, so reading its stored contributions as well would put
// the withdrawn value straight back.
func (s *ComposeContributionsTestSuite) Test_a_link_that_ran_supersedes_what_it_recorded_before() {
	stored := s.storedSet(
		"saveOrderFunction::ordersTable",
		"saveOrderFunction::spec.environment.variables.TABLE_NAME",
		"orders-old",
	)
	stored.ResourceDataMappings["saveOrderFunction::spec.environment.variables.TABLE_REGION"] =
		"link.region"
	stored.Data["link"].Fields["region"] = core.MappingNodeFromString("eu-west-1")

	result, err := ComposeResourceContributions(
		core.MappingNodeFields(),
		"saveOrderFunction",
		&ContributionInputs{
			Produced: []LinkResourceContribution{
				s.set(
					"saveOrderFunction::ordersTable",
					"spec.environment.variables.TABLE_NAME",
					"orders-new",
				),
			},
			StoredLinks:         []state.LinkState{stored},
			SupersededLinkNames: []string{"saveOrderFunction::ordersTable"},
		},
	)

	s.Require().NoError(err)
	s.Assert().Equal(
		"orders-new",
		s.valueAt(result.Spec, "$.environment.variables.TABLE_NAME"),
	)
	s.Assert().Nil(
		result.Spec.Fields["environment"].Fields["variables"].Fields["TABLE_REGION"],
		"a field the link no longer contributes has been withdrawn rather than carried over",
	)
}

func (s *ComposeContributionsTestSuite) Test_does_not_read_back_what_a_superseded_link_recorded() {
	stored := s.storedSet(
		"saveOrderFunction::ordersTable",
		"saveOrderFunction::spec.tableName",
		"orders",
	)

	result, err := ComposeResourceContributions(
		core.MappingNodeFields(),
		"saveOrderFunction",
		&ContributionInputs{
			StoredLinks:         []state.LinkState{stored},
			SupersededLinkNames: []string{"saveOrderFunction::ordersTable"},
		},
	)

	s.Require().NoError(err)
	s.Assert().Nil(
		result.Spec.Fields["tableName"],
		"the contribution it withdrew is not put back",
	)
}

func (s *ComposeContributionsTestSuite) Test_reads_back_what_a_link_that_is_not_superseded_recorded() {
	stored := s.storedSet(
		"saveOrderFunction::ordersTable",
		"saveOrderFunction::spec.tableName",
		"orders",
	)

	result, err := ComposeResourceContributions(
		core.MappingNodeFields(),
		"saveOrderFunction",
		&ContributionInputs{
			StoredLinks: []state.LinkState{stored},
		},
	)

	s.Require().NoError(err)
	s.Assert().Equal("orders", s.valueAt(result.Spec, "$.tableName"))
}

func (s *ComposeContributionsTestSuite) Test_drops_the_contributions_of_a_removed_link() {
	result, err := ComposeResourceContributions(
		core.MappingNodeFields(),
		"ordersRole",
		&ContributionInputs{
			Produced: nil,
			StoredLinks: []state.LinkState{
				s.storedAppend("archiveFunction::appQueue", "spec.policies", "sqs:SendMessage"),
			},
			RemovedLinkNames: []string{"archiveFunction::appQueue"},
		},
	)

	s.Require().NoError(err)
	s.Assert().Nil(
		result.Spec.Fields["policies"],
		"a removed link's statement goes with it, without the framework having to find it",
	)
}

// The stream a table may still be needed for, and the auth token live clients hold. The
// link is gone and never runs again, so what it recorded is the only account of the choice.
func (s *ComposeContributionsTestSuite) Test_keeps_a_removed_links_contribution_marked_to_outlive_it() {
	stored := s.storedSet("ordersTable::archiveFunction", "ordersTable::spec.streamEnabled", "true")
	stored.ContributionRecords["ordersTable::spec.streamEnabled"] = state.ContributionRecord{
		Action:          int(provider.ContributionActionSet),
		RetainOnRemoval: true,
	}

	result, err := ComposeResourceContributions(
		core.MappingNodeFields(),
		"ordersTable",
		&ContributionInputs{
			Produced:         nil,
			StoredLinks:      []state.LinkState{stored},
			RemovedLinkNames: []string{"ordersTable::archiveFunction"},
		},
	)

	s.Require().NoError(err)
	s.Assert().Equal("true", s.valueAt(result.Spec, "$.streamEnabled"))
}

// An appended element's position must not depend on whether its link ran, or the element
// moves between deployments and staging reports it as modified when nothing changed.
func (s *ComposeContributionsTestSuite) Test_orders_by_link_rather_than_by_which_links_ran() {
	first := s.appended("aFunction::aTable", "spec.policies", "a-statement")
	second := s.appended("bFunction::bTable", "spec.policies", "b-statement")

	bothRan, err := ComposeResourceContributions(
		core.MappingNodeFields(),
		"ordersRole",
		&ContributionInputs{
			Produced:    []LinkResourceContribution{first, second},
			StoredLinks: nil,
		},
	)
	s.Require().NoError(err)

	// The same two contributions, with the alphabetically first link supplying its own
	// from state instead of having run.
	onlySecondRan, err := ComposeResourceContributions(
		core.MappingNodeFields(),
		"ordersRole",
		&ContributionInputs{
			Produced: []LinkResourceContribution{second},
			StoredLinks: []state.LinkState{
				s.storedAppend("aFunction::aTable", "spec.policies", "a-statement"),
			},
		},
	)
	s.Require().NoError(err)

	s.Assert().Equal(
		s.itemsAt(bothRan.Spec, "$.policies"),
		s.itemsAt(onlySecondRan.Spec, "$.policies"),
	)
}

// A mapping saved before contribution records were kept has no record beside it, and the
// replacement it would have performed is what it still performs.
func (s *ComposeContributionsTestSuite) Test_applies_a_stored_contribution_with_no_record_as_a_replacement() {
	stored := s.storedSet("saveOrderFunction::ordersTable", "saveOrderFunction::spec.tableName", "orders")
	stored.ContributionRecords = nil

	result, err := ComposeResourceContributions(
		core.MappingNodeFields(),
		"saveOrderFunction",
		&ContributionInputs{
			Produced:    nil,
			StoredLinks: []state.LinkState{stored},
		},
	)

	s.Require().NoError(err)
	s.Assert().Equal("orders", s.valueAt(result.Spec, "$.tableName"))
}

// A contribution the link's data no longer holds a value for has to be reported rather
// than passed over. Composing without it produces a spec missing the field, which deploys
// as the contribution having been withdrawn by nobody.
func (s *ComposeContributionsTestSuite) Test_reports_a_stored_contribution_whose_value_has_gone() {
	stored := s.storedSet(
		"saveOrderFunction::ordersTable",
		"saveOrderFunction::spec.tableName",
		"orders",
	)
	// The mapping still points into the link's data, and the data no longer holds it.
	stored.Data = map[string]*core.MappingNode{}

	result, err := ComposeResourceContributions(
		core.MappingNodeFields(),
		"saveOrderFunction",
		&ContributionInputs{
			StoredLinks: []state.LinkState{stored},
		},
	)

	s.Require().NoError(err)
	s.Require().Len(result.Unresolved, 1)
	s.Assert().Equal("saveOrderFunction::ordersTable", result.Unresolved[0].LinkName)
	s.Assert().Equal("spec.tableName", result.Unresolved[0].ResourceFieldPath)
	s.Assert().Equal(
		"link.value",
		result.Unresolved[0].LinkDataPath,
		"the path the value was expected at says whether the data or the resource is at fault",
	)
}

func (s *ComposeContributionsTestSuite) set(
	linkName string,
	fieldPath string,
	value string,
) LinkResourceContribution {
	return s.contribution(linkName, fieldPath, value, provider.ContributionActionSet)
}

func (s *ComposeContributionsTestSuite) appended(
	linkName string,
	fieldPath string,
	value string,
) LinkResourceContribution {
	return s.contribution(linkName, fieldPath, value, provider.ContributionActionAppend)
}

func (s *ComposeContributionsTestSuite) contribution(
	linkName string,
	fieldPath string,
	value string,
	action provider.ContributionAction,
) LinkResourceContribution {
	resourceName := "ordersRole"
	if action == provider.ContributionActionSet {
		resourceName = "saveOrderFunction"
	}

	return LinkResourceContribution{
		LinkName: linkName,
		Contribution: &provider.ResourceContribution{
			ResourceName: resourceName,
			FieldPath:    fieldPath,
			Value:        core.MappingNodeFromString(value),
			Action:       action,
		},
	}
}

func (s *ComposeContributionsTestSuite) storedSet(
	linkName string,
	resourceFieldPath string,
	value string,
) state.LinkState {
	return s.stored(linkName, resourceFieldPath, value, provider.ContributionActionSet)
}

func (s *ComposeContributionsTestSuite) storedAppend(
	linkName string,
	fieldPath string,
	value string,
) state.LinkState {
	return s.stored(linkName, "ordersRole::"+fieldPath, value, provider.ContributionActionAppend)
}

// A link as state holds it: the value in its data, a mapping saying which resource field
// the value feeds, and a record saying how to apply it. The mapping is keyed by the full
// "{resourceName}::{fieldPath}" the caller gives, since a key in any other form names no
// resource and is passed over.
func (s *ComposeContributionsTestSuite) stored(
	linkName string,
	resourceFieldPath string,
	value string,
	action provider.ContributionAction,
) state.LinkState {
	s.Require().Contains(
		resourceFieldPath,
		"::",
		"a stored mapping key has to name the resource it targets",
	)

	return state.LinkState{
		Name: linkName,
		Data: map[string]*core.MappingNode{
			"link": core.MappingNodeFields(
				"value",
				core.MappingNodeFromString(value),
			),
		},
		ResourceDataMappings: map[string]string{
			resourceFieldPath: "link.value",
		},
		ContributionRecords: map[string]state.ContributionRecord{
			resourceFieldPath: {Action: int(action)},
		},
	}
}

func (s *ComposeContributionsTestSuite) valueAt(spec *core.MappingNode, path string) string {
	value, err := core.GetPathValue(path, spec, core.MappingNodeMaxTraverseDepth)
	s.Require().NoError(err)
	s.Require().NotNil(value, "nothing at %s", path)

	return core.StringValue(value)
}

func (s *ComposeContributionsTestSuite) itemsAt(spec *core.MappingNode, path string) []string {
	value, err := core.GetPathValue(path, spec, core.MappingNodeMaxTraverseDepth)
	s.Require().NoError(err)
	s.Require().NotNil(value, "nothing at %s", path)

	items := []string{}
	for _, item := range value.Items {
		items = append(items, core.StringValue(item))
	}

	return items
}

func TestComposeContributionsTestSuite(t *testing.T) {
	suite.Run(t, new(ComposeContributionsTestSuite))
}
