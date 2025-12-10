package container

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/includes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/refgraph"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/blueprint/subengine"
)

type childBlueprintLoadResult struct {
	includeName    string
	childContainer BlueprintContainer
	childState     *state.InstanceState
	childParams    core.BlueprintParams
}

type childBlueprintLoadInput struct {
	parentInstanceID       string
	parentInstanceTreePath string
	instanceTreePath       string
	includeTreePath        string
	node                   *refgraph.ReferenceChainNode
	resolveFor             subengine.ResolveForStage
	logger                 core.Logger
}

func loadChildBlueprint(
	ctx context.Context,
	input *childBlueprintLoadInput,
	substitutionResolver IncludeSubstitutionResolver,
	childResolver includes.ChildResolver,
	createChildBlueprintLoader ChildBlueprintLoaderFactory,
	stateContainer state.Container,
	paramOverrides core.BlueprintParams,
) (*childBlueprintLoadResult, error) {

	includeName := core.ToLogicalChildName(input.node.ElementName)

	input.logger.Debug("resolving include definition for child blueprint")
	resolvedInclude, err := resolveIncludeForChildBlueprint(
		ctx,
		input.node,
		includeName,
		input.resolveFor,
		substitutionResolver,
	)
	if err != nil {
		return nil, err
	}

	input.logger.Debug(
		"resolving child blueprint document",
		core.StringLogField("path", core.StringValue(resolvedInclude.Path)),
	)
	childBlueprintInfo, err := childResolver.Resolve(ctx, includeName, resolvedInclude, paramOverrides)
	if err != nil {
		return nil, err
	}

	// Derive the child blueprint's directory for resolving nested relative paths.
	childBlueprintDir := deriveChildBlueprintDir(childBlueprintInfo, resolvedInclude, paramOverrides)

	childParams := paramOverrides.
		WithBlueprintVariables(
			extractIncludeVariables(resolvedInclude),
			/* keepExisting */ false,
		).
		WithContextVariables(
			createContextVarsForChildBlueprint(
				input.parentInstanceID,
				input.parentInstanceTreePath,
				input.includeTreePath,
				childBlueprintDir,
			),
			/* keepExisting */ true,
		)

	childLoader := createChildBlueprintLoader(
		/* derivedFromTemplate */ []string{},
		/* resourceTemplates */ map[string]string{},
	)

	var childContainer BlueprintContainer
	if childBlueprintInfo.AbsolutePath != nil {
		childContainer, err = childLoader.Load(ctx, *childBlueprintInfo.AbsolutePath, childParams)
		if err != nil {
			return nil, err
		}
	} else {
		format, err := extractChildBlueprintFormat(includeName, resolvedInclude)
		if err != nil {
			return nil, err
		}

		childContainer, err = childLoader.LoadString(
			ctx,
			*childBlueprintInfo.BlueprintSource,
			format,
			childParams,
		)
		if err != nil {
			return nil, err
		}
	}

	input.logger.Debug(
		"loading child blueprint state",
		core.StringLogField("instanceID", input.parentInstanceID),
	)
	childState, err := getChildState(ctx, input.parentInstanceID, includeName, stateContainer)
	if err != nil {
		return nil, err
	}

	if hasBlueprintCycle(input.parentInstanceTreePath, childState.InstanceID) {
		input.logger.Debug(
			"detected blueprint cycle",
			core.StringLogField("instanceID", childState.InstanceID),
			core.StringLogField("parentInstanceTreePath", input.parentInstanceTreePath),
		)
		return nil, errBlueprintCycleDetected(
			includeName,
			input.parentInstanceTreePath,
			childState.InstanceID,
		)
	}

	return &childBlueprintLoadResult{
		childContainer: childContainer,
		childState:     childState,
		childParams:    childParams,
		includeName:    includeName,
	}, nil
}

// deriveChildBlueprintDir derives the directory of the child blueprint
// from the resolved include path. This is used to update the __blueprintDir
// context variable so that nested child blueprints can resolve relative paths.
func deriveChildBlueprintDir(
	childBlueprintInfo *includes.ChildBlueprintInfo,
	resolvedInclude *subengine.ResolvedInclude,
	params core.BlueprintParams,
) string {
	// If the resolver provided an absolute path, use its directory.
	if childBlueprintInfo.AbsolutePath != nil {
		return filepath.Dir(*childBlueprintInfo.AbsolutePath)
	}

	// Otherwise, derive the directory from the include path.
	includePath := core.StringValue(resolvedInclude.Path)
	if includePath == "" {
		return ""
	}

	// If the include path is absolute, use its directory.
	if filepath.IsAbs(includePath) {
		return filepath.Dir(includePath)
	}

	// If the include path is relative, resolve it against the parent's blueprint directory.
	if params != nil {
		parentDir := params.ContextVariable(BlueprintDirectoryContextVar)
		if parentDir != nil && parentDir.StringValue != nil && *parentDir.StringValue != "" {
			resolvedPath := filepath.Join(*parentDir.StringValue, includePath)
			return filepath.Dir(resolvedPath)
		}
	}

	return ""
}

func getChildState(
	ctx context.Context,
	parentInstanceID string,
	includeName string,
	stateContainer state.Container,
) (*state.InstanceState, error) {
	children := stateContainer.Children()
	childState, err := children.Get(ctx, parentInstanceID, includeName)
	if err != nil {
		if !state.IsInstanceNotFound(err) {
			return nil, err
		} else {
			// Change staging includes describing the planned state for a new blueprint,
			// an empty instance ID will be used to indicate that the blueprint instance is new.
			// Deployment includes creating new blueprint instances, so an instance ID will be
			// assigned to the new blueprint instance later.
			return &state.InstanceState{
				InstanceID: "",
			}, nil
		}
	}

	return &childState, nil
}

func resolveIncludeForChildBlueprint(
	ctx context.Context,
	node *refgraph.ReferenceChainNode,
	includeName string,
	resolveFor subengine.ResolveForStage,
	substitutionResolver IncludeSubstitutionResolver,
) (*subengine.ResolvedInclude, error) {
	include, isInclude := node.Element.(*schema.Include)
	if !isInclude {
		return nil, fmt.Errorf("child blueprint node is not an include")
	}

	resolvedIncludeResult, err := substitutionResolver.ResolveInInclude(
		ctx,
		includeName,
		include,
		&subengine.ResolveIncludeTargetInfo{
			ResolveFor: resolveFor,
		},
	)
	if err != nil {
		return nil, err
	}

	actionText := "changes can only be staged"
	if resolveFor == subengine.ResolveForDeployment {
		actionText = "the child blueprint can only be deployed"
	}

	if len(resolvedIncludeResult.ResolveOnDeploy) > 0 {
		return nil, fmt.Errorf(
			"child blueprint include %q has unresolved substitutions, "+
				"%s for child blueprints when "+
				"all the information required to fetch and load the blueprint is available",
			node.ElementName,
			actionText,
		)
	}

	return resolvedIncludeResult.ResolvedInclude, nil
}
