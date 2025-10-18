package subengine

import (
	"context"
	"fmt"
	"slices"
	"strings"

	bpcore "github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/errors"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/resourcehelpers"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/speccore"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
	"github.com/newstack-cloud/bluelink/libs/common/core"
)

// SubstitutionResolver is an interface that provides functionality to resolve
// substitutions in components of a blueprint.
// Resolving involves taking a parsed representation of a substitution, resolving referenced
// values and executing functions to produce a final output.
type SubstitutionResolver interface {
	// ResolveInResource resolves substitutions in a resource.
	ResolveInResource(
		ctx context.Context,
		resourceName string,
		resource *schema.Resource,
		resolveTargetInfo *ResolveResourceTargetInfo,
	) (*ResolveInResourceResult, error)
	// ResolveResourceEach resolves the substitution in the `each` property of a resource
	// that is expected to resolve to a list of items that will be mapped to a planned and
	// eventually deployed resource.
	ResolveResourceEach(
		ctx context.Context,
		resourceName string,
		resource *schema.Resource,
		resolveFor ResolveForStage,
	) ([]*bpcore.MappingNode, error)
	// ResolveInDataSource resolves substitutions in a data source.
	ResolveInDataSource(
		ctx context.Context,
		dataSourceName string,
		dataSource *schema.DataSource,
		resolveTargetInfo *ResolveDataSourceTargetInfo,
	) (*ResolveInDataSourceResult, error)
	// ResolveInMappingNode resolves substitutions in a mapping node, primarily used
	// for the top-level blueprint metadata.
	ResolveInMappingNode(
		ctx context.Context,
		currentElementName string,
		mappingNode *bpcore.MappingNode,
		resolveTargetInfo *ResolveMappingNodeTargetInfo,
	) (*ResolveInMappingNodeResult, error)
	// ResolveInValue resolves substitutions in a value.
	ResolveInValue(
		ctx context.Context,
		valueName string,
		value *schema.Value,
		resolveTargetInfo *ResolveValueTargetInfo,
	) (*ResolveInValueResult, error)
	// ResolveInInclude resolves substitutions in an include.
	ResolveInInclude(
		ctx context.Context,
		includeName string,
		include *schema.Include,
		resolveTargetInfo *ResolveIncludeTargetInfo,
	) (*ResolveInIncludeResult, error)
	// ResolveInExport resolves substitutions in an export.
	ResolveInExport(
		ctx context.Context,
		exportName string,
		export *schema.Export,
		resolveTargetInfo *ResolveExportTargetInfo,
	) (*ResolveInExportResult, error)
	// ResolveSubstitution resolves a substitution value from a string or a substitution
	// in a provided context.
	// This is to be used to resolve values outside the regular substitution resolution context
	// (e.g. resolving a reference to a resource property in an export).
	ResolveSubstitution(
		ctx context.Context,
		value *substitutions.StringOrSubstitution,
		inElementName string,
		inElementProperty string,
		resolveTargetInfo *ResolveTargetInfo,
	) (*ResolveResult, error)
}

// ResolveTargetInfo contains information about the target of a substitution
// that is being resolved in a resource.
type ResolveResourceTargetInfo struct {
	ResolveFor        ResolveForStage
	PartiallyResolved *provider.ResolvedResource
}

// ResolveInResourceResult contains a resolved resource
// and a list of full property paths that must be resolved during deployment.
type ResolveInResourceResult struct {
	ResolvedResource *provider.ResolvedResource
	ResolveOnDeploy  []string
}

// ResolveDataSourceTargetInfo contains information about the target of a substitution
// that is being resolved in a data source.
type ResolveDataSourceTargetInfo struct {
	ResolveFor        ResolveForStage
	PartiallyResolved *provider.ResolvedDataSource
}

// ResolveInDataSourceResult contains a resolved data source
// and a list of full property paths that must be resolved during deployment.
type ResolveInDataSourceResult struct {
	ResolvedDataSource *provider.ResolvedDataSource
	ResolveOnDeploy    []string
}

// ResolveMappingNodeTargetInfo contains information about the target of a substitution
// that is being resolved in a mapping node.
type ResolveMappingNodeTargetInfo struct {
	ResolveFor        ResolveForStage
	PartiallyResolved *bpcore.MappingNode
}

// ResolveInMappingNodeResult contains a resolved mapping node
// and a list of full property paths that must be resolved during deployment.
type ResolveInMappingNodeResult struct {
	ResolvedMappingNode *bpcore.MappingNode
	ResolveOnDeploy     []string
}

// ResolveValueTargetInfo contains information about the target of a substitution
// that is being resolved in a value.
type ResolveValueTargetInfo struct {
	ResolveFor        ResolveForStage
	PartiallyResolved *ResolvedValue
}

// ResolveInValueResult contains a resolved value
// and a list of full property paths that must be resolved during deployment.
type ResolveInValueResult struct {
	ResolvedValue   *ResolvedValue
	ResolveOnDeploy []string
}

// ResolveIncludeTargetInfo contains information about the target of a substitution
// that is being resolved in a child blueprint include.
type ResolveIncludeTargetInfo struct {
	ResolveFor        ResolveForStage
	PartiallyResolved *ResolvedInclude
}

// ResolveInIncludeResult contains a resolved include
// and a list of full property paths that must be resolved during deployment.
type ResolveInIncludeResult struct {
	ResolvedInclude *ResolvedInclude
	ResolveOnDeploy []string
}

// ResolveExportTargetInfo contains information about the target of a substitution
// that is being resolved in a blueprint export.
type ResolveExportTargetInfo struct {
	ResolveFor        ResolveForStage
	PartiallyResolved *ResolvedExport
}

// ResolveInExportResult contains a resolved export
// and a list of full property paths that must be resolved during deployment.
type ResolveInExportResult struct {
	ResolvedExport  *ResolvedExport
	ResolveOnDeploy []string
}

// ResolveTargetInfo contains information about the target of a substitution
// that is being resolved in a context outside of regular substitution resolution.
type ResolveTargetInfo struct {
	ResolveFor        ResolveForStage
	PartiallyResolved interface{}
}

// ResolveResult contains a resolved mapping node and a list of full property paths
// that must be resolved during deployment.
type ResolveResult struct {
	Resolved        *bpcore.MappingNode
	ResolveOnDeploy []string
}

// ExportFieldInfo contains information about an exported field from a child blueprint
// that is used in resolving references to child blueprint exports.
type ChildExportFieldInfo struct {
	Value           *bpcore.MappingNode
	Removed         bool
	ResolveOnDeploy bool
}

// ResolveForStage is an enum that indicates the stage at which a substitution
// is being resolved for.
type ResolveForStage string

const (
	// ResolveForChangeStaging indicates that the substitution is being resolved for staging changes.
	ResolveForChangeStaging ResolveForStage = "change_staging"
	// ResolveForDeployment indicates that the substitution is being resolved for deploying a blueprint.
	ResolveForDeployment ResolveForStage = "deployment"
)

type defaultSubstitutionResolver struct {
	funcRegistry                   provider.FunctionRegistry
	resourceRegistry               resourcehelpers.Registry
	dataSourceRegistry             provider.DataSourceRegistry
	stateContainer                 state.Container
	spec                           speccore.BlueprintSpec
	params                         bpcore.BlueprintParams
	valueCache                     *bpcore.Cache[*ResolvedValue]
	dataSourceResolveResultCache   *bpcore.Cache[*ResolveInDataSourceResult]
	dataSourceDataCache            *bpcore.Cache[map[string]*bpcore.MappingNode]
	resourceCache                  *bpcore.Cache[*provider.ResolvedResource]
	resourceTemplateInputElemCache *bpcore.Cache[[]*bpcore.MappingNode]
	childExportFieldCache          *bpcore.Cache[*ChildExportFieldInfo]
	resourceStateCache             *bpcore.Cache[*state.ResourceState]
}

type Registries struct {
	FuncRegistry       provider.FunctionRegistry
	ResourceRegistry   resourcehelpers.Registry
	DataSourceRegistry provider.DataSourceRegistry
}

// NewDefaultSubstitutionResolver creates a new default implementation
// of a substitution resolver.
func NewDefaultSubstitutionResolver(
	registries *Registries,
	stateContainer state.Container,
	// The resource cache is passed down from the container as resources
	// are resolved before references to them are resolved.
	// The substitution resolver can safely assume that ordering is taken care of
	// so that a resolved resource is available when needed.
	resourceCache *bpcore.Cache[*provider.ResolvedResource],
	// The cache is used to retrieve "elem" and "i" references for resources
	// expanded from a resource template.
	resourceTemplateInputElemCache *bpcore.Cache[[]*bpcore.MappingNode],
	// A cache holding child export fields that is used to resolve references
	// to child blueprint exports.
	childExportFieldCache *bpcore.Cache[*ChildExportFieldInfo],
	spec speccore.BlueprintSpec,
	params bpcore.BlueprintParams,
) SubstitutionResolver {
	return &defaultSubstitutionResolver{
		funcRegistry:                   registries.FuncRegistry,
		resourceRegistry:               registries.ResourceRegistry,
		dataSourceRegistry:             registries.DataSourceRegistry,
		stateContainer:                 stateContainer,
		spec:                           spec,
		params:                         params,
		valueCache:                     bpcore.NewCache[*ResolvedValue](),
		dataSourceResolveResultCache:   bpcore.NewCache[*ResolveInDataSourceResult](),
		dataSourceDataCache:            bpcore.NewCache[map[string]*bpcore.MappingNode](),
		resourceCache:                  resourceCache,
		resourceTemplateInputElemCache: resourceTemplateInputElemCache,
		childExportFieldCache:          childExportFieldCache,
		resourceStateCache:             bpcore.NewCache[*state.ResourceState](),
	}
}

func (r *defaultSubstitutionResolver) ResolveResourceEach(
	ctx context.Context,
	resourceName string,
	resource *schema.Resource,
	resolveFor ResolveForStage,
) ([]*bpcore.MappingNode, error) {
	elementName := bpcore.ResourceElementID(resourceName)
	eachResolved, err := r.resolveSubstitutions(
		ctx,
		resource.Each,
		&resolveContext{
			rootElementName:        elementName,
			rootElementProperty:    "each",
			currentElementName:     elementName,
			currentElementProperty: "each",
			disallowedElementTypes: []string{"resources", "children"},
		},
	)
	if err != nil {
		return nil, err
	}

	isArray := bpcore.IsArrayMappingNode(eachResolved)
	if isArray && len(eachResolved.Items) == 0 {
		return nil, errEmptyResourceEach(elementName, resourceName)
	} else if !isArray {
		return nil, errResourceEachNotArray(elementName, resourceName, eachResolved)
	}

	return eachResolved.Items, nil
}

func (r *defaultSubstitutionResolver) ResolveInResource(
	ctx context.Context,
	resourceName string,
	resource *schema.Resource,
	resolveTargetInfo *ResolveResourceTargetInfo,
) (*ResolveInResourceResult, error) {
	resolveOnDeploy := []string{}

	elementName := bpcore.ResourceElementID(resourceName)
	resolvedResource, err := r.resolveInResource(ctx, resource, &resolveContext{
		rootElementName:    elementName,
		currentElementName: elementName,
		resolveFor:         resolveTargetInfo.ResolveFor,
		partiallyResolved:  resolveTargetInfo.PartiallyResolved,
	})
	finalErr := handleResolveError(err, &resolveOnDeploy)
	if finalErr != nil {
		return nil, finalErr
	}

	return &ResolveInResourceResult{
		ResolvedResource: resolvedResource,
		ResolveOnDeploy:  resolveOnDeploy,
	}, nil
}

func (r *defaultSubstitutionResolver) resolveInResource(
	ctx context.Context,
	resource *schema.Resource,
	resolveCtx *resolveContext,
) (*provider.ResolvedResource, error) {
	resolveOnDeployErrs := []*resolveOnDeployError{}

	resolvedDescription, err := r.resolveInResourceDescription(
		ctx,
		resource.Description,
		resolveCtx,
	)
	finalErr := handleCollectResolveError(err, &resolveOnDeployErrs)
	if finalErr != nil {
		return nil, finalErr
	}

	resolvedMetadata, err := r.resolveInResourceMetadata(ctx, resource.Metadata, resolveCtx)
	finalErr = handleCollectResolveError(err, &resolveOnDeployErrs)
	if finalErr != nil {
		return nil, finalErr
	}

	resolvedCondition, err := r.resolveInResourceCondition(
		ctx,
		resource.Condition,
		resolveContextFromParent("condition", resolveCtx),
	)
	finalErr = handleCollectResolveError(err, &resolveOnDeployErrs)
	if finalErr != nil {
		return nil, finalErr
	}

	partiallyResolvedSpec := getPartiallyResolvedResourceSpec(resolveCtx)
	resolvedSpec, err := r.resolveInMappingNode(
		ctx,
		resource.Spec,
		partiallyResolvedSpec,
		resolveContextFromParent("spec", resolveCtx),
		/* depth */ 0,
	)
	finalErr = handleCollectResolveError(err, &resolveOnDeployErrs)
	if finalErr != nil {
		return nil, finalErr
	}

	resolvedResource := &provider.ResolvedResource{
		Type:         resource.Type,
		Description:  resolvedDescription,
		Metadata:     resolvedMetadata,
		Condition:    resolvedCondition,
		Spec:         resolvedSpec,
		LinkSelector: resource.LinkSelector,
	}

	if len(resolveOnDeployErrs) > 0 {
		return resolvedResource, &resolveOnDeployErrors{
			errors: resolveOnDeployErrs,
		}
	}

	return resolvedResource, nil
}

func (r *defaultSubstitutionResolver) resolveInResourceDescription(
	ctx context.Context,
	description *substitutions.StringOrSubstitutions,
	resolveCtx *resolveContext,
) (*bpcore.MappingNode, error) {
	if resolveCtx.partiallyResolved != nil {
		partiallyResolved, ok := resolveCtx.partiallyResolved.(*provider.ResolvedResource)
		if ok &&
			partiallyResolved != nil &&
			partiallyResolved.Description != nil {
			return partiallyResolved.Description, nil
		}
	}

	return r.resolveSubstitutions(
		ctx,
		description,
		resolveContextFromParent("description", resolveCtx),
	)
}

func (r *defaultSubstitutionResolver) resolveInResourceCondition(
	ctx context.Context,
	condition *schema.Condition,
	resolveCtx *resolveContext,
) (*provider.ResolvedResourceCondition, error) {
	if condition == nil {
		return nil, nil
	}

	// Resolving a condition is all or nothing, if any part of the condition
	// cannot be resolved during change staging then the entire condition must
	// be resolved on deploy.
	if resolveCtx.partiallyResolved != nil {
		partiallyResolved, ok := resolveCtx.partiallyResolved.(*provider.ResolvedResource)
		if ok &&
			partiallyResolved != nil &&
			partiallyResolved.Condition != nil {
			return partiallyResolved.Condition, nil
		}
	}

	if condition.StringValue != nil {
		resolved, err := r.resolveSubstitutions(
			ctx,
			condition.StringValue,
			resolveCtx,
		)
		if err != nil {
			return nil, err
		}

		return &provider.ResolvedResourceCondition{
			StringValue: resolved,
		}, nil
	}

	if condition.And != nil {
		resolvedAnd, err := r.resolveInResourceConditions(
			ctx,
			condition.And,
			resolveCtx,
		)
		if err != nil {
			return nil, err
		}

		return &provider.ResolvedResourceCondition{
			And: resolvedAnd,
		}, nil
	}

	if condition.Or != nil {
		resolvedOr, err := r.resolveInResourceConditions(
			ctx,
			condition.Or,
			resolveCtx,
		)
		if err != nil {
			return nil, err
		}

		return &provider.ResolvedResourceCondition{
			Or: resolvedOr,
		}, nil
	}

	if condition.Not != nil {
		return r.resolveInResourceCondition(
			ctx,
			condition.Not,
			resolveCtx,
		)
	}

	return nil, nil
}

func (r *defaultSubstitutionResolver) resolveInResourceConditions(
	ctx context.Context,
	conditions []*schema.Condition,
	resolveCtx *resolveContext,
) ([]*provider.ResolvedResourceCondition, error) {
	resolvedConditions := []*provider.ResolvedResourceCondition{}
	resolveOnDeployErrs := []*resolveOnDeployError{}

	for _, condition := range conditions {
		resolvedCondition, err := r.resolveInResourceCondition(
			ctx,
			condition,
			resolveCtx,
		)
		finalErr := handleCollectResolveError(err, &resolveOnDeployErrs)
		if finalErr != nil {
			return nil, finalErr
		}

		resolvedConditions = append(resolvedConditions, resolvedCondition)
	}

	if len(resolveOnDeployErrs) > 0 {
		return resolvedConditions, &resolveOnDeployErrors{
			errors: resolveOnDeployErrs,
		}
	}

	return resolvedConditions, nil
}

func (r *defaultSubstitutionResolver) resolveInResourceMetadata(
	ctx context.Context,
	metadata *schema.Metadata,
	resolveCtx *resolveContext,
) (*provider.ResolvedResourceMetadata, error) {
	if metadata == nil {
		return nil, nil
	}

	resolveOnDeployErrs := []*resolveOnDeployError{}

	resolvedDisplayName, err := r.resolveInResourceMetadataDisplayName(
		ctx,
		metadata.DisplayName,
		resolveCtx,
	)
	finalErr := handleCollectResolveError(err, &resolveOnDeployErrs)
	if finalErr != nil {
		return nil, finalErr
	}

	resolvedAnnotations, err := r.resolveInResourceMetadataAnnotations(
		ctx,
		metadata.Annotations,
		resolveCtx,
	)
	finalErr = handleCollectResolveError(err, &resolveOnDeployErrs)
	if finalErr != nil {
		return nil, finalErr
	}

	resolvedCustom, err := r.resolveInResourceCustomMetadata(
		ctx,
		metadata.Custom,
		resolveCtx,
	)
	finalErr = handleCollectResolveError(err, &resolveOnDeployErrs)
	if finalErr != nil {
		return nil, finalErr
	}

	resolvedResourceMetadata := &provider.ResolvedResourceMetadata{
		DisplayName: resolvedDisplayName,
		Annotations: resolvedAnnotations,
		Labels:      metadata.Labels,
		Custom:      resolvedCustom,
	}

	if len(resolveOnDeployErrs) > 0 {
		return resolvedResourceMetadata, &resolveOnDeployErrors{
			errors: resolveOnDeployErrs,
		}
	}

	return resolvedResourceMetadata, nil
}

func (r *defaultSubstitutionResolver) resolveInResourceMetadataDisplayName(
	ctx context.Context,
	displayName *substitutions.StringOrSubstitutions,
	resolveCtx *resolveContext,
) (*bpcore.MappingNode, error) {
	if resolveCtx.partiallyResolved != nil {
		partiallyResolved, ok := resolveCtx.partiallyResolved.(*provider.ResolvedResource)
		if ok &&
			partiallyResolved != nil &&
			partiallyResolved.Metadata != nil &&
			partiallyResolved.Metadata.DisplayName != nil {
			return partiallyResolved.Metadata.DisplayName, nil
		}
	}

	return r.resolveSubstitutions(
		ctx,
		displayName,
		resolveContextFromParent("metadata.displayName", resolveCtx),
	)
}

func (r *defaultSubstitutionResolver) resolveInResourceMetadataAnnotations(
	ctx context.Context,
	annotations *schema.StringOrSubstitutionsMap,
	resolveCtx *resolveContext,
) (*bpcore.MappingNode, error) {
	if annotations == nil {
		return nil, nil
	}

	annotationsToResolve := annotations
	partiallyResolvedAnnotations := &bpcore.MappingNode{
		Fields: map[string]*bpcore.MappingNode{},
	}
	if resolveCtx.partiallyResolved != nil {
		partiallyResolved, ok := resolveCtx.partiallyResolved.(*provider.ResolvedResource)
		if ok &&
			partiallyResolved != nil &&
			partiallyResolved.Metadata != nil &&
			partiallyResolved.Metadata.Annotations != nil {

			partiallyResolvedAnnotations = partiallyResolved.Metadata.Annotations
			annotationsToResolve = filterOutResolvedAnnotations(
				partiallyResolvedAnnotations,
				annotations,
			)
		}
	}

	resolvedAnnotations, err := r.resolveInStringOrSubsMap(
		ctx,
		annotationsToResolve,
		resolveContextFromParent("metadata.annotations", resolveCtx),
	)
	if err != nil {
		// When some annotations can not be resolved at the current stage,
		// we still want to return the annotations that were resolved.
		return bpcore.MergeMaps(
			partiallyResolvedAnnotations,
			resolvedAnnotations,
		), err
	}

	return bpcore.MergeMaps(
		partiallyResolvedAnnotations,
		resolvedAnnotations,
	), err
}

func (r *defaultSubstitutionResolver) resolveInResourceCustomMetadata(
	ctx context.Context,
	custom *bpcore.MappingNode,
	resolveCtx *resolveContext,
) (*bpcore.MappingNode, error) {
	return r.resolveInMappingNode(
		ctx,
		custom,
		getPartiallyResolvedResourceCustomMetadata(resolveCtx),
		resolveContextFromParent("metadata.custom", resolveCtx),
		/* depth */ 0,
	)
}

func (r *defaultSubstitutionResolver) ResolveInDataSource(
	ctx context.Context,
	dataSourceName string,
	dataSource *schema.DataSource,
	resolveTargetInfo *ResolveDataSourceTargetInfo,
) (*ResolveInDataSourceResult, error) {
	resolveOnDeploy := []string{}

	elementName := bpcore.DataSourceElementID(dataSourceName)
	resolvedDataSource, err := r.resolveInDataSource(ctx, dataSource, &resolveContext{
		rootElementName:    elementName,
		currentElementName: elementName,
		resolveFor:         resolveTargetInfo.ResolveFor,
		partiallyResolved:  resolveTargetInfo.PartiallyResolved,
	})
	finalErr := handleResolveError(err, &resolveOnDeploy)
	if finalErr != nil {
		return nil, finalErr
	}

	return &ResolveInDataSourceResult{
		ResolvedDataSource: resolvedDataSource,
		ResolveOnDeploy:    resolveOnDeploy,
	}, nil
}

func (r *defaultSubstitutionResolver) resolveInDataSource(
	ctx context.Context,
	dataSource *schema.DataSource,
	resolveCtx *resolveContext,
) (*provider.ResolvedDataSource, error) {
	resolveOnDeployErrs := []*resolveOnDeployError{}

	resolvedDescription, err := r.resolveSubstitutions(
		ctx,
		dataSource.Description,
		resolveContextFromParent("description", resolveCtx),
	)
	finalErr := handleCollectResolveError(err, &resolveOnDeployErrs)
	if finalErr != nil {
		return nil, finalErr
	}

	resolvedMetadata, err := r.resolveInDataSourceMetadata(
		ctx,
		dataSource.DataSourceMetadata,
		resolveCtx,
	)
	finalErr = handleCollectResolveError(err, &resolveOnDeployErrs)
	if finalErr != nil {
		return nil, finalErr
	}

	resolvedDataSourceFilters, err := r.resolveInDataSourceFilters(
		ctx,
		dataSource.Filter,
		resolveCtx,
	)
	finalErr = handleCollectResolveError(err, &resolveOnDeployErrs)
	if finalErr != nil {
		return nil, finalErr
	}

	resolvedDataSourceExports, err := r.resolveInDataSourceExports(
		ctx,
		schema.GetDataSourceType(dataSource),
		dataSource.Exports,
		resolveCtx,
	)
	finalErr = handleCollectResolveError(err, &resolveOnDeployErrs)
	if finalErr != nil {
		return nil, finalErr
	}

	resolvedDataSource := &provider.ResolvedDataSource{
		Type:               dataSource.Type,
		Description:        resolvedDescription,
		DataSourceMetadata: resolvedMetadata,
		Filter:             resolvedDataSourceFilters,
		Exports:            resolvedDataSourceExports,
	}

	if len(resolveOnDeployErrs) > 0 {
		return resolvedDataSource, &resolveOnDeployErrors{
			errors: resolveOnDeployErrs,
		}
	}

	return resolvedDataSource, nil
}

func (r *defaultSubstitutionResolver) resolveInDataSourceMetadata(
	ctx context.Context,
	dataSourceMetadata *schema.DataSourceMetadata,
	resolveCtx *resolveContext,
) (*provider.ResolvedDataSourceMetadata, error) {
	if dataSourceMetadata == nil {
		return nil, nil
	}

	resolveOnDeployErrs := []*resolveOnDeployError{}

	resolvedDisplayName, err := r.resolveSubstitutions(
		ctx,
		dataSourceMetadata.DisplayName,
		resolveContextFromParent("metadata.displayName", resolveCtx),
	)
	finalErr := handleCollectResolveError(err, &resolveOnDeployErrs)
	if finalErr != nil {
		return nil, finalErr
	}

	resolvedAnnotations, err := r.resolveInStringOrSubsMap(
		ctx,
		dataSourceMetadata.Annotations,
		resolveContextFromParent("metadata.annotations", resolveCtx),
	)
	finalErr = handleCollectResolveError(err, &resolveOnDeployErrs)
	if finalErr != nil {
		return nil, finalErr
	}

	resolvedCustom, err := r.resolveInMappingNode(
		ctx,
		dataSourceMetadata.Custom,
		getPartiallyResolvedDataSourceCustomMetadata(resolveCtx),
		resolveContextFromParent("metadata.custom", resolveCtx),
		/* depth */ 0,
	)
	finalErr = handleCollectResolveError(err, &resolveOnDeployErrs)
	if finalErr != nil {
		return nil, finalErr
	}

	resolvedDataSourceMetadata := &provider.ResolvedDataSourceMetadata{
		DisplayName: resolvedDisplayName,
		Annotations: resolvedAnnotations,
		Custom:      resolvedCustom,
	}

	if len(resolveOnDeployErrs) > 0 {
		return resolvedDataSourceMetadata, &resolveOnDeployErrors{
			errors: resolveOnDeployErrs,
		}
	}

	return resolvedDataSourceMetadata, nil
}

func (r *defaultSubstitutionResolver) resolveInDataSourceFilters(
	ctx context.Context,
	filters *schema.DataSourceFilters,
	resolveCtx *resolveContext,
) (*provider.ResolvedDataSourceFilters, error) {
	if filters == nil {
		return nil, nil
	}

	resolveOnDeployErrs := []*resolveOnDeployError{}

	resolvedFilters := &provider.ResolvedDataSourceFilters{
		Filters: []*provider.ResolvedDataSourceFilter{},
	}

	for i, filter := range filters.Filters {
		filterPath := fmt.Sprintf("filters[%d]", i)
		resolvedFilter, err := r.resolveInDataSourceFilter(
			ctx,
			filter,
			resolveContextFromParent(filterPath, resolveCtx),
		)
		finalErr := handleCollectResolveError(err, &resolveOnDeployErrs)
		if finalErr != nil {
			return nil, finalErr
		}

		if resolvedFilter != nil {
			resolvedFilters.Filters = append(resolvedFilters.Filters, resolvedFilter)
		}
	}

	if len(resolveOnDeployErrs) > 0 {
		return resolvedFilters, &resolveOnDeployErrors{
			errors: resolveOnDeployErrs,
		}
	}

	return resolvedFilters, nil
}

func (r *defaultSubstitutionResolver) resolveInDataSourceFilter(
	ctx context.Context,
	filter *schema.DataSourceFilter,
	resolveCtx *resolveContext,
) (*provider.ResolvedDataSourceFilter, error) {
	if filter == nil {
		return nil, nil
	}

	resolveOnDeployErrs := []*resolveOnDeployError{}

	resolvedSearchValues, err := r.resolveStringOrSubstitutionsSlice(
		ctx,
		filter.Search.Values,
		resolveContextFromParent("filter.search", resolveCtx),
	)
	finalErr := handleCollectResolveError(err, &resolveOnDeployErrs)
	if finalErr != nil {
		return nil, finalErr
	}

	resolvedDataSourceMetadata := &provider.ResolvedDataSourceFilter{
		Field:    filter.Field,
		Operator: filter.Operator,
		Search: &provider.ResolvedDataSourceFilterSearch{
			Values: resolvedSearchValues,
		},
	}

	if len(resolveOnDeployErrs) > 0 {
		return resolvedDataSourceMetadata, &resolveOnDeployErrors{
			errors: resolveOnDeployErrs,
		}
	}

	return resolvedDataSourceMetadata, nil
}

func (r *defaultSubstitutionResolver) resolveInDataSourceExports(
	ctx context.Context,
	dataSourceType string,
	exports *schema.DataSourceFieldExportMap,
	resolveCtx *resolveContext,
) (map[string]*provider.ResolvedDataSourceFieldExport, error) {
	if exports == nil {
		return nil, nil
	}

	resolvedExports := map[string]*provider.ResolvedDataSourceFieldExport{}
	resolveOnDeployErrs := []*resolveOnDeployError{}

	if exports.ExportAll {
		// When ExportAll is true, it indicates that all fields of the data source
		// should be exported.
		// This is indicated by `exports: "*"` being set in a data source
		// instead of a map of exports in the current blueprint.
		return r.resolveDataSourceExportsFromSpec(
			ctx,
			dataSourceType,
		)
	}

	for exportName, export := range exports.Values {

		resolvedDescription, err := r.resolveSubstitutions(
			ctx,
			export.Description,
			resolveContextFromParent(
				fmt.Sprintf("exports.%s.description", exportName),
				resolveCtx,
			),
		)
		finalErr := handleCollectResolveError(err, &resolveOnDeployErrs)
		if finalErr != nil {
			return nil, finalErr
		}

		resolvedExports[exportName] = &provider.ResolvedDataSourceFieldExport{
			Type:        export.Type,
			AliasFor:    export.AliasFor,
			Description: resolvedDescription,
		}
	}

	if len(resolveOnDeployErrs) > 0 {
		return resolvedExports, &resolveOnDeployErrors{
			errors: resolveOnDeployErrs,
		}
	}

	return resolvedExports, nil
}

func (r *defaultSubstitutionResolver) resolveDataSourceExportsFromSpec(
	ctx context.Context,
	dataSourceType string,
) (map[string]*provider.ResolvedDataSourceFieldExport, error) {
	specDefOutput, err := r.dataSourceRegistry.GetSpecDefinition(
		ctx,
		dataSourceType,
		&provider.DataSourceGetSpecDefinitionInput{
			ProviderContext: provider.NewProviderContextFromParams(
				provider.ExtractProviderFromItemType(dataSourceType),
				r.params,
			),
		},
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get data source spec definition for %s: %w",
			dataSourceType,
			err,
		)
	}

	if specDefOutput == nil ||
		specDefOutput.SpecDefinition == nil ||
		len(specDefOutput.SpecDefinition.Fields) == 0 {
		return nil, fmt.Errorf(
			"data source spec definition for %s is either not set or has no fields",
			dataSourceType,
		)
	}

	resolvedExports := map[string]*provider.ResolvedDataSourceFieldExport{}
	for exportName, exportSchema := range specDefOutput.SpecDefinition.Fields {
		resolvedExports[exportName] = &provider.ResolvedDataSourceFieldExport{
			// There is one-to-one mapping between the data source export schema
			// field type and the data source field export type.
			Type: &schema.DataSourceFieldTypeWrapper{
				Value: schema.DataSourceFieldType(exportSchema.Type),
			},
		}
	}

	return resolvedExports, nil
}

func (r *defaultSubstitutionResolver) ResolveInMappingNode(
	ctx context.Context,
	currentElementName string,
	mappingNode *bpcore.MappingNode,
	resolveTargetInfo *ResolveMappingNodeTargetInfo,
) (*ResolveInMappingNodeResult, error) {
	resolveOnDeploy := []string{}

	resolved, err := r.resolveInMappingNode(
		ctx,
		mappingNode,
		resolveTargetInfo.PartiallyResolved,
		&resolveContext{
			rootElementName:    currentElementName,
			currentElementName: currentElementName,
			resolveFor:         resolveTargetInfo.ResolveFor,
			partiallyResolved:  resolveTargetInfo.PartiallyResolved,
		},
		/* depth */ 0,
	)
	finalErr := handleResolveError(err, &resolveOnDeploy)
	if finalErr != nil {
		return nil, finalErr
	}

	return &ResolveInMappingNodeResult{
		ResolvedMappingNode: resolved,
		ResolveOnDeploy:     resolveOnDeploy,
	}, nil
}

func (r *defaultSubstitutionResolver) resolveInMappingNode(
	ctx context.Context,
	mappingNode *bpcore.MappingNode,
	// A partially resolved mapping node is passed down to combine with the resolved
	// mapping node.
	// This differs from resolveContext.partiallyResolved which holds the full partially
	// resolved element in a blueprint (e.g. a resource or data source).
	partiallyResolved *bpcore.MappingNode,
	resolveCtx *resolveContext,
	depth int,
) (*bpcore.MappingNode, error) {
	// Depth counting starts at 0.
	if mappingNode == nil || depth >= bpcore.MappingNodeMaxTraverseDepth {
		return nil, nil
	}

	resolveOnDeployErrs := []*resolveOnDeployError{}

	if mappingNode.Scalar != nil {
		return mappingNode, nil
	}

	var resolvedMappingNode *bpcore.MappingNode

	if mappingNode.StringWithSubstitutions != nil {
		if getStringWithSubstitutions(partiallyResolved) != nil {
			return partiallyResolved, nil
		}

		resolvedSub, err := r.resolveSubstitutions(
			ctx,
			mappingNode.StringWithSubstitutions,
			resolveCtx,
		)
		finalErr := handleCollectResolveError(err, &resolveOnDeployErrs)
		if finalErr != nil {
			return nil, finalErr
		}

		resolvedMappingNode = resolvedSub
	}

	if mappingNode.Items != nil {
		resolvedItems, err := r.resolveInMappingNodeSlice(
			ctx,
			mappingNode.Items,
			getItems(partiallyResolved),
			resolveCtx,
			depth,
		)
		finalErr := handleCollectResolveError(err, &resolveOnDeployErrs)
		if finalErr != nil {
			return nil, finalErr
		}

		resolvedMappingNode = &bpcore.MappingNode{
			Items: resolvedItems,
		}
	}

	if mappingNode.Fields != nil {
		resolvedFields, err := r.resolveInMappingNodeFields(
			ctx,
			mappingNode.Fields,
			getFields(partiallyResolved),
			resolveCtx,
			depth,
		)
		finalErr := handleCollectResolveError(err, &resolveOnDeployErrs)
		if finalErr != nil {
			return nil, finalErr
		}

		resolvedMappingNode = &bpcore.MappingNode{
			Fields: resolvedFields,
		}
	}

	if len(resolveOnDeployErrs) > 0 {
		return resolvedMappingNode, &resolveOnDeployErrors{
			errors: resolveOnDeployErrs,
		}
	}

	return resolvedMappingNode, nil
}

func (r *defaultSubstitutionResolver) resolveInMappingNodeSlice(
	ctx context.Context,
	items []*bpcore.MappingNode,
	partiallyResolvedItems []*bpcore.MappingNode,
	resolveCtx *resolveContext,
	depth int,
) ([]*bpcore.MappingNode, error) {
	resolvedItems := []*bpcore.MappingNode{}
	resolveOnDeployErrs := []*resolveOnDeployError{}

	for i, item := range items {
		partiallyResolvedItem := getItem(partiallyResolvedItems, i)
		propertyPath := fmt.Sprintf("%s[%d]", resolveCtx.currentElementProperty, i)
		resolvedItem, err := r.resolveInMappingNode(
			ctx,
			item,
			partiallyResolvedItem,
			resolveContextFromParent(propertyPath, resolveCtx),
			depth+1,
		)
		finalErr := handleCollectResolveError(err, &resolveOnDeployErrs)
		if finalErr != nil {
			return nil, finalErr
		}

		resolvedItems = append(resolvedItems, resolvedItem)
	}

	if len(resolveOnDeployErrs) > 0 {
		return resolvedItems, &resolveOnDeployErrors{
			errors: resolveOnDeployErrs,
		}
	}

	return resolvedItems, nil
}

func (r *defaultSubstitutionResolver) resolveInMappingNodeFields(
	ctx context.Context,
	fields map[string]*bpcore.MappingNode,
	partiallyResolvedFields map[string]*bpcore.MappingNode,
	resolveCtx *resolveContext,
	depth int,
) (map[string]*bpcore.MappingNode, error) {
	resolvedFields := map[string]*bpcore.MappingNode{}
	resolveOnDeployErrs := []*resolveOnDeployError{}

	for fieldName, field := range fields {
		partiallyResolvedField := getField(partiallyResolvedFields, fieldName)
		propertyPath := fmt.Sprintf("%s[\"%s\"]", resolveCtx.currentElementProperty, fieldName)
		resolvedField, err := r.resolveInMappingNode(
			ctx,
			field,
			partiallyResolvedField,
			resolveContextFromParent(propertyPath, resolveCtx),
			depth+1,
		)
		finalErr := handleCollectResolveError(err, &resolveOnDeployErrs)
		if finalErr != nil {
			return nil, finalErr
		}

		resolvedFields[fieldName] = resolvedField
	}

	if len(resolveOnDeployErrs) > 0 {
		return resolvedFields, &resolveOnDeployErrors{
			errors: resolveOnDeployErrs,
		}
	}

	return resolvedFields, nil
}

func (r *defaultSubstitutionResolver) ResolveInValue(
	ctx context.Context,
	valueName string,
	value *schema.Value,
	resolveTargetInfo *ResolveValueTargetInfo,
) (*ResolveInValueResult, error) {
	resolveOnDeploy := []string{}

	elementName := bpcore.ValueElementID(valueName)
	resolvedValue, err := r.resolveInValue(ctx, value, &resolveContext{
		rootElementName:    elementName,
		currentElementName: elementName,
		resolveFor:         resolveTargetInfo.ResolveFor,
		partiallyResolved:  resolveTargetInfo.PartiallyResolved,
	})
	if err != nil {
		finalErr := handleResolveError(err, &resolveOnDeploy)
		if finalErr != nil {
			return nil, finalErr
		}
	}

	return &ResolveInValueResult{
		ResolvedValue:   resolvedValue,
		ResolveOnDeploy: resolveOnDeploy,
	}, nil
}

func (r *defaultSubstitutionResolver) resolveInValue(
	ctx context.Context,
	value *schema.Value,
	resolveCtx *resolveContext,
) (*ResolvedValue, error) {
	resolveOnDeployErrs := []*resolveOnDeployError{}

	resolvedContent, err := r.resolveInMappingNode(
		ctx,
		value.Value,
		// Values are ephemeral and do not have a partially resolved state.
		/* partiallyResolved */
		nil,
		resolveContextFromParent("value", resolveCtx),
		/* depth */ 0,
	)
	if err != nil {
		finalErr := handleCollectResolveError(err, &resolveOnDeployErrs)
		if finalErr != nil {
			return nil, finalErr
		}
	}

	resolvedDescription, err := r.resolveSubstitutions(
		ctx,
		value.Description,
		resolveContextFromParent("description", resolveCtx),
	)
	if err != nil {
		finalErr := handleCollectResolveError(err, &resolveOnDeployErrs)
		if finalErr != nil {
			return nil, finalErr
		}
	}

	resolvedValue := &ResolvedValue{
		Type:        value.Type,
		Value:       resolvedContent,
		Description: resolvedDescription,
		Secret:      value.Secret,
	}

	if len(resolveOnDeployErrs) > 0 {
		return resolvedValue, &resolveOnDeployErrors{
			errors: resolveOnDeployErrs,
		}
	}

	return resolvedValue, nil
}

func (r *defaultSubstitutionResolver) ResolveInInclude(
	ctx context.Context,
	includeName string,
	include *schema.Include,
	resolveTargetInfo *ResolveIncludeTargetInfo,
) (*ResolveInIncludeResult, error) {
	resolveOnDeploy := []string{}

	elementName := bpcore.ChildElementID(includeName)
	resolvedInclude, err := r.resolveInInclude(ctx, include, &resolveContext{
		rootElementName:    elementName,
		currentElementName: elementName,
		resolveFor:         resolveTargetInfo.ResolveFor,
		partiallyResolved:  resolveTargetInfo.PartiallyResolved,
	})
	if err != nil {
		finalErr := handleResolveError(err, &resolveOnDeploy)
		if finalErr != nil {
			return nil, finalErr
		}
	}

	return &ResolveInIncludeResult{
		ResolvedInclude: resolvedInclude,
		ResolveOnDeploy: resolveOnDeploy,
	}, nil
}

func (r *defaultSubstitutionResolver) resolveInInclude(
	ctx context.Context,
	include *schema.Include,
	resolveCtx *resolveContext,
) (*ResolvedInclude, error) {
	resolveOnDeployErrs := []*resolveOnDeployError{}

	resolvedPath, err := r.resolveSubstitutions(
		ctx,
		include.Path,
		resolveContextFromParent("path", resolveCtx),
	)
	if err != nil {
		finalErr := handleCollectResolveError(err, &resolveOnDeployErrs)
		if finalErr != nil {
			return nil, finalErr
		}
	}

	resolvedVars, err := r.resolveInMappingNode(
		ctx,
		include.Variables,
		getPartiallyResolvedIncludeVariables(resolveCtx),
		resolveContextFromParent("variables", resolveCtx),
		/* depth */ 0,
	)
	if err != nil {
		finalErr := handleCollectResolveError(err, &resolveOnDeployErrs)
		if finalErr != nil {
			return nil, finalErr
		}
	}

	resolvedMetadata, err := r.resolveInMappingNode(
		ctx,
		include.Metadata,
		getPartiallyResolvedIncludeMetadata(resolveCtx),
		resolveContextFromParent("metadata", resolveCtx),
		/* depth */ 0,
	)
	if err != nil {
		finalErr := handleCollectResolveError(err, &resolveOnDeployErrs)
		if finalErr != nil {
			return nil, finalErr
		}
	}

	resolvedDescription, err := r.resolveSubstitutions(
		ctx,
		include.Description,
		resolveContextFromParent("description", resolveCtx),
	)
	if err != nil {
		finalErr := handleCollectResolveError(err, &resolveOnDeployErrs)
		if finalErr != nil {
			return nil, finalErr
		}
	}

	resolvedInclude := &ResolvedInclude{
		Path:        resolvedPath,
		Variables:   resolvedVars,
		Metadata:    resolvedMetadata,
		Description: resolvedDescription,
	}

	if len(resolveOnDeployErrs) > 0 {
		return resolvedInclude, &resolveOnDeployErrors{
			errors: resolveOnDeployErrs,
		}
	}

	return resolvedInclude, nil
}

func (r *defaultSubstitutionResolver) ResolveInExport(
	ctx context.Context,
	exportName string,
	export *schema.Export,
	resolveTargetInfo *ResolveExportTargetInfo,
) (*ResolveInExportResult, error) {
	resolveOnDeploy := []string{}

	elementName := bpcore.ExportElementID(exportName)
	resolvedExport, err := r.resolveInExport(ctx, export, &resolveContext{
		rootElementName:    elementName,
		currentElementName: elementName,
		resolveFor:         resolveTargetInfo.ResolveFor,
		partiallyResolved:  resolveTargetInfo.PartiallyResolved,
	})
	if err != nil {
		finalErr := handleResolveError(err, &resolveOnDeploy)
		if finalErr != nil {
			return nil, finalErr
		}
	}

	return &ResolveInExportResult{
		ResolvedExport:  resolvedExport,
		ResolveOnDeploy: resolveOnDeploy,
	}, nil
}

func (r *defaultSubstitutionResolver) resolveInExport(
	ctx context.Context,
	export *schema.Export,
	resolveCtx *resolveContext,
) (*ResolvedExport, error) {
	resolveOnDeployErrs := []*resolveOnDeployError{}

	resolvedDescription, err := r.resolveSubstitutions(
		ctx,
		export.Description,
		resolveContextFromParent("description", resolveCtx),
	)
	if err != nil {
		finalErr := handleCollectResolveError(err, &resolveOnDeployErrs)
		if finalErr != nil {
			return nil, finalErr
		}
	}

	resolvedExport := &ResolvedExport{
		Type:        export.Type,
		Field:       export.Field,
		Description: resolvedDescription,
	}

	if len(resolveOnDeployErrs) > 0 {
		return resolvedExport, &resolveOnDeployErrors{
			errors: resolveOnDeployErrs,
		}
	}

	return resolvedExport, nil
}

func (r *defaultSubstitutionResolver) resolveInStringOrSubsMap(
	ctx context.Context,
	stringOrSubsMap *schema.StringOrSubstitutionsMap,
	resolveCtx *resolveContext,
) (*bpcore.MappingNode, error) {
	if stringOrSubsMap == nil {
		return nil, nil
	}

	resolvedMapping := map[string]*bpcore.MappingNode{}
	resolveOnDeployErrs := []*resolveOnDeployError{}
	for key, value := range stringOrSubsMap.Values {
		resolvedValue, err := r.resolveSubstitutions(
			ctx,
			value,
			resolveCtx,
		)
		if err != nil {
			finalErr := handleCollectResolveError(err, &resolveOnDeployErrs)
			if finalErr != nil {
				return nil, finalErr
			}
		}

		resolvedMapping[key] = resolvedValue
	}

	resolvedMappingNode := &bpcore.MappingNode{
		Fields: resolvedMapping,
	}

	if len(resolveOnDeployErrs) > 0 {
		return resolvedMappingNode, &resolveOnDeployErrors{
			errors: resolveOnDeployErrs,
		}
	}

	return resolvedMappingNode, nil
}

func (r *defaultSubstitutionResolver) resolveStringOrSubstitutionsSlice(
	ctx context.Context,
	stringOrSubsSlice []*substitutions.StringOrSubstitutions,
	resolveCtx *resolveContext,
) ([]*bpcore.MappingNode, error) {
	resolvedSlice := []*bpcore.MappingNode{}

	resolveOnDeployErrs := []*resolveOnDeployError{}

	for _, stringOrSubs := range stringOrSubsSlice {
		resolvedValue, err := r.resolveSubstitutions(
			ctx,
			stringOrSubs,
			resolveCtx,
		)
		if err != nil {
			finalErr := handleCollectResolveError(err, &resolveOnDeployErrs)
			if finalErr != nil {
				return nil, finalErr
			}
		}

		resolvedSlice = append(resolvedSlice, resolvedValue)
	}

	if len(resolveOnDeployErrs) > 0 {
		return resolvedSlice, &resolveOnDeployErrors{
			errors: resolveOnDeployErrs,
		}
	}

	return resolvedSlice, nil
}

func (r *defaultSubstitutionResolver) resolveSubstitutions(
	ctx context.Context,
	stringOrSubs *substitutions.StringOrSubstitutions,
	resolveCtx *resolveContext,
) (*bpcore.MappingNode, error) {
	if stringOrSubs == nil || stringOrSubs.Values == nil {
		return nil, nil
	}

	isStringInterpolation := len(stringOrSubs.Values) > 1
	if !isStringInterpolation {
		// A set of dependencies needed to make function calls with a call stack
		// scoped to the current ${..} substitution.
		functionCallDeps := createFunctionCallDependencies(
			r.funcRegistry,
			r.params,
			stringOrSubs.Values[0].SourceMeta,
		)
		return r.resolveSubstitution(
			ctx,
			stringOrSubs.Values[0],
			functionCallDeps,
			resolveCtx,
		)
	}

	sb := &strings.Builder{}
	for _, value := range stringOrSubs.Values {
		// A set of dependencies needed to make function calls with a call stack
		// scoped to the current substitution.
		functionCallDeps := createFunctionCallDependencies(
			r.funcRegistry,
			r.params,
			value.SourceMeta,
		)
		resolvedValue, err := r.resolveSubstitution(
			ctx,
			value,
			functionCallDeps,
			resolveCtx,
		)
		if err != nil {
			return nil, err
		}

		if stringValue, err := resolvedValueToString(resolvedValue); err == nil {
			sb.WriteString(stringValue)
		} else {
			return nil, errInvalidInterpolationSubType(resolveCtx.currentElementName, resolvedValue)
		}
	}

	resolvedStr := sb.String()
	return &bpcore.MappingNode{
		Scalar: &bpcore.ScalarValue{
			StringValue: &resolvedStr,
		},
	}, nil
}

func (r *defaultSubstitutionResolver) ResolveSubstitution(
	ctx context.Context,
	value *substitutions.StringOrSubstitution,
	inElementName string,
	inElementProperty string,
	resolveTargetInfo *ResolveTargetInfo,
) (*ResolveResult, error) {

	resolveOnDeploy := []string{}

	functionCallDeps := createFunctionCallDependencies(
		r.funcRegistry,
		r.params,
		value.SourceMeta,
	)
	mappingNode, err := r.resolveSubstitution(
		ctx,
		value,
		functionCallDeps,
		&resolveContext{
			rootElementName:        inElementName,
			rootElementProperty:    inElementProperty,
			currentElementName:     inElementName,
			currentElementProperty: inElementProperty,
			resolveFor:             resolveTargetInfo.ResolveFor,
			partiallyResolved:      resolveTargetInfo.PartiallyResolved,
		},
	)
	if err != nil {
		finalErr := handleResolveError(err, &resolveOnDeploy)
		if finalErr != nil {
			return nil, finalErr
		}
	}

	return &ResolveResult{
		Resolved:        mappingNode,
		ResolveOnDeploy: resolveOnDeploy,
	}, nil
}

func (r *defaultSubstitutionResolver) resolveSubstitution(
	ctx context.Context,
	value *substitutions.StringOrSubstitution,
	functionCallDeps *functionCallDependencies,
	resolveCtx *resolveContext,
) (*bpcore.MappingNode, error) {
	if value.StringValue != nil {
		return r.resolveScalarFromSubStringValue(
			value,
			resolveCtx,
		), nil
	}

	if value.SubstitutionValue != nil {
		return r.resolveSubstitutionValue(
			ctx,
			value.SubstitutionValue,
			functionCallDeps,
			resolveCtx,
		)
	}

	return nil, errEmptySubstitutionValue(resolveCtx.currentElementName)
}

func (r *defaultSubstitutionResolver) resolveScalarFromSubStringValue(
	value *substitutions.StringOrSubstitution,
	resolveCtx *resolveContext,
) *bpcore.MappingNode {
	if value.StringValue == nil {
		return nil
	}

	stringVal := *value.StringValue

	if !resolveCtx.isAnnotation {
		return bpcore.MappingNodeFromString(stringVal)
	}

	return bpcore.ParseScalarMappingNode(stringVal)
}

func (r *defaultSubstitutionResolver) resolveSubstitutionValue(
	ctx context.Context,
	substitutionValue *substitutions.Substitution,
	functionCallDeps *functionCallDependencies,
	resolveCtx *resolveContext,
) (*bpcore.MappingNode, error) {

	resolvedScalar := r.resolveScalar(substitutionValue)
	if resolvedScalar != nil {
		return resolvedScalar, nil
	}

	if substitutionValue.Variable != nil {
		return r.resolveVariable(resolveCtx.currentElementName, substitutionValue.Variable)
	}

	if substitutionValue.ValueReference != nil {
		return r.resolveValue(
			ctx,
			substitutionValue.ValueReference,
			resolveCtx,
		)
	}

	if substitutionValue.ElemReference != nil {
		return r.resolveElemReference(
			substitutionValue.ElemReference,
			resolveCtx,
		)
	}

	if substitutionValue.ElemIndexReference != nil {
		return r.resolveElemIndexReference(
			resolveCtx,
		)
	}

	if substitutionValue.DataSourceProperty != nil {
		return r.resolveDataSourceProperty(
			ctx,
			substitutionValue.DataSourceProperty,
			resolveCtx,
		)
	}

	if substitutionValue.ResourceProperty != nil {
		if slices.Contains(resolveCtx.disallowedElementTypes, "resources") {
			return nil, errDisallowedElementType(
				resolveCtx.rootElementName,
				resolveCtx.rootElementProperty,
				"resource",
			)
		}

		return r.resolveResourceProperty(
			ctx,
			substitutionValue.ResourceProperty,
			resolveCtx,
		)
	}

	if substitutionValue.Child != nil {
		if slices.Contains(resolveCtx.disallowedElementTypes, "children") {
			return nil, errDisallowedElementType(
				resolveCtx.rootElementName,
				resolveCtx.rootElementProperty,
				"child",
			)
		}

		return r.resolveChildReference(
			ctx,
			substitutionValue.Child,
			resolveCtx,
		)
	}

	if substitutionValue.Function != nil {
		funcCallOutput, err := r.resolveFunctionCall(
			ctx,
			substitutionValue.Function,
			functionCallDeps,
			resolveCtx,
		)
		if err != nil {
			return nil, err
		}

		if funcCallOutput.value != nil {
			return funcCallOutput.value, nil
		}

		return nil, errHigherOrderFunctionNotSupported(
			resolveCtx.currentElementName,
			string(substitutionValue.Function.FunctionName),
		)
	}

	return nil, errEmptySubstitutionValue(resolveCtx.currentElementName)
}

func (r *defaultSubstitutionResolver) resolveScalar(
	substitutionValue *substitutions.Substitution,
) *bpcore.MappingNode {
	if substitutionValue.StringValue != nil {
		return &bpcore.MappingNode{
			Scalar: &bpcore.ScalarValue{
				StringValue: substitutionValue.StringValue,
			},
		}
	}

	if substitutionValue.IntValue != nil {
		intVal := int(*substitutionValue.IntValue)
		return &bpcore.MappingNode{
			Scalar: &bpcore.ScalarValue{
				IntValue: &intVal,
			},
		}
	}

	if substitutionValue.FloatValue != nil {
		return &bpcore.MappingNode{
			Scalar: &bpcore.ScalarValue{
				FloatValue: substitutionValue.FloatValue,
			},
		}
	}

	if substitutionValue.BoolValue != nil {
		return &bpcore.MappingNode{
			Scalar: &bpcore.ScalarValue{
				BoolValue: substitutionValue.BoolValue,
			},
		}
	}

	return nil
}

func (r *defaultSubstitutionResolver) resolveVariable(
	elementName string,
	variable *substitutions.SubstitutionVariable,
) (*bpcore.MappingNode, error) {

	varValue := r.params.BlueprintVariable(variable.VariableName)
	if varValue == nil {
		specVar := getVariable(variable.VariableName, r.spec.Schema())
		if specVar == nil {
			return nil, errMissingVariable(elementName, variable.VariableName)
		}

		if specVar.Default != nil {
			return &bpcore.MappingNode{
				Scalar: specVar.Default,
			}, nil
		}
	}

	return &bpcore.MappingNode{
		Scalar: varValue,
	}, nil
}

func (r *defaultSubstitutionResolver) resolveValue(
	ctx context.Context,
	value *substitutions.SubstitutionValueReference,
	resolveCtx *resolveContext,
) (*bpcore.MappingNode, error) {
	cached, hasValue := r.valueCache.Get(value.ValueName)
	if hasValue {
		return cached.Value, nil
	}

	computed, err := r.computeValue(ctx, value, resolveCtx)
	if err != nil {
		return nil, err
	}

	r.valueCache.Set(value.ValueName, computed.ResolvedValue)

	if len(computed.ResolveOnDeploy) > 0 {
		return computed.ResolvedValue.Value, errMustResolveOnDeployMultiple(
			append(
				computed.ResolveOnDeploy,
				// Ensure that the current element property is included in the list of paths
				// to be resolved on deploy.
				// If the referenced value needs to be resolved on deploy, then the
				// location where it is referenced must also be resolved on deploy.
				bpcore.ElementPropertyPath(
					resolveCtx.currentElementName,
					resolveCtx.currentElementProperty,
				),
			),
		)
	}

	return computed.ResolvedValue.Value, nil
}

func (r *defaultSubstitutionResolver) computeValue(
	ctx context.Context,
	value *substitutions.SubstitutionValueReference,
	resolveCtx *resolveContext,
) (*ResolveInValueResult, error) {
	resolveOnDeploy := []string{}

	valueSpec := getValue(value.ValueName, r.spec.Schema())
	if valueSpec == nil {
		return nil, errMissingValue(resolveCtx.currentElementName, value.ValueName)
	}

	elementID := bpcore.ValueElementID(value.ValueName)
	resolvedValue, err := r.resolveInValue(
		ctx,
		valueSpec,
		resolveContextForCurrentElement(elementID, resolveCtx),
	)
	if err != nil {
		finalErr := handleResolveError(err, &resolveOnDeploy)
		if finalErr != nil {
			return nil, finalErr
		}
	}

	return &ResolveInValueResult{
		ResolvedValue:   resolvedValue,
		ResolveOnDeploy: resolveOnDeploy,
	}, nil
}

func (r *defaultSubstitutionResolver) resolveElemReference(
	elemRef *substitutions.SubstitutionElemReference,
	resolveCtx *resolveContext,
) (*bpcore.MappingNode, error) {
	resourceName := resourceNameFromElementID(resolveCtx.currentElementName)
	resourceNameParts, couldBeTemplate := extractResourceTemplateNameParts(resourceName)
	if !couldBeTemplate {
		return nil, errResourceNotTemplate(resolveCtx.currentElementName, resourceName)
	}

	items, hasItems := r.resourceTemplateInputElemCache.Get(
		resourceNameParts.templateName,
	)
	if !hasItems {
		return nil, errMissingResourceTemplateInputElements(
			resolveCtx.currentElementName,
			resourceNameParts.templateName,
		)
	}

	if len(items) <= resourceNameParts.index {
		// If the resource name ends with _\d+ and the index is out of bounds, then
		// it is likely a resource that just happens to end with a number and can not
		// be considered a template.
		// These errors are primarily to catch user errors, index out of bounds error
		// for a resource template would be an error in the change staging logic
		// that wouldn't make sense to the user.
		return nil, errResourceNotTemplate(
			resolveCtx.currentElementName,
			resourceName,
		)
	}

	return getPathValueFromMappingNode(
		items[resourceNameParts.index],
		elemRef.Path,
		elemRef,
		resolveCtx,
		/* mappingNodeStartsAfter */ 0,
		errMissingCurrentElementProperty,
	)
}

func (r *defaultSubstitutionResolver) resolveElemIndexReference(
	resolveCtx *resolveContext,
) (*bpcore.MappingNode, error) {
	resourceName := resourceNameFromElementID(resolveCtx.currentElementName)
	resourceNameParts, couldBeTemplate := extractResourceTemplateNameParts(resourceName)
	if !couldBeTemplate {
		return nil, errResourceNotTemplate(resolveCtx.currentElementName, resourceName)
	}

	return &bpcore.MappingNode{
		Scalar: &bpcore.ScalarValue{
			IntValue: &resourceNameParts.index,
		},
	}, nil
}

func (r *defaultSubstitutionResolver) resolveDataSourceProperty(
	ctx context.Context,
	dataSourceProperty *substitutions.SubstitutionDataSourceProperty,
	resolveCtx *resolveContext,
) (*bpcore.MappingNode, error) {
	resolvedDataSource, err := r.resolveDataSource(ctx, dataSourceProperty, resolveCtx)
	if err != nil {
		return nil, err
	}

	cached, hasValue := r.dataSourceDataCache.Get(dataSourceProperty.DataSourceName)
	if hasValue {
		return extractDataSourceProperty(resolveCtx.currentElementName, resolvedDataSource, cached, dataSourceProperty)
	}

	providerNamespace := provider.ExtractProviderFromItemType(resolvedDataSource.Type.Value)

	dataOutput, err := r.dataSourceRegistry.Fetch(
		ctx,
		resolvedDataSource.Type.Value,
		&provider.DataSourceFetchInput{
			DataSourceWithResolvedSubs: resolvedDataSource,
			ProviderContext: provider.NewProviderContextFromParams(
				providerNamespace,
				r.params,
			),
		},
	)
	if err != nil {
		return nil, err
	}

	r.dataSourceDataCache.Set(dataSourceProperty.DataSourceName, dataOutput.Data)

	return extractDataSourceProperty(
		resolveCtx.currentElementName,
		resolvedDataSource,
		dataOutput.Data,
		dataSourceProperty,
	)
}

func extractDataSourceProperty(
	parentElementName string,
	resolvedDataSource *provider.ResolvedDataSource,
	data map[string]*bpcore.MappingNode,
	prop *substitutions.SubstitutionDataSourceProperty,
) (*bpcore.MappingNode, error) {
	if data == nil {
		return nil, errEmptyDataSourceData(parentElementName, prop.DataSourceName)
	}

	value, hasValue := getDataSourceFieldByPropOrAlias(data, prop.FieldName, resolvedDataSource)
	if !hasValue {
		return nil, errMissingDataSourceProperty(parentElementName, prop.DataSourceName, prop.FieldName)
	}

	finalValue := value
	if prop.PrimitiveArrIndex != nil {
		if value.Items == nil {
			return nil, errDataSourcePropNotArray(parentElementName, prop.DataSourceName, prop.FieldName)
		}

		if int(*prop.PrimitiveArrIndex) >= len(value.Items) {
			return nil, errDataSourcePropArrayIndexOutOfBounds(
				parentElementName,
				prop.DataSourceName,
				prop.FieldName,
				int(*prop.PrimitiveArrIndex),
			)
		}

		finalValue = value.Items[*prop.PrimitiveArrIndex]
	}

	return finalValue, nil
}

func (r *defaultSubstitutionResolver) resolveDataSource(
	ctx context.Context,
	prop *substitutions.SubstitutionDataSourceProperty,
	resolveCtx *resolveContext,
) (*provider.ResolvedDataSource, error) {
	cached, hasDataSource := r.dataSourceResolveResultCache.Get(prop.DataSourceName)
	if hasDataSource {
		return expandResolveDataSourceResultWithError(cached, resolveCtx)
	}

	computed, err := r.computeDataSource(ctx, prop, resolveCtx)
	if err != nil {
		return nil, err
	}

	r.dataSourceResolveResultCache.Set(prop.DataSourceName, computed)

	return expandResolveDataSourceResultWithError(computed, resolveCtx)
}

func (r *defaultSubstitutionResolver) computeDataSource(
	ctx context.Context,
	prop *substitutions.SubstitutionDataSourceProperty,
	resolveCtx *resolveContext,
) (*ResolveInDataSourceResult, error) {
	resolveOnDeploy := []string{}

	dataSourceSpec := getDataSource(prop.DataSourceName, r.spec.Schema())
	if dataSourceSpec == nil {
		return nil, errMissingDataSource(resolveCtx.currentElementName, prop.DataSourceName)
	}

	elementID := bpcore.DataSourceElementID(prop.DataSourceName)
	resolvedDataSource, err := r.resolveInDataSource(
		ctx,
		dataSourceSpec,
		resolveContextForCurrentElement(elementID, resolveCtx),
	)
	if err != nil {
		finalErr := handleResolveError(err, &resolveOnDeploy)
		if finalErr != nil {
			return nil, finalErr
		}
	}

	return &ResolveInDataSourceResult{
		ResolvedDataSource: resolvedDataSource,
		ResolveOnDeploy:    resolveOnDeploy,
	}, nil
}

func (r *defaultSubstitutionResolver) resolveResourceProperty(
	ctx context.Context,
	resourceProperty *substitutions.SubstitutionResourceProperty,
	resolveCtx *resolveContext,
) (*bpcore.MappingNode, error) {

	if len(resourceProperty.Path) == 0 ||
		(len(resourceProperty.Path) > 1 && resourceProperty.Path[0].FieldName == "spec") {

		return r.resolveResourceSpecProperty(
			ctx,
			resourceProperty,
			resolveCtx,
		)
	}

	if len(resourceProperty.Path) > 1 && resourceProperty.Path[0].FieldName == "metadata" {
		return r.resolveResourceMetadataProperty(
			resourceProperty,
			resolveCtx,
		)
	}

	return nil, errInvalidResourcePropertyPath(resolveCtx.currentElementName, resourceProperty)
}

func (r *defaultSubstitutionResolver) resolveResourceSpecProperty(
	ctx context.Context,
	prop *substitutions.SubstitutionResourceProperty,
	resolveCtx *resolveContext,
) (*bpcore.MappingNode, error) {

	blueprintResource := r.spec.ResourceSchema(prop.ResourceName)
	if blueprintResource == nil {
		return nil, errReferencedResourceMissing(resolveCtx.currentElementName, prop.ResourceName)
	}

	resourceType := string(blueprintResource.Type.Value)
	providerNamespace := provider.ExtractProviderFromItemType(resourceType)
	output, err := r.resourceRegistry.GetSpecDefinition(
		ctx,
		resourceType,
		&provider.ResourceGetSpecDefinitionInput{
			ProviderContext: provider.NewProviderContextFromParams(
				providerNamespace,
				r.params,
			),
		},
	)
	if err != nil {
		return nil, err
	}

	if output.SpecDefinition == nil || output.SpecDefinition.Schema == nil {
		return nil, errMissingResourceSpecDefinition(
			resolveCtx.currentElementName,
			prop.ResourceName,
			resourceType,
		)
	}

	definition, err := getResourceSpecPropertyDefinition(
		output.SpecDefinition,
		prop,
		resourceType,
		resolveCtx,
	)
	if err != nil {
		return nil, err
	}

	if definition.Computed && resolveCtx.resolveFor == ResolveForChangeStaging {
		return nil, errMustResolveOnDeploy(
			resolveCtx.currentElementName,
			resolveCtx.currentElementProperty,
		)
	}

	// During change staging, the resolved value will contain all user-provided
	// values that can be resolved without requiring the resource to be deployed.
	resourceName := getFinalResourceName(prop)
	resource, hasResource := r.resourceCache.Get(resourceName)
	if !hasResource {
		return nil, errResourceNotResolved(resolveCtx.currentElementName, prop.ResourceName)
	}

	resolved, err := getResourceSpecPropertyValue(resource, prop, resolveCtx)
	runErr, isRunErr := err.(*errors.RunError)
	if err != nil && isRunErr &&
		runErr.ReasonCode == ErrorReasonCodeMissingResourceSpecProperty {
		return r.resolveResourceSpecPropertyFromStateOrDefault(ctx, prop, definition, err, resolveCtx)
	} else if err != nil {
		return nil, err
	}

	return resolved, nil
}

func (r *defaultSubstitutionResolver) resolveResourceSpecPropertyFromStateOrDefault(
	ctx context.Context,
	prop *substitutions.SubstitutionResourceProperty,
	definition *provider.ResourceDefinitionsSchema,
	originalErr error,
	resolveCtx *resolveContext,
) (*bpcore.MappingNode, error) {
	if resolveCtx.resolveFor == ResolveForDeployment {
		resourceName := getFinalResourceName(prop)
		resourceState, hasResourceState := r.resourceStateCache.Get(resourceName)
		if !hasResourceState {
			instanceID, err := bpcore.BlueprintInstanceIDFromContext(ctx)
			if err != nil {
				return nil, err
			}

			resources := r.stateContainer.Resources()
			freshResourceState, err := resources.GetByName(ctx, instanceID, resourceName)
			if err != nil {
				if state.IsResourceNotFound(err) || state.IsInstanceNotFound(err) {
					return getResourceSpecDefaultValueFromDefinition(err, definition)
				}
				return nil, err
			}

			r.resourceStateCache.Set(resourceName, &freshResourceState)
			resourceState = &freshResourceState
		}

		return getResourceSpecPropertyFromState(
			resourceState,
			prop,
			definition,
			resolveCtx,
		)
	}

	return getResourceSpecDefaultValueFromDefinition(originalErr, definition)
}

func (r *defaultSubstitutionResolver) resolveResourceMetadataProperty(
	prop *substitutions.SubstitutionResourceProperty,
	resolveCtx *resolveContext,
) (*bpcore.MappingNode, error) {

	blueprintResource := r.spec.ResourceSchema(prop.ResourceName)
	if blueprintResource == nil {
		return nil, errReferencedResourceMissing(resolveCtx.currentElementName, prop.ResourceName)
	}

	// During change staging, the resolved value will contain all user-provided
	// values that can be resolved without requiring the resource to be deployed.
	resourceName := getFinalResourceName(prop)
	resource, hasResource := r.resourceCache.Get(resourceName)
	if !hasResource {
		return nil, errResourceNotResolved(resolveCtx.currentElementName, prop.ResourceName)
	}

	resolved, err := getResourceMetadataPropertyValue(resource, prop, resolveCtx)
	if err != nil {
		return nil, err
	}

	return resolved, nil
}

func (r *defaultSubstitutionResolver) resolveChildReference(
	ctx context.Context,
	childReference *substitutions.SubstitutionChild,
	resolveCtx *resolveContext,
) (*bpcore.MappingNode, error) {

	if len(childReference.Path) == 0 {
		return nil, errEmptyChildPath(resolveCtx.currentElementName, childReference.ChildName)
	}

	cacheKey := substitutions.RenderFieldPath(
		childReference.ChildName,
		childReference.Path[0].FieldName,
	)
	// The child export cache is derived from change staging a child blueprint so can reliably
	// be used to resolve child exports during change staging.
	childExportInfo, hasChildExportInfo := r.childExportFieldCache.Get(cacheKey)
	if hasChildExportInfo &&
		!childExportInfo.Removed &&
		!childExportInfo.ResolveOnDeploy {
		return getChildExportProperty(childExportInfo.Value, childReference, resolveCtx)
	}

	if resolveCtx.resolveFor == ResolveForChangeStaging {
		// If we can't get the child export from the cache and we are resolving for change
		// staging, then trying to source the child export from the state would lead to
		// incorrect reporting of the changes that will occur during deployment as the
		// state may not be up-to-date.
		return nil, errMustResolveOnDeploy(
			resolveCtx.currentElementName,
			resolveCtx.currentElementProperty,
		)
	}

	instanceID, err := bpcore.BlueprintInstanceIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	children := r.stateContainer.Children()
	childState, err := children.Get(ctx, instanceID, childReference.ChildName)
	if err != nil {
		return nil, err
	}

	exportState := getChildExport(childReference.Path[0].FieldName, &childState)
	if exportState == nil {
		return nil, errMissingChildExport(
			resolveCtx.currentElementName,
			childReference.ChildName,
			childReference,
		)
	}

	return getChildExportProperty(exportState, childReference, resolveCtx)
}

func (r *defaultSubstitutionResolver) resolveFunctionCall(
	ctx context.Context,
	function *substitutions.SubstitutionFunctionExpr,
	functionCallDeps *functionCallDependencies,
	resolveCtx *resolveContext,
) (*resolvedFunctionCallValue, error) {

	hasFunction, err := functionCallDeps.scopedRegistry.HasFunction(
		ctx,
		string(function.FunctionName),
	)
	if err != nil {
		return nil, err
	}

	if !hasFunction {
		return nil, errMissingFunction(resolveCtx.currentElementName, string(function.FunctionName))
	}

	// Link functions can only be resolved during deployment
	// as referenced resources need to be have been deployed
	// to be able to resolve the link.
	// errMustResolveOnDeploy should be handled at a per-substitution level
	// to be collected in a list of elements that need to be resolved
	// during deployment.
	if function.FunctionName == substitutions.SubstitutionFunctionLink &&
		resolveCtx.resolveFor == ResolveForChangeStaging {
		return nil, errMustResolveOnDeploy(
			resolveCtx.currentElementName,
			resolveCtx.currentElementProperty,
		)
	}

	resolvedArgs := []*resolvedFunctionCallValue{}
	for index, arg := range function.Arguments {
		if arg.Value != nil {
			resolvedArg, err := r.resolveFunctionCallArg(
				ctx,
				arg,
				functionCallDeps,
				resolveCtx,
			)
			if err != nil {
				return nil, err
			}

			resolvedArgs = append(resolvedArgs, resolvedArg)
		} else {
			return nil, createEmptyArgError(
				resolveCtx.currentElementName,
				string(function.FunctionName),
				arg,
				index,
			)
		}
	}

	args := functionCallDeps.callCtx.NewCallArgs(
		core.Map(resolvedArgs, transformValueForFunctionCall)...,
	)
	output, err := functionCallDeps.scopedRegistry.Call(
		ctx,
		string(function.FunctionName),
		&provider.FunctionCallInput{
			Arguments:   args,
			CallContext: functionCallDeps.callCtx,
		},
	)
	if err != nil {
		return nil, err
	}

	if output.ResponseData == nil && output.FunctionInfo.FunctionName == "" {
		return nil, errEmptyFunctionOutput(
			resolveCtx.currentElementName,
			string(function.FunctionName),
		)
	}

	if output.ResponseData != nil {
		return &resolvedFunctionCallValue{
			value: GoValueToMappingNode(output.ResponseData),
		}, nil
	}

	return &resolvedFunctionCallValue{
		function: output.FunctionInfo,
	}, nil
}

func (r *defaultSubstitutionResolver) resolveFunctionCallArg(
	ctx context.Context,
	arg *substitutions.SubstitutionFunctionArg,
	functionCallDeps *functionCallDependencies,
	resolveCtx *resolveContext,
) (*resolvedFunctionCallValue, error) {
	// Function calls have to be treated differently from other substitutions,
	// a function call can return a value or a partially applied function,
	// to keep the surface area of accounting for different types of outputs
	// as small as possible, this logic is kept within function call resolution.
	if arg.Value.Function != nil {
		funcOutput, err := r.resolveFunctionCall(
			ctx,
			arg.Value.Function,
			functionCallDeps,
			resolveCtx,
		)
		if err != nil {
			return nil, err
		}

		return funcOutput, nil
	}

	resolvedArg, err := r.resolveSubstitutionValue(
		ctx,
		arg.Value,
		functionCallDeps,
		resolveCtx,
	)
	if err != nil {
		return nil, err
	}

	return &resolvedFunctionCallValue{
		value: resolvedArg,
	}, nil
}
