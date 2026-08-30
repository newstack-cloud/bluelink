package container

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/changes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal/memstate"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/providerhelpers"
	"github.com/newstack-cloud/bluelink/libs/blueprint/refgraph"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
)

const linkThroughputTimeout = 5 * time.Minute

// Shape of a generated blueprint, and the per-phase cost of a link in it.
//
// The table is the priority resource of a function-to-table link, so a function is
// deployed after every table it links to. A function completing is therefore what
// releases its links, which is the arrangement the throughput brief measured against a
// real account: "for a Lambda function that is typically its ~3 outbound links".
//
// Tables are private to a function rather than shared. A shared table would put every
// link that reaches it behind one resource lock, which is a real effect but a different
// one, and it would hide the question this asks which is whether links on unrelated resources
// run at the same time.
type linkThroughputShape struct {
	functions         int
	tablesPerFunction int
	// sharedTable puts every function's links on one table instead of giving each
	// function its own, so the links share their resource B while their resource A
	// differs. Several functions reading one table is an ordinary blueprint, and it is
	// the shape that distinguishes resource affinity keyed on A from affinity keyed on
	// both endpoints.
	sharedTable          bool
	updateResourceA      time.Duration
	updateResourceB      time.Duration
	updateIntermediaries time.Duration
}

func (s linkThroughputShape) links() int {
	return s.functions * s.tablesPerFunction
}

func (s linkThroughputShape) name() string {
	if s.sharedTable {
		return fmt.Sprintf(
			"functions=%d/sharedTable/links=%d",
			s.functions,
			s.links(),
		)
	}

	return fmt.Sprintf(
		"functions=%d/linksPerFunction=%d/links=%d",
		s.functions,
		s.tablesPerFunction,
		s.links(),
	)
}

// Samples how many links are inside a link phase at the same time, and how long each
// resource spends with a link writing it.
//
// Concurrency is sampled on transitions rather than on a timer, every enter and leave
// closes off the interval that preceded it, so the mean is time-weighted and a burst
// that lasts a millisecond cannot be missed by a sampler that happened to look either
// side of it.
type linkThroughputRecorder struct {
	mu sync.Mutex

	inFlight int
	peak     int

	lastChange     time.Time
	weightedInUse  time.Duration
	observedWindow time.Duration

	resourceBusy  map[string]time.Duration
	resourceStart map[string]time.Time
}

func newLinkThroughputRecorder() *linkThroughputRecorder {
	return &linkThroughputRecorder{
		resourceBusy:  map[string]time.Duration{},
		resourceStart: map[string]time.Time{},
	}
}

func (r *linkThroughputRecorder) enter(resourceName string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.closeInterval()
	r.inFlight++
	if r.inFlight > r.peak {
		r.peak = r.inFlight
	}

	if _, alreadyWriting := r.resourceStart[resourceName]; !alreadyWriting {
		r.resourceStart[resourceName] = time.Now()
	}
}

func (r *linkThroughputRecorder) leave(resourceName string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.closeInterval()
	r.inFlight--

	if startedAt, writing := r.resourceStart[resourceName]; writing {
		r.resourceBusy[resourceName] += time.Since(startedAt)
		delete(r.resourceStart, resourceName)
	}
}

// The interval that just ended is credited at the concurrency it was held at, which is
// what makes the mean time-weighted rather than an average over transitions.
//
// The mutex must be held when calling this.
func (r *linkThroughputRecorder) closeInterval() {
	now := time.Now()
	if !r.lastChange.IsZero() {
		elapsed := now.Sub(r.lastChange)
		r.observedWindow += elapsed
		r.weightedInUse += time.Duration(r.inFlight) * elapsed
	}
	r.lastChange = now
}

func (r *linkThroughputRecorder) peakInFlight() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.peak
}

// The span from the first link phase starting to the last one ending, which is the link
// phase itself rather than the deployment around it. Resource deployment dominates the
// wall clock of a generated blueprint and would drown out what this is measuring.
func (r *linkThroughputRecorder) linkPhaseDuration() time.Duration {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.observedWindow
}

func (r *linkThroughputRecorder) meanInFlight() float64 {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.observedWindow == 0 {
		return 0
	}

	return float64(r.weightedInUse) / float64(r.observedWindow)
}

// The longest any single resource spent with a link writing it, which is the
// per-resource serialisation the model calls `d · c`.
func (r *linkThroughputRecorder) worstResourceBusy() time.Duration {
	r.mu.Lock()
	defer r.mu.Unlock()

	worst := time.Duration(0)
	for _, busy := range r.resourceBusy {
		if busy > worst {
			worst = busy
		}
	}

	return worst
}

// Gives a link a configurable cost in each of its three phases, and reports when it is
// inside one.
//
// Only the resource each phase actually writes is recorded as busy, resource A for
// UpdateResourceA and resource B for UpdateResourceB. Attributing both phases to both
// resources would report serialisation that is not there.
type latencyObservedLink struct {
	provider.Link
	shape    linkThroughputShape
	recorder *linkThroughputRecorder
}

func (l *latencyObservedLink) UpdateResourceA(
	ctx context.Context,
	input *provider.LinkUpdateResourceInput,
) (*provider.LinkUpdateResourceOutput, error) {
	l.recorder.enter(input.ResourceInfo.ResourceName)
	defer l.recorder.leave(input.ResourceInfo.ResourceName)

	if err := sleepWithContext(ctx, l.shape.updateResourceA); err != nil {
		return nil, err
	}

	return l.Link.UpdateResourceA(ctx, input)
}

func (l *latencyObservedLink) UpdateResourceB(
	ctx context.Context,
	input *provider.LinkUpdateResourceInput,
) (*provider.LinkUpdateResourceOutput, error) {
	l.recorder.enter(input.ResourceInfo.ResourceName)
	defer l.recorder.leave(input.ResourceInfo.ResourceName)

	if err := sleepWithContext(ctx, l.shape.updateResourceB); err != nil {
		return nil, err
	}

	return l.Link.UpdateResourceB(ctx, input)
}

func (l *latencyObservedLink) UpdateIntermediaryResources(
	ctx context.Context,
	input *provider.LinkUpdateIntermediaryResourcesInput,
) (*provider.LinkUpdateIntermediaryResourcesOutput, error) {
	l.recorder.enter(input.ResourceAInfo.ResourceName)
	defer l.recorder.leave(input.ResourceAInfo.ResourceName)

	if err := sleepWithContext(ctx, l.shape.updateIntermediaries); err != nil {
		return nil, err
	}

	return l.Link.UpdateIntermediaryResources(ctx, input)
}

func latencyObservedAWSProvider(
	stateContainer state.Container,
	shape linkThroughputShape,
	recorder *linkThroughputRecorder,
) provider.Provider {
	awsProvider := newTestAWSProvider(
		/* alwaysStabilise */ true,
		/* skipRetryFailuresForLinkNames */ []string{},
		stateContainer,
	)
	mock := awsProvider.(*internal.ProviderMock)

	observed := map[string]provider.Link{}
	for linkType, linkImpl := range mock.Links {
		observed[linkType] = &latencyObservedLink{
			Link:     linkImpl,
			shape:    shape,
			recorder: recorder,
		}
	}
	mock.Links = observed

	return mock
}

func generateLinkThroughputBlueprint(shape linkThroughputShape) string {
	var spec strings.Builder
	spec.WriteString("version: 2025-11-02\nresources:\n")

	if shape.sharedTable {
		return generateSharedTableBlueprint(shape)
	}

	for functionIndex := range shape.functions {
		group := fmt.Sprintf("group%d", functionIndex)

		for tableIndex := range shape.tablesPerFunction {
			fmt.Fprintf(&spec, `  table%d_%d:
    type: aws/dynamodb/table
    metadata:
      labels:
        linkGroup: %s
    spec:
      tableName: "table-%d-%d"
      region: "eu-west-2"
`, functionIndex, tableIndex, group, functionIndex, tableIndex)
		}

		fmt.Fprintf(&spec, `  function%d:
    type: aws/lambda/function
    linkSelector:
      byLabel:
        linkGroup: %s
    spec:
      handler: "src/handler%d.handler"
`, functionIndex, group, functionIndex)
	}

	return spec.String()
}

// Every function links to the same tables, so the links share their resource B.
func generateSharedTableBlueprint(shape linkThroughputShape) string {
	var spec strings.Builder
	spec.WriteString("version: 2025-11-02\nresources:\n")

	for tableIndex := range shape.tablesPerFunction {
		spec.WriteString(fmt.Sprintf(`  sharedTable%d:
    type: aws/dynamodb/table
    metadata:
      labels:
        linkGroup: shared
    spec:
      tableName: "shared-table-%d"
      region: "eu-west-2"
`, tableIndex, tableIndex))
	}

	for functionIndex := range shape.functions {
		spec.WriteString(fmt.Sprintf(`  function%d:
    type: aws/lambda/function
    linkSelector:
      byLabel:
        linkGroup: shared
    spec:
      handler: "src/handler%d.handler"
`, functionIndex, functionIndex))
	}

	return spec.String()
}

func newLinkThroughputLoader(
	stateContainer state.Container,
	awsProvider provider.Provider,
) Loader {
	return NewDefaultLoader(
		map[string]provider.Provider{
			"aws":     awsProvider,
			"example": newTestExampleProvider(),
			"core": providerhelpers.NewCoreProvider(
				stateContainer.Links(),
				core.BlueprintInstanceIDFromContext,
				os.Getwd,
				provider.NewFileSourceRegistry(),
				core.SystemClock{},
			),
		},
		map[string]transform.SpecTransformer{},
		stateContainer,
		newFSChildResolver(),
		WithLoaderTransformSpec(false),
		WithLoaderValidateRuntimeValues(true),
		WithLoaderRefChainCollectorFactory(refgraph.NewRefChainCollector),
		WithLoaderLogger(core.NewNopLogger()),
	)
}

func stageLinkThroughputChanges(
	ctx context.Context,
	blueprintContainer BlueprintContainer,
	params core.BlueprintParams,
) (*changes.BlueprintChanges, error) {
	stagingChannels := createChangeStagingChannels()
	err := blueprintContainer.StageChanges(
		ctx,
		&StageChangesInput{InstanceID: ""},
		stagingChannels,
		params,
	)
	if err != nil {
		return nil, err
	}

	for {
		select {
		case <-stagingChannels.ChildChangesChan:
		case <-stagingChannels.LinkChangesChan:
		case <-stagingChannels.ResourceChangesChan:
		case changeSet := <-stagingChannels.CompleteChan:
			return &changeSet, nil
		case err := <-stagingChannels.ErrChan:
			return nil, err
		case <-time.After(linkThroughputTimeout):
			return nil, errors.New(timeoutMessage)
		}
	}
}

func awaitLinkThroughputDeployment(channels *DeployChannels) error {
	for {
		select {
		case <-channels.ResourceUpdateChan:
		case <-channels.ChildUpdateChan:
		case <-channels.LinkUpdateChan:
		case <-channels.DeploymentUpdateChan:
		case <-channels.FinishChan:
			return nil
		case err := <-channels.ErrChan:
			return err
		case <-time.After(linkThroughputTimeout):
			return errors.New(timeoutMessage)
		}
	}
}

type linkThroughputResult struct {
	deployment        time.Duration
	linkPhase         time.Duration
	peakInFlight      int
	meanInFlight      float64
	worstResourceBusy time.Duration
}

func runLinkThroughputDeployment(
	ctx context.Context,
	shape linkThroughputShape,
	runIndex int,
) (*linkThroughputResult, error) {
	recorder := newLinkThroughputRecorder()
	stateContainer := memstate.NewMemoryStateContainer()
	loader := newLinkThroughputLoader(
		stateContainer,
		latencyObservedAWSProvider(stateContainer, shape, recorder),
	)

	params := core.NewDefaultParams(
		map[string]map[string]*core.ScalarValue{},
		map[string]map[string]*core.ScalarValue{},
		map[string]*core.ScalarValue{},
		map[string]*core.ScalarValue{},
	)

	blueprintContainer, err := loader.LoadString(
		ctx,
		generateLinkThroughputBlueprint(shape),
		schema.YAMLSpecFormat,
		params,
	)
	if err != nil {
		return nil, err
	}

	deployChanges, err := stageLinkThroughputChanges(ctx, blueprintContainer, params)
	if err != nil {
		return nil, err
	}

	channels := CreateDeployChannels()
	startedAt := time.Now()
	err = blueprintContainer.Deploy(
		ctx,
		&DeployInput{
			InstanceName: fmt.Sprintf("LinkThroughputInstance%d", runIndex),
			Changes:      deployChanges,
			Rollback:     false,
		},
		channels,
		params,
	)
	if err != nil {
		return nil, err
	}

	if err := awaitLinkThroughputDeployment(channels); err != nil {
		return nil, err
	}

	return &linkThroughputResult{
		deployment:        time.Since(startedAt),
		linkPhase:         recorder.linkPhaseDuration(),
		peakInFlight:      recorder.peakInFlight(),
		meanInFlight:      recorder.meanInFlight(),
		worstResourceBusy: recorder.worstResourceBusy(),
	}, nil
}

// Deploys a generated blueprint against a caller-supplied provider, for tests that need to
// observe what the provider was asked to do rather than how long the deployment took.
func deployGeneratedBlueprint(
	ctx context.Context,
	shape linkThroughputShape,
	stateContainer state.Container,
	awsProvider provider.Provider,
) error {
	return deployGeneratedBlueprintSpec(
		ctx,
		generateLinkThroughputBlueprint(shape),
		stateContainer,
		awsProvider,
	)
}

func deployGeneratedBlueprintSpec(
	ctx context.Context,
	spec string,
	stateContainer state.Container,
	awsProvider provider.Provider,
) error {
	loader := newLinkThroughputLoader(stateContainer, awsProvider)
	params := core.NewDefaultParams(
		map[string]map[string]*core.ScalarValue{},
		map[string]map[string]*core.ScalarValue{},
		map[string]*core.ScalarValue{},
		map[string]*core.ScalarValue{},
	)

	blueprintContainer, err := loader.LoadString(
		ctx,
		spec,
		schema.YAMLSpecFormat,
		params,
	)
	if err != nil {
		return err
	}

	deployChanges, err := stageLinkThroughputChanges(ctx, blueprintContainer, params)
	if err != nil {
		return err
	}

	channels := CreateDeployChannels()
	err = blueprintContainer.Deploy(
		ctx,
		&DeployInput{
			InstanceName: "SettleObservedInstance",
			Changes:      deployChanges,
			Rollback:     false,
		},
		channels,
		params,
	)
	if err != nil {
		return err
	}

	return awaitLinkThroughputDeployment(channels)
}

// BenchmarkLinkThroughput measures how many links a deployment runs at once, and what
// that costs in wall clock, against a mock provider with a configurable per-phase
// latency.
//
// The metric that matters is peak links in flight. Wall clock is reported because it is
// the thing being complained about, but it is a property of the latency configured here
// rather than of anything real, so it is only meaningful compared between runs of this
// benchmark.
func BenchmarkLinkThroughput(b *testing.B) {
	shapes := []linkThroughputShape{
		{functions: 8, tablesPerFunction: 1},
		{functions: 8, tablesPerFunction: 3},
		{functions: 8, tablesPerFunction: 6},
		{functions: 20, tablesPerFunction: 3},
		{functions: 12, tablesPerFunction: 1, sharedTable: true},
		{functions: 12, tablesPerFunction: 3, sharedTable: true},
	}

	for _, shape := range shapes {
		shape.updateResourceA = 20 * time.Millisecond
		shape.updateResourceB = 20 * time.Millisecond
		shape.updateIntermediaries = 20 * time.Millisecond

		b.Run(shape.name(), func(b *testing.B) {
			results := make([]*linkThroughputResult, 0, b.N)

			for runIndex := range b.N {
				result, err := runLinkThroughputDeployment(
					context.Background(),
					shape,
					runIndex,
				)
				if err != nil {
					b.Fatalf("link throughput deployment failed: %v", err)
				}
				results = append(results, result)
			}

			reportLinkThroughput(b, shape, results)
		})
	}
}

func reportLinkThroughput(
	b *testing.B,
	shape linkThroughputShape,
	results []*linkThroughputResult,
) {
	b.Helper()

	if len(results) == 0 {
		return
	}

	peaks := make([]int, 0, len(results))
	meanSum := 0.0
	linkPhaseSum := time.Duration(0)
	deploymentSum := time.Duration(0)
	worstBusy := time.Duration(0)

	for _, result := range results {
		peaks = append(peaks, result.peakInFlight)
		meanSum += result.meanInFlight
		linkPhaseSum += result.linkPhase
		deploymentSum += result.deployment
		if result.worstResourceBusy > worstBusy {
			worstBusy = result.worstResourceBusy
		}
	}
	sort.Ints(peaks)

	runs := float64(len(results))
	linkPhase := linkPhaseSum.Seconds() / runs

	b.ReportMetric(float64(peaks[len(peaks)/2]), "peak-links-in-flight")
	b.ReportMetric(meanSum/runs, "mean-links-in-flight")
	b.ReportMetric(float64(shape.links()), "links")
	b.ReportMetric(linkPhase, "link-phase-secs")
	b.ReportMetric(float64(shape.links())/linkPhase, "links-per-sec")
	b.ReportMetric(deploymentSum.Seconds()/runs, "deployment-secs")
	b.ReportMetric(worstBusy.Seconds(), "worst-resource-busy-secs")
}
