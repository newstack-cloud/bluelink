package container

import (
	"context"
	"fmt"
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
	"github.com/stretchr/testify/require"
)

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
func TestUpdateDeployKeepsFieldsContributedByLinks(t *testing.T) {
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
	instanceID := deployInitialInstanceForProjectionTest(t, loader, params)
	requireLinkRecordedItsContribution(t, stateContainer, instanceID)
	deployedResource.forget()

	updatedContainer, err := loader.Load(
		context.Background(),
		"__testdata/container/deploy/blueprint-link-projection-update.yml",
		params,
	)
	require.NoError(t, err)

	updateChangeStagingChannels := createChangeStagingChannels()
	err = updatedContainer.StageChanges(
		context.Background(),
		&StageChangesInput{InstanceID: instanceID},
		updateChangeStagingChannels,
		params,
	)
	require.NoError(t, err)

	updateChanges, err := consumeStagedChangesForTest(updateChangeStagingChannels)
	require.NoError(t, err)
	// The change this deployment is for has nothing to do with the link, which is what
	// makes the contribution's removal silent.
	require.Contains(t, updateChanges.ResourceChanges, "ordersFunction")
	requireNoChangesReportedForContribution(t, updateChanges.ResourceChanges["ordersFunction"])

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
	require.NoError(t, err)

	finishedMessage := consumeUntilFinishForTest(t, updateChannels, "update deploy")
	require.Equal(
		t,
		core.InstanceStatusUpdated,
		finishedMessage.Status,
		fmt.Sprintf("update deploy failed: %v", finishedMessage.FailureReasons),
	)

	deployedSpec := deployedResource.lastDeployedSpec()
	require.NotNil(t, deployedSpec)

	tableName, _ := core.GetPathValue(
		"$.environment.variables.TABLE_NAME_ordersTable",
		deployedSpec,
		core.MappingNodeMaxTraverseDepth,
	)
	require.NotNil(
		t,
		tableName,
		"the environment variable the link contributed was not part of the deployed spec, "+
			"so deploying removes it from the live resource",
	)
	require.Equal(t, "production-orders", core.StringValue(tableName))

	// The change the deployment is actually for must still be applied.
	handler, _ := core.GetPathValue("$.handler", deployedSpec, core.MappingNodeMaxTraverseDepth)
	require.Equal(t, "src/orders_v2.handler", core.StringValue(handler))

	requireStateHoldsOnlyTheDeclaredSpec(t, stateContainer, instanceID)
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
func requireStateHoldsOnlyTheDeclaredSpec(
	t *testing.T,
	stateContainer state.Container,
	instanceID string,
) {
	t.Helper()

	resourceState, err := stateContainer.Resources().GetByName(
		context.Background(),
		instanceID,
		"ordersFunction",
	)
	require.NoError(t, err)

	contributed, _ := core.GetPathValue(
		"$.environment.variables.TABLE_NAME_ordersTable",
		resourceState.SpecData,
		core.MappingNodeMaxTraverseDepth,
	)
	require.Nil(
		t,
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
	require.Equal(t, "src/orders_v2.handler", core.StringValue(handler))
}

// Removing a link takes away everything it contributed, and that has to be visible in the
// change set before it is applied.
//
// The fields belong to the link rather than to the blueprint, so they leave the resource
// when the link does. That is a correct consequence of removing the link, and an operator
// approving the change set is entitled to see it rather than discover it from the
// behaviour of a deployed application.
func TestStagingReportsFieldsLostWhenALinkIsRemoved(t *testing.T) {
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

	instanceID := deployInitialInstanceForProjectionTest(t, loader, params)
	requireLinkRecordedItsContribution(t, stateContainer, instanceID)

	unlinkedContainer, err := loader.Load(
		context.Background(),
		"__testdata/container/deploy/blueprint-link-projection-unlinked.yml",
		params,
	)
	require.NoError(t, err)

	stagingChannels := createChangeStagingChannels()
	err = unlinkedContainer.StageChanges(
		context.Background(),
		&StageChangesInput{InstanceID: instanceID},
		stagingChannels,
		params,
	)
	require.NoError(t, err)

	stagedChanges, err := consumeStagedChangesForTest(stagingChannels)
	require.NoError(t, err)

	require.Contains(
		t,
		stagedChanges.RemovedLinks,
		"ordersFunction::ordersTable",
		"the link itself must be staged for removal",
	)

	functionChanges, hasChanges := stagedChanges.ResourceChanges["ordersFunction"]
	require.True(
		t,
		hasChanges,
		"the resource loses the fields the link contributed, so it has changes",
	)
	require.Contains(
		t,
		functionChanges.RemovedFields,
		"spec.environment.variables.TABLE_NAME_ordersTable",
		"the field the removed link contributed is taken away without being reported",
	)

	// Reporting the path alone leaves an operator to work out why a field they never
	// wrote is disappearing.
	require.Equal(
		t,
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
func TestUpdateDeployRecoversContributionMissingFromLinkData(t *testing.T) {
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

	instanceID := deployInitialInstanceForProjectionTest(t, loader, params)
	requireLinkRecordedItsContribution(t, stateContainer, instanceID)
	simulateLinkWritesToLiveResource(t, stateContainer, instanceID, deployedResource)
	forgetLinkData(t, stateContainer, instanceID)
	deployedResource.forget()

	updatedContainer, err := loader.Load(
		context.Background(),
		"__testdata/container/deploy/blueprint-link-projection-update.yml",
		params,
	)
	require.NoError(t, err)

	stagingChannels := createChangeStagingChannels()
	err = updatedContainer.StageChanges(
		context.Background(),
		&StageChangesInput{InstanceID: instanceID},
		stagingChannels,
		params,
	)
	require.NoError(t, err)

	updateChanges, err := consumeStagedChangesForTest(stagingChannels)
	require.NoError(t, err)

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
	require.NoError(t, err)

	finishedMessage := consumeUntilFinishForTest(t, updateChannels, "update deploy")
	require.Equal(
		t,
		core.InstanceStatusUpdated,
		finishedMessage.Status,
		fmt.Sprintf("update deploy failed: %v", finishedMessage.FailureReasons),
	)

	deployedSpec := deployedResource.lastDeployedSpec()
	require.NotNil(t, deployedSpec)

	tableName, _ := core.GetPathValue(
		"$.environment.variables.TABLE_NAME_ordersTable",
		deployedSpec,
		core.MappingNodeMaxTraverseDepth,
	)
	require.NotNil(
		t,
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
	require.NoError(t, err)
	require.NotEmpty(t, links)
	require.NotEmpty(
		t,
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
func simulateLinkWritesToLiveResource(
	t *testing.T,
	stateContainer state.Container,
	instanceID string,
	resource *recordingLambdaResource,
) {
	t.Helper()

	links, err := stateContainer.Links().ListWithResourceDataMappings(
		context.Background(),
		instanceID,
		"ordersFunction",
	)
	require.NoError(t, err)

	live, err := specmerge.ApplyLinkProjections(
		resource.lastDeployedSpec(),
		"ordersFunction",
		links,
	)
	require.NoError(t, err)
	require.Empty(t, live.Unresolved)

	resource.setLiveSpec(live.Spec)
}

// A contribution that is not on the deployed resource either, and so cannot be recovered,
// fails that resource rather than deploying it without the field.
//
// Deploying applies the resource's complete intended state, so a field a link owns that is
// missing from the spec is removed from the deployed resource. Leaving the resource alone
// keeps what links contributed to it live and intact.
func TestUpdateDeployFailsResourceWhenAContributionCannotBeRecovered(t *testing.T) {
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

	instanceID := deployInitialInstanceForProjectionTest(t, loader, params)
	requireLinkRecordedItsContribution(t, stateContainer, instanceID)

	// The value is in neither place it could be: not recorded against the link, and not
	// on the deployed resource to read back, since the link never reached it.
	forgetLinkData(t, stateContainer, instanceID)

	updatedContainer, err := loader.Load(
		context.Background(),
		"__testdata/container/deploy/blueprint-link-projection-update.yml",
		params,
	)
	require.NoError(t, err)

	stagingChannels := createChangeStagingChannels()
	err = updatedContainer.StageChanges(
		context.Background(),
		&StageChangesInput{InstanceID: instanceID},
		stagingChannels,
		params,
	)
	require.NoError(t, err)

	updateChanges, err := consumeStagedChangesForTest(stagingChannels)
	require.NoError(t, err)

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
	require.NoError(t, err)

	finishedMessage := consumeUntilFinishForTest(t, updateChannels, "update deploy")
	require.NotEqual(
		t,
		core.InstanceStatusUpdated,
		finishedMessage.Status,
		"the resource was deployed without a field a link is recorded as contributing",
	)
	require.NotEmpty(t, finishedMessage.FailureReasons)
}

// Empties a link's data while leaving its mappings in place, which is the state a
// deployment interrupted between writing a resource and saving its link leaves behind.
func forgetLinkData(
	t *testing.T,
	stateContainer state.Container,
	instanceID string,
) {
	t.Helper()

	links, err := stateContainer.Links().ListWithResourceDataMappings(
		context.Background(),
		instanceID,
		"ordersFunction",
	)
	require.NoError(t, err)
	require.NotEmpty(t, links)

	for _, link := range links {
		link.Data = map[string]*core.MappingNode{}
		require.NoError(t, stateContainer.Links().Save(context.Background(), link))
	}
}

// A link that has not changed contributes the same fields to both sides of the comparison,
// so it produces no field changes at all.
//
// The change set is what a user reads before approving a deployment. A link's fields
// appearing as new or modified on every update, when nothing about them has changed, is
// noise that makes a real change harder to see.
func requireNoChangesReportedForContribution(
	t *testing.T,
	resourceChanges provider.Changes,
) {
	t.Helper()

	contributedField := "spec.environment.variables.TABLE_NAME_ordersTable"

	for _, fieldChange := range resourceChanges.NewFields {
		require.NotEqual(
			t,
			contributedField,
			fieldChange.FieldPath,
			"a field the link contributed is reported as new when the link has not changed",
		)
	}

	for _, fieldChange := range resourceChanges.ModifiedFields {
		require.NotEqual(
			t,
			contributedField,
			fieldChange.FieldPath,
			"a field the link contributed is reported as modified when the link has not changed",
		)
	}

	require.NotContains(
		t,
		resourceChanges.RemovedFields,
		contributedField,
		"a field the link contributed is reported as removed when the link still contributes it",
	)
}

// The link's own record of what it contributed, written by the initial deploy. The
// framework has no other record of it: the function's persisted spec does not carry the
// link's environment variables.
func requireLinkRecordedItsContribution(
	t *testing.T,
	stateContainer state.Container,
	instanceID string,
) {
	t.Helper()

	links, err := stateContainer.Links().ListWithResourceDataMappings(
		context.Background(),
		instanceID,
		"ordersFunction",
	)
	require.NoError(t, err)
	require.NotEmpty(
		t,
		links,
		"the link recorded no contribution to the function, so this test would pass "+
			"whether or not the deployment composes them",
	)
}

func deployInitialInstanceForProjectionTest(
	t *testing.T,
	loader Loader,
	params core.BlueprintParams,
) string {
	t.Helper()

	initialContainer, err := loader.Load(
		context.Background(),
		"__testdata/container/deploy/blueprint-link-projection-initial.yml",
		params,
	)
	require.NoError(t, err)

	stagingChannels := createChangeStagingChannels()
	err = initialContainer.StageChanges(
		context.Background(),
		&StageChangesInput{},
		stagingChannels,
		params,
	)
	require.NoError(t, err)

	initialChanges, err := consumeStagedChangesForTest(stagingChannels)
	require.NoError(t, err)

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
	require.NoError(t, err)

	finishedMessage := consumeUntilFinishForTest(t, deployChannels, "initial deploy")
	require.Equal(
		t,
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
