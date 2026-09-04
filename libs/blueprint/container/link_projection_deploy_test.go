package container

import (
	"context"
	"fmt"
	"maps"
	"sync"
	"testing"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal/memstate"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/refgraph"
	"github.com/newstack-cloud/bluelink/libs/blueprint/specmerge"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/stretchr/testify/suite"
)

type LinkProjectionDeployTestSuite struct {
	suite.Suite
}

// A resource that a link has written to must be deployed with what the link wrote, when
// it is updated for a reason that has nothing to do with the link.
//
// A deploy applies a resource's complete intended state rather than a patch of the fields
// that changed, so a spec built from the blueprint alone removes everything a link
// contributed. This previous behaviour would remove link contributions
// such as the environment variables an access link populated, the networking a VPC
// link configured, the policy statements granted to a shared execution role. Nothing puts
// them back, because a link whose own inputs have not changed is staged as NO CHANGE and
// never runs.
//
// Observed against a live instance, where a one field change elsewhere in the blueprint
// stripped every link-granted IAM policy from the application's execution roles and
// reported success. The application was left running without the permissions its handlers
// depend on.
func (s *LinkProjectionDeployTestSuite) Test_update_deploy_keeps_fields_contributed_by_links() {
	stateContainer := memstate.NewMemoryStateContainer()
	deployedResource := &recordingLambdaResource{}
	loader, _ := newLinkProjectionLoader(
		stateContainer,
		func(lambdaResource provider.Resource) provider.Resource {
			deployedResource.Resource = lambdaResource
			return deployedResource
		},
	)
	params := newLinkProjectionParams()

	// The initial deploy runs the link, which writes the function's environment
	// variables and records that it owns them.
	instanceID := s.deployInitialInstanceForProjectionTest(loader, params)
	s.requireLinkRecordedItsContribution(stateContainer, instanceID)
	deployedResource.forget()

	updatedContainer, err := loader.Load(
		context.Background(),
		"__testdata/container/deploy/blueprint-link-projection-update.yml",
		params,
	)
	s.Require().NoError(err)

	updateChangeStagingChannels := createChangeStagingChannels()
	err = updatedContainer.StageChanges(
		context.Background(),
		&StageChangesInput{InstanceID: instanceID},
		updateChangeStagingChannels,
		params,
	)
	s.Require().NoError(err)

	updateChanges, err := consumeStagedChangesForTest(updateChangeStagingChannels)
	s.Require().NoError(err)
	// The change this deployment is for has nothing to do with the link, which is what
	// makes the contribution's removal silent.
	s.Require().Contains(updateChanges.ResourceChanges, "ordersFunction")
	s.requireNoChangesReportedForContribution(updateChanges.ResourceChanges["ordersFunction"])

	updateChannels := CreateDeployChannels()
	err = updatedContainer.Deploy(
		context.Background(),
		&DeployInput{
			InstanceID: instanceID,
			Changes:    updateChanges,
			Rollback:   false,
		},
		updateChannels,
		params,
	)
	s.Require().NoError(err)

	finishedMessage := consumeUntilFinishForTest(s.T(), updateChannels, "update deploy")
	s.Require().Equal(
		core.InstanceStatusUpdated,
		finishedMessage.Status,
		fmt.Sprintf("update deploy failed: %v", finishedMessage.FailureReasons),
	)

	deployedSpec := deployedResource.lastDeployedSpec()
	s.Require().NotNil(deployedSpec)

	tableName, _ := core.GetPathValue(
		"$.environment.variables.TABLE_NAME_ordersTable",
		deployedSpec,
		core.MappingNodeMaxTraverseDepth,
	)
	s.Require().NotNil(
		tableName,
		"the environment variable the link contributed was not part of the deployed spec, "+
			"so deploying removes it from the live resource",
	)
	s.Require().Equal("production-orders", core.StringValue(tableName))

	// The change the deployment is actually for must still be applied.
	handler, _ := core.GetPathValue("$.handler", deployedSpec, core.MappingNodeMaxTraverseDepth)
	s.Require().Equal("src/orders_v2.handler", core.StringValue(handler))

	s.requireStateHoldsOnlyTheDeclaredSpec(stateContainer, instanceID)
}

// The resource's persisted spec holds what the blueprint declares, and not what links
// have contributed.
//
// The link records its own contribution along with the resource field it wrote, and that
// record is what composition reads. Copying the value into the resource's spec as well
// would leave two records of the same thing, making it possible for them to diverge and
// makes it harder to reason about what the resource's spec actually contains when it differs
// from what the blueprint declares without cross-referencing with link data mappings.
// This is especially an issue for debugging when a deployment doesn't go to plan,
// when resource or link operations fail or are interrupted.
func (s *LinkProjectionDeployTestSuite) requireStateHoldsOnlyTheDeclaredSpec(
	stateContainer state.Container,
	instanceID string,
) {

	resourceState, err := stateContainer.Resources().GetByName(
		context.Background(),
		instanceID,
		"ordersFunction",
	)
	s.Require().NoError(err)

	contributed, _ := core.GetPathValue(
		"$.environment.variables.TABLE_NAME_ordersTable",
		resourceState.SpecData,
		core.MappingNodeMaxTraverseDepth,
	)
	s.Require().Nil(
		contributed,
		"the link's contribution was copied into the resource's own spec, which leaves "+
			"two records of it that can disagree",
	)

	// What the blueprint declares, including fields the provider computed, is still
	// recorded.
	handler, _ := core.GetPathValue(
		"$.handler",
		resourceState.SpecData,
		core.MappingNodeMaxTraverseDepth,
	)
	s.Require().Equal("src/orders_v2.handler", core.StringValue(handler))
}

// Removing a link takes away everything it contributed, and that has to be visible in the
// change set before it is applied.
//
// The fields belong to the link rather than to the blueprint, so they leave the resource
// when the link does. That is a correct consequence of removing the link, and an operator
// approving the change set is entitled to see it rather than discover it from the
// behaviour of a deployed application.
func (s *LinkProjectionDeployTestSuite) Test_staging_reports_fields_lost_when_a_link_is_removed() {
	stateContainer := memstate.NewMemoryStateContainer()
	deployedResource := &recordingLambdaResource{}
	loader, _ := newLinkProjectionLoader(
		stateContainer,
		func(lambdaResource provider.Resource) provider.Resource {
			deployedResource.Resource = lambdaResource
			return deployedResource
		},
	)
	params := newLinkProjectionParams()

	instanceID := s.deployInitialInstanceForProjectionTest(loader, params)
	s.requireLinkRecordedItsContribution(stateContainer, instanceID)

	unlinkedContainer, err := loader.Load(
		context.Background(),
		"__testdata/container/deploy/blueprint-link-projection-unlinked.yml",
		params,
	)
	s.Require().NoError(err)

	stagingChannels := createChangeStagingChannels()
	err = unlinkedContainer.StageChanges(
		context.Background(),
		&StageChangesInput{InstanceID: instanceID},
		stagingChannels,
		params,
	)
	s.Require().NoError(err)

	stagedChanges, err := consumeStagedChangesForTest(stagingChannels)
	s.Require().NoError(err)

	s.Require().Contains(
		stagedChanges.RemovedLinks,
		"ordersFunction::ordersTable",
		"the link itself must be staged for removal",
	)

	functionChanges, hasChanges := stagedChanges.ResourceChanges["ordersFunction"]
	s.Require().True(
		hasChanges,
		"the resource loses the fields the link contributed, so it has changes",
	)
	s.Require().Contains(
		functionChanges.RemovedFields,
		"spec.environment.variables.TABLE_NAME_ordersTable",
		"the field the removed link contributed is taken away without being reported",
	)

	// Reporting the path alone leaves an operator to work out why a field they never
	// wrote is disappearing.
	s.Require().Equal(
		"ordersFunction::ordersTable",
		functionChanges.LinkOwnedFields["spec.environment.variables.TABLE_NAME_ordersTable"],
		"the removed field is not attributed to the link that contributed it",
	)
}

// A contribution the framework has a record of and no value for is read back from the
// deployed resource, rather than the resource being deployed without it.
//
// A deployment that writes a resource through a link and stops before the link's data is
// saved leaves exactly this. The value is on the live resource, the mapping says where it
// belongs, and nothing holds the value itself. Recovering it is the difference between
// carrying on and refusing to deploy a resource that would lose the field.
func (s *LinkProjectionDeployTestSuite) Test_update_deploy_recovers_contribution_missing_from_link_data() {
	stateContainer := memstate.NewMemoryStateContainer()
	deployedResource := &recordingLambdaResource{}
	loader, _ := newLinkProjectionLoader(
		stateContainer,
		func(lambdaResource provider.Resource) provider.Resource {
			deployedResource.Resource = lambdaResource
			return deployedResource
		},
	)
	params := newLinkProjectionParams()

	instanceID := s.deployInitialInstanceForProjectionTest(loader, params)
	s.requireLinkRecordedItsContribution(stateContainer, instanceID)
	s.simulateLinkWritesToLiveResource(stateContainer, instanceID, deployedResource)
	s.forgetLinkData(stateContainer, instanceID)
	deployedResource.forget()

	updatedContainer, err := loader.Load(
		context.Background(),
		"__testdata/container/deploy/blueprint-link-projection-update.yml",
		params,
	)
	s.Require().NoError(err)

	stagingChannels := createChangeStagingChannels()
	err = updatedContainer.StageChanges(
		context.Background(),
		&StageChangesInput{InstanceID: instanceID},
		stagingChannels,
		params,
	)
	s.Require().NoError(err)

	updateChanges, err := consumeStagedChangesForTest(stagingChannels)
	s.Require().NoError(err)

	updateChannels := CreateDeployChannels()
	err = updatedContainer.Deploy(
		context.Background(),
		&DeployInput{
			InstanceID: instanceID,
			Changes:    updateChanges,
			Rollback:   false,
		},
		updateChannels,
		params,
	)
	s.Require().NoError(err)

	finishedMessage := consumeUntilFinishForTest(s.T(), updateChannels, "update deploy")
	s.Require().Equal(
		core.InstanceStatusUpdated,
		finishedMessage.Status,
		fmt.Sprintf("update deploy failed: %v", finishedMessage.FailureReasons),
	)

	deployedSpec := deployedResource.lastDeployedSpec()
	s.Require().NotNil(deployedSpec)

	tableName, _ := core.GetPathValue(
		"$.environment.variables.TABLE_NAME_ordersTable",
		deployedSpec,
		core.MappingNodeMaxTraverseDepth,
	)
	s.Require().NotNil(
		tableName,
		"the contribution was not read back from the deployed resource, so the resource "+
			"was either deployed without it or not deployed at all",
	)

	// What was read back is recorded against the link that owns it, so the next
	// deployment does not have to go looking for it again.
	links, err := stateContainer.Links().ListWithResourceDataMappings(
		context.Background(),
		instanceID,
		"ordersFunction",
	)
	s.Require().NoError(err)
	s.Require().NotEmpty(links)
	s.Require().NotEmpty(
		links[0].Data,
		"the recovered value was not recorded against the link",
	)
}

// Puts what the link contributed onto the live resource.
//
// A link writes to a resource after the framework has deployed it, through the provider's
// own API rather than through the framework, so the deployed resource carries the
// contribution while the spec the framework applied did not. The test provider's link does
// not reach a real service, so the effect of it is applied here.
func (s *LinkProjectionDeployTestSuite) simulateLinkWritesToLiveResource(
	stateContainer state.Container,
	instanceID string,
	resource *recordingLambdaResource,
) {

	links, err := stateContainer.Links().ListWithResourceDataMappings(
		context.Background(),
		instanceID,
		"ordersFunction",
	)
	s.Require().NoError(err)

	live, err := specmerge.ApplyLinkProjections(
		resource.lastDeployedSpec(),
		"ordersFunction",
		links,
	)
	s.Require().NoError(err)
	s.Require().Empty(live.Unresolved)

	resource.setLiveSpec(live.Spec)
}

// A contribution that is not on the deployed resource either, and so cannot be recovered,
// fails that resource rather than deploying it without the field.
//
// Deploying applies the resource's complete intended state, so a field a link owns that is
// missing from the spec is removed from the deployed resource. Leaving the resource alone
// keeps what links contributed to it live and intact.
func (s *LinkProjectionDeployTestSuite) Test_update_deploy_fails_resource_when_a_contribution_cannot_be_recovered() {
	stateContainer := memstate.NewMemoryStateContainer()
	deployedResource := &recordingLambdaResource{}
	loader, _ := newLinkProjectionLoader(
		stateContainer,
		func(lambdaResource provider.Resource) provider.Resource {
			deployedResource.Resource = lambdaResource
			return deployedResource
		},
	)
	params := newLinkProjectionParams()

	instanceID := s.deployInitialInstanceForProjectionTest(loader, params)
	s.requireLinkRecordedItsContribution(stateContainer, instanceID)

	// The value is in neither place it could be: not recorded against the link, and not
	// on the deployed resource to read back, since the link never reached it.
	s.forgetLinkData(stateContainer, instanceID)

	updatedContainer, err := loader.Load(
		context.Background(),
		"__testdata/container/deploy/blueprint-link-projection-update.yml",
		params,
	)
	s.Require().NoError(err)

	stagingChannels := createChangeStagingChannels()
	err = updatedContainer.StageChanges(
		context.Background(),
		&StageChangesInput{InstanceID: instanceID},
		stagingChannels,
		params,
	)
	s.Require().NoError(err)

	updateChanges, err := consumeStagedChangesForTest(stagingChannels)
	s.Require().NoError(err)

	updateChannels := CreateDeployChannels()
	err = updatedContainer.Deploy(
		context.Background(),
		&DeployInput{
			InstanceID: instanceID,
			Changes:    updateChanges,
			Rollback:   false,
		},
		updateChannels,
		params,
	)
	s.Require().NoError(err)

	finishedMessage := consumeUntilFinishForTest(s.T(), updateChannels, "update deploy")
	s.Require().NotEqual(
		core.InstanceStatusUpdated,
		finishedMessage.Status,
		"the resource was deployed without a field a link is recorded as contributing",
	)
	s.Require().NotEmpty(finishedMessage.FailureReasons)
}

// Empties a link's data while leaving its mappings in place, which is the state a
// deployment interrupted between writing a resource and saving its link leaves behind.
func (s *LinkProjectionDeployTestSuite) forgetLinkData(
	stateContainer state.Container,
	instanceID string,
) {

	links, err := stateContainer.Links().ListWithResourceDataMappings(
		context.Background(),
		instanceID,
		"ordersFunction",
	)
	s.Require().NoError(err)
	s.Require().NotEmpty(links)

	for _, link := range links {
		link.Data = map[string]*core.MappingNode{}
		s.Require().NoError(stateContainer.Links().Save(context.Background(), link))
	}
}

// A link that has not changed contributes the same fields to both sides of the comparison,
// so it produces no field changes at all.
//
// The change set is what a user reads before approving a deployment. A link's fields
// appearing as new or modified on every update, when nothing about them has changed, is
// noise that makes a real change harder to see.
func (s *LinkProjectionDeployTestSuite) requireNoChangesReportedForContribution(
	resourceChanges provider.Changes,
) {

	contributedField := "spec.environment.variables.TABLE_NAME_ordersTable"

	for _, fieldChange := range resourceChanges.NewFields {
		s.Require().NotEqual(
			contributedField,
			fieldChange.FieldPath,
			"a field the link contributed is reported as new when the link has not changed",
		)
	}

	for _, fieldChange := range resourceChanges.ModifiedFields {
		s.Require().NotEqual(
			contributedField,
			fieldChange.FieldPath,
			"a field the link contributed is reported as modified when the link has not changed",
		)
	}

	s.Require().NotContains(
		resourceChanges.RemovedFields,
		contributedField,
		"a field the link contributed is reported as removed when the link still contributes it",
	)
}

// The link's own record of what it contributed, written by the initial deploy. The
// framework has no other record of it: the function's persisted spec does not carry the
// link's environment variables.
func (s *LinkProjectionDeployTestSuite) requireLinkRecordedItsContribution(
	stateContainer state.Container,
	instanceID string,
) {

	links, err := stateContainer.Links().ListWithResourceDataMappings(
		context.Background(),
		instanceID,
		"ordersFunction",
	)
	s.Require().NoError(err)
	s.Require().NotEmpty(
		links,
		"the link recorded no contribution to the function, so this test would pass "+
			"whether or not the deployment composes them",
	)
}

func (s *LinkProjectionDeployTestSuite) deployInitialInstanceForProjectionTest(
	loader Loader,
	params core.BlueprintParams,
) string {

	initialContainer, err := loader.Load(
		context.Background(),
		"__testdata/container/deploy/blueprint-link-projection-initial.yml",
		params,
	)
	s.Require().NoError(err)

	stagingChannels := createChangeStagingChannels()
	err = initialContainer.StageChanges(
		context.Background(),
		&StageChangesInput{},
		stagingChannels,
		params,
	)
	s.Require().NoError(err)

	initialChanges, err := consumeStagedChangesForTest(stagingChannels)
	s.Require().NoError(err)

	deployChannels := CreateDeployChannels()
	err = initialContainer.Deploy(
		context.Background(),
		&DeployInput{
			InstanceName: "LinkProjectionInstance",
			Changes:      initialChanges,
			Rollback:     false,
		},
		deployChannels,
		params,
	)
	s.Require().NoError(err)

	finishedMessage := consumeUntilFinishForTest(s.T(), deployChannels, "initial deploy")
	s.Require().Equal(
		core.InstanceStatusDeployed,
		finishedMessage.Status,
		fmt.Sprintf("initial deploy failed: %v", finishedMessage.FailureReasons),
	)

	return finishedMessage.InstanceID
}

// The provider the rest of the container tests use, with the lambda function swapped for
// one that records the spec it was asked to deploy.
func newLinkProjectionLoader(
	stateContainer state.Container,
	recordDeployedSpec func(provider.Resource) provider.Resource,
) (Loader, provider.Provider) {
	awsProvider := newTestAWSProvider(
		/* alwaysStabilise */ true,
		/* skipRetryFailuresForLinkNames */ []string{"ordersFunction::ordersTable"},
		stateContainer,
	).(*internal.ProviderMock)
	awsProvider.Resources["aws/lambda/function"] = recordDeployedSpec(
		awsProvider.Resources["aws/lambda/function"],
	)

	loader := NewDefaultLoader(
		map[string]provider.Provider{"aws": awsProvider},
		map[string]transform.SpecTransformer{},
		stateContainer,
		newFSChildResolver(),
		WithLoaderTransformSpec(false),
		WithLoaderValidateRuntimeValues(true),
		WithLoaderRefChainCollectorFactory(refgraph.NewRefChainCollector),
		WithLoaderResourceStabilityPollingConfig(&ResourceStabilityPollingConfig{
			PollingInterval: 10 * time.Millisecond,
			PollingTimeout:  1 * time.Second,
		}),
		WithLoaderLogger(core.NewNopLogger()),
	)

	return loader, awsProvider
}

func newLinkProjectionParams() core.BlueprintParams {
	return core.NewDefaultParams(
		map[string]map[string]*core.ScalarValue{},
		map[string]map[string]*core.ScalarValue{},
		map[string]*core.ScalarValue{},
		map[string]*core.ScalarValue{},
	)
}

// A function that records the spec it was asked to deploy, so a test can assert on what
// the framework decided the resource should look like.
type recordingLambdaResource struct {
	provider.Resource
	mu         sync.Mutex
	deployedAs *core.MappingNode
	// The resource as it exists outside the framework, which is whatever was last
	// deployed to it. Unlike deployedAs it is not forgotten between deployments, since
	// the deployed resource does not stop existing between them.
	liveSpec *core.MappingNode
}

// The environment variables links contribute are part of the resource's schema, so that
// change staging can see them. A change set is produced by walking the spec definition, so
// a field absent from it is invisible to the comparison whether or not it is in the spec.
func (r *recordingLambdaResource) GetSpecDefinition(
	ctx context.Context,
	input *provider.ResourceGetSpecDefinitionInput,
) (*provider.ResourceGetSpecDefinitionOutput, error) {
	return &provider.ResourceGetSpecDefinitionOutput{
		SpecDefinition: &provider.ResourceSpecDefinition{
			Schema: &provider.ResourceDefinitionsSchema{
				Type: provider.ResourceDefinitionsSchemaTypeObject,
				Attributes: map[string]*provider.ResourceDefinitionsSchema{
					"id": {
						Type:     provider.ResourceDefinitionsSchemaTypeString,
						Computed: true,
					},
					"handler": {
						Type: provider.ResourceDefinitionsSchemaTypeString,
					},
					"policies": {
						Type:     provider.ResourceDefinitionsSchemaTypeArray,
						Nullable: true,
						Items: &provider.ResourceDefinitionsSchema{
							Type: provider.ResourceDefinitionsSchemaTypeString,
						},
					},
					"environment": {
						Type:     provider.ResourceDefinitionsSchemaTypeObject,
						Nullable: true,
						Attributes: map[string]*provider.ResourceDefinitionsSchema{
							"variables": {
								Type:     provider.ResourceDefinitionsSchemaTypeMap,
								Nullable: true,
								MapValues: &provider.ResourceDefinitionsSchema{
									Type: provider.ResourceDefinitionsSchemaTypeString,
								},
							},
						},
					},
				},
			},
		},
	}, nil
}

func (r *recordingLambdaResource) Deploy(
	ctx context.Context,
	input *provider.ResourceDeployInput,
) (*provider.ResourceDeployOutput, error) {
	r.mu.Lock()
	r.deployedAs = core.CopyMappingNode(
		input.Changes.AppliedResourceInfo.ResourceWithResolvedSubs.Spec,
	)
	r.liveSpec = core.CopyMappingNode(r.deployedAs)
	r.mu.Unlock()

	return r.Resource.Deploy(ctx, input)
}

func (r *recordingLambdaResource) setLiveSpec(spec *core.MappingNode) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.liveSpec = core.CopyMappingNode(spec)
}

// The resource as it is outside the framework, which is what was last deployed to it
// including everything links contributed.
func (r *recordingLambdaResource) GetExternalState(
	ctx context.Context,
	input *provider.ResourceGetExternalStateInput,
) (*provider.ResourceGetExternalStateOutput, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.liveSpec == nil {
		return r.Resource.GetExternalState(ctx, input)
	}

	return &provider.ResourceGetExternalStateOutput{
		ResourceSpecState: core.CopyMappingNode(r.liveSpec),
	}, nil
}

func (r *recordingLambdaResource) lastDeployedSpec() *core.MappingNode {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.deployedAs
}

// Clears what was recorded, so an assertion is made against the deployment under test
// rather than one that set it up.
func (r *recordingLambdaResource) forget() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.deployedAs = nil
}

// A contribution recorded as appended has to be deployed as an addition to the field it
// targets, not as its whole value.
func (s *LinkProjectionDeployTestSuite) Test_update_deploy_applies_an_appended_contribution_as_an_addition() {
	stateContainer := memstate.NewMemoryStateContainer()
	deployedResource := &recordingLambdaResource{}
	loader, _ := newLinkProjectionLoader(
		stateContainer,
		func(lambdaResource provider.Resource) provider.Resource {
			deployedResource.Resource = lambdaResource
			return deployedResource
		},
	)
	params := newLinkProjectionParams()

	instanceID := s.deployInitialInstanceForProjectionTest(loader, params)
	s.seedAppendedContributionForProjectionTest(stateContainer, instanceID)
	deployedResource.forget()

	updatedContainer, err := loader.Load(
		context.Background(),
		"__testdata/container/deploy/blueprint-link-projection-update.yml",
		params,
	)
	s.Require().NoError(err)

	updateChangeStagingChannels := createChangeStagingChannels()
	err = updatedContainer.StageChanges(
		context.Background(),
		&StageChangesInput{InstanceID: instanceID},
		updateChangeStagingChannels,
		params,
	)
	s.Require().NoError(err)

	updateChanges, err := consumeStagedChangesForTest(updateChangeStagingChannels)
	s.Require().NoError(err)

	updateChannels := CreateDeployChannels()
	err = updatedContainer.Deploy(
		context.Background(),
		&DeployInput{
			InstanceID: instanceID,
			Changes:    updateChanges,
		},
		updateChannels,
		params,
	)
	s.Require().NoError(err)

	finishedMessage := consumeUntilFinishForTest(s.T(), updateChannels, "update deploy")
	s.Require().Equal(core.InstanceStatusUpdated, finishedMessage.Status)

	s.Require().NotContains(
		updateChanges.RemovedLinks,
		"ordersFunction::ordersTable",
		"the contributing link is still in the blueprint, so its contribution stands",
	)

	deployedSpec := deployedResource.lastDeployedSpec()
	s.Require().NotNil(deployedSpec, "the function was not deployed")

	policies, err := core.GetPathValue(
		"$.policies",
		deployedSpec,
		core.MappingNodeMaxTraverseDepth,
	)
	s.Require().NoError(err)
	s.Require().NotNil(policies, "the contribution is absent from the deployed spec")
	s.Require().Equal(
		[]string{"dynamodb:PutItem"},
		mappingNodeItemsAsStrings(policies),
		"the contribution was deployed as the field's whole value rather than added to it",
	)
}

func mappingNodeItemsAsStrings(node *core.MappingNode) []string {
	items := []string{}
	for _, item := range node.Items {
		items = append(items, core.StringValue(item))
	}

	return items
}

// Records an appended contribution against the link the blueprint already has, as state
// holds one once a link has contributed.
//
// The link is left in the blueprint so that the deployment does not remove it, and it
// declares no contribution targets, so what it recorded is read back rather than replaced
// by what it produces when it runs.
func (s *LinkProjectionDeployTestSuite) seedAppendedContributionForProjectionTest(
	stateContainer state.Container,
	instanceID string,
) {

	existing, err := stateContainer.Links().GetByName(
		context.Background(),
		instanceID,
		"ordersFunction::ordersTable",
	)
	s.Require().NoError(err)

	contributed := ContributionsToLinkData([]*provider.ResourceContribution{
		{
			ResourceName: "ordersFunction",
			FieldPath:    "spec.policies",
			Value:        core.MappingNodeFromString("dynamodb:PutItem"),
			Action:       provider.ContributionActionAppend,
		},
	})

	// Kept alongside what the link already recorded, since a link contributes through the
	// same mappings it writes imperatively through.
	maps.Copy(contributed.ResourceDataMappings, existing.ResourceDataMappings)
	maps.Copy(contributed.Data, existing.Data)

	existing.Data = contributed.Data
	existing.ResourceDataMappings = contributed.ResourceDataMappings
	existing.ContributionRecords = contributed.ContributionRecords

	s.Require().NoError(stateContainer.Links().Save(context.Background(), existing))
}

func TestLinkProjectionDeployTestSuite(t *testing.T) {
	suite.Run(t, new(LinkProjectionDeployTestSuite))
}
