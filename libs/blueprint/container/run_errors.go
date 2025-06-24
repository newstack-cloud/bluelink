package container

import (
	"fmt"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/errors"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

const (
	// ErrorReasonCodeMissingChildBlueprintPath
	// is provided when the reason for an error
	// during deployment or change staging is due to
	// a missing path to a child blueprint in an include.
	ErrorReasonCodeMissingChildBlueprintPath errors.ErrorReasonCode = "missing_child_blueprint_path"
	// ErrorReasonCodeEmptyChildBlueprintPath
	// is provided when the reason for an error
	// during deployment or change staging is due to
	// an empty path to a child blueprint in an include.
	ErrorReasonCodeEmptyChildBlueprintPath errors.ErrorReasonCode = "empty_child_blueprint_path"
	// ErrorReasonCodeResourceTemplateLinkLengthMismatch
	// is provided when the reason for an error
	// during deployment or change staging is due to
	// a mismatch in the length of the resolved items
	// for linked resource templates.
	ErrorReasonCodeResourceTemplateLinkLengthMismatch errors.ErrorReasonCode = "resource_template_link_length_mismatch"
	// ErrorReasonCodeBlueprintCycleDetected
	// is provided when the reason for an error
	// during deployment or change staging is due to
	// a cyclic blueprint inclusion detected.
	ErrorReasonCodeBlueprintCycleDetected errors.ErrorReasonCode = "blueprint_cycle_detected"
	// ErrorReasonCodeMaxBlueprintDepthExceeded
	// is provided when the reason for an error
	// during deployment or change staging is due to
	// the maximum blueprint depth being exceeded.
	ErrorReasonCodeMaxBlueprintDepthExceeded errors.ErrorReasonCode = "max_blueprint_depth_exceeded"
	// ErrorReasonCodeRemovedResourceHasDependents
	// is provided when the reason for an error
	// during deployment is due to a resource that is
	// to be removed having dependents that will not be
	// removed or recreated.
	ErrorReasonCodeRemovedResourceHasDependents errors.ErrorReasonCode = "removed_resource_has_dependents"
	// ErrorReasonCodeRemovedChildHasDependents
	// is provided when the reason for an error
	// during deployment is due to a child blueprint that is
	// to be removed having dependents that will not be
	// removed or recreated.
	ErrorReasonCodeRemovedChildHasDependents errors.ErrorReasonCode = "removed_child_has_dependents"
	// ErrorReasonCodeResourceNotFoundInState
	// is provided when the reason for an error
	// during deployment or change staging is due to
	// a resource not being found in the state of a blueprint instance.
	ErrorReasonCodeResourceNotFoundInState errors.ErrorReasonCode = "resource_not_found_in_state"
	// ErrorReasonCodeLinkNotFoundInState
	// is provided when the reason for an error
	// during deployment or change staging is due to
	// a link not being found in the state of a blueprint instance.
	ErrorReasonCodeLinkNotFoundInState errors.ErrorReasonCode = "link_not_found_in_state"
	// ErrorReasonCodeChildNotFoundInState
	// is provided when the reason for an error
	// during deployment or change staging is due to
	// a child not being found in the state of a blueprint instance.
	ErrorReasonCodeChildNotFoundInState errors.ErrorReasonCode = "child_not_found_in_state"
	// ErrorReasonCodeInvalidLogicalLinkName
	// is provided when the reason for an error
	// during deployment or change staging is due to
	// an invalid logical link name being provided when preparing to deploy
	// or destroy a link between resources.
	ErrorReasonCodeInvalidLogicalLinkName errors.ErrorReasonCode = "invalid_logical_link_name"
	// ErrorReasonCodeDeployMissingInstanceID
	// is provided when the reason for an error
	// during deployment is due to a missing instance ID
	// when deploying changes that modify existing resources or child blueprints.
	ErrorReasonCodeDeployMissingInstanceID errors.ErrorReasonCode = "deploy_missing_instance_id"
	// ErrorReasonCodeDeployMissingResourceChanges
	// is provided when the reason for an error
	// during deployment is due to missing changes for a resource
	// that is being deployed.
	ErrorReasonCodeDeployMissingResourceChanges errors.ErrorReasonCode = "deploy_missing_resource_changes"
	// ErrorReasonCodeDeployMissingPartiallyResolvedResource
	// is provided when the reason for an error
	// during deployment is due to a missing partially resolved resource
	// for a resource that is being deployed.
	ErrorReasonCodeDeployMissingPartiallyResolvedResource errors.ErrorReasonCode = "deploy_missing_partially_resolved_resource"
	// ErrorReasonCodeDriftDetected
	// is provided when the reason for an error
	// during deployment or change staging is due to
	// drift being detected in resources.
	ErrorReasonCodeDriftDetected errors.ErrorReasonCode = "drift_detected"
	// ErrorReasonCodeDeployInstanceIDAndNameProvided
	// is provided when the reason for an error
	// when destroying an instance or change staging is due to both an instance ID
	// and a name being provided.
	// This does not apply to deployments, for which an instance ID and name can be
	// provided and will both be used in the process of saving new blueprint instances.
	ErrorReasonCodeInstanceIDAndNameProvided errors.ErrorReasonCode = "instance_id_and_name_provided"
	// ErrorReasonCodeMissingNameForNewInstance
	// is provided when the reason for an error
	// during deployment is due to a missing name for a new instance
	// that is being created.
	// All blueprint instances require a user-defined name
	// (or one generated by the caller system based on its context).
	ErrorReasonCodeMissingNameForNewInstance errors.ErrorReasonCode = "missing_name_for_new_instance"
	// ErrorReasonCodeChildBlueprintError
	// is provided when the reason for an error
	// during deployment or change staging is due to
	// an error in a child blueprint.
	// This is used to wrap errors that occur in child blueprints
	// that are not run errors.
	ErrorReasonCodeChildBlueprintError errors.ErrorReasonCode = "child_blueprint_error"
)

func errMissingChildBlueprintPath(includeName string) error {
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeMissingChildBlueprintPath,
		Err:        fmt.Errorf("[include.%s]: child blueprint path is missing for include", includeName),
	}
}

func errEmptyChildBlueprintPath(includeName string) error {
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeEmptyChildBlueprintPath,
		Err:        fmt.Errorf("[include.%s]: child blueprint path is empty for include", includeName),
	}
}

func errResourceTemplateLinkLengthMismatch(
	linkFrom string,
	linkTo string,
) error {
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeResourceTemplateLinkLengthMismatch,
		Err: fmt.Errorf(
			"resource template %s has a link to resource template %s with a different input length, links between resource templates can only be made "+
				"when the resolved items list from the `each` property of both templates is of the same length",
			linkFrom,
			linkTo,
		),
	}
}

func errBlueprintCycleDetected(
	includeName string,
	instanceTreePath string,
	cyclicInstanceID string,
) error {
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeBlueprintCycleDetected,
		Err: fmt.Errorf(
			"[include.%s]: cyclic blueprint inclusion detected, instance %q is an ancestor of the "+
				"current blueprint as shown in the instance tree path: %q",
			includeName,
			cyclicInstanceID,
			instanceTreePath,
		),
	}
}

func errMaxBlueprintDepthExceeded(
	instanceTreePath string,
	maxDepth int,
) error {
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeMaxBlueprintDepthExceeded,
		Err: fmt.Errorf(
			"max blueprint depth exceeded, instance tree path: %q, "+
				"only %d levels of blueprint includes are allowed",
			instanceTreePath,
			maxDepth,
		),
	}
}

func errResourceToBeRemovedHasDependents(
	resourceName string,
	dependents *CollectedElements,
) error {
	dependentsList := formatElements(dependents)
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeRemovedResourceHasDependents,
		Err: fmt.Errorf(
			"resource %q cannot be removed because it has dependents "+
				"that will not be removed or recreated: %v",
			resourceName,
			strings.Join(dependentsList, ", "),
		),
	}
}

func errChildToBeRemovedHasDependents(
	childName string,
	dependents *CollectedElements,
) error {
	dependentsList := formatElements(dependents)
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeRemovedChildHasDependents,
		Err: fmt.Errorf(
			"child blueprint %q cannot be removed because it has dependents "+
				"that will not be removed or recreated: %v",
			childName,
			strings.Join(dependentsList, ", "),
		),
	}
}

func errResourceNotFoundInState(instanceID string, resourceName string) error {
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeResourceNotFoundInState,
		Err: fmt.Errorf(
			"resource %q not found in state for blueprint instance %q",
			resourceName,
			instanceID,
		),
	}
}

func errLinkNotFoundInState(instanceID string, linkName string) error {
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeLinkNotFoundInState,
		Err: fmt.Errorf(
			"link %q not found in state for blueprint instance %q",
			linkName,
			instanceID,
		),
	}
}

func errChildNotFoundInState(instanceID string, childName string) error {
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeChildNotFoundInState,
		Err: fmt.Errorf(
			"child %q not found in state for blueprint instance %q",
			childName,
			instanceID,
		),
	}
}

func errInvalidLogicalLinkName(linkName string, instanceID string) error {
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeInvalidLogicalLinkName,
		Err: fmt.Errorf(
			"invalid logical link name %q has been provided in "+
				"blueprint instance %q, logical link names "+
				"must be of the form `{resourceA}::{resourceB}`",
			linkName,
			instanceID,
		),
	}
}

func errInstanceIDRequiredForChanges() error {
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeDeployMissingInstanceID,
		Err: fmt.Errorf(
			"an instance ID is required for deployments where " +
				"the provided change set contains modifications " +
				"to existing resources or child blueprints",
		),
	}
}

func errInstanceIDAndNameProvided() error {
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeInstanceIDAndNameProvided,
		Err: fmt.Errorf(
			"an instance ID and name cannot be provided at the same time for this purpose, " +
				"only one of them can be used for staging changes or destroying a blueprint instance",
		),
	}
}

func errMissingNameForNewInstance() error {
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeMissingNameForNewInstance,
		Err: fmt.Errorf(
			"an instance name is required for new blueprint instances, " +
				"the name must be provided in the deploy input",
		),
	}
}

func errMissingResourceChanges(resourceName string) error {
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeDeployMissingResourceChanges,
		Err: fmt.Errorf(
			"no changes provided for resource %q, at "+
				"least one change is required in the provided set of changes",
			resourceName,
		),
	}
}

func errMissingPartiallyResolvedResource(resourceName string) error {
	return &errors.RunError{
		ReasonCode: ErrorReasonCodeDeployMissingPartiallyResolvedResource,
		Err: fmt.Errorf(
			"resource %q is missing from the partially resolved resources, "+
				"a partially resolved resource must be provided "+
				"for each resource in the given set of changes",
			resourceName,
		),
	}
}

func errDriftDetected(
	driftResults map[string]*state.ResourceDriftState,
) error {
	var driftedResources []string
	for resourceID := range driftResults {
		driftedResources = append(driftedResources, resourceID)
	}

	return &errors.RunError{
		ReasonCode: ErrorReasonCodeDriftDetected,
		Err: fmt.Errorf(
			"drift detected in resources: %v. This must be resolved before you can deploy a new update, "+
				"you can load the state to see the drift details",
			strings.Join(driftedResources, ", "),
		),
	}
}

func formatElements(elements *CollectedElements) []string {
	var formatted []string

	for _, resource := range elements.Resources {
		formatted = append(formatted, core.ResourceElementID(resource.ResourceName))
	}

	for _, child := range elements.Children {
		formatted = append(formatted, core.ChildElementID(child.ChildName))
	}

	return formatted
}

func wrapErrorForChildContext(
	err error,
	params core.BlueprintParams,
) error {
	includeTreePath := getIncludeTreePath(params, "")
	if includeTreePath == "" {
		return err
	}

	runErr, isRunErr := err.(*errors.RunError)
	if isRunErr {
		// Be sure not to wrap errors that already have a child blueprint path,
		// we want to surface the most precise location of the error.
		if runErr.ChildBlueprintPath != "" {
			return err
		}

		return &errors.RunError{
			ReasonCode:         runErr.ReasonCode,
			Err:                runErr.Err,
			ChildErrors:        runErr.ChildErrors,
			ChildBlueprintPath: includeTreePath,
		}
	}
	return &errors.RunError{
		ReasonCode:         ErrorReasonCodeChildBlueprintError,
		Err:                err,
		ChildBlueprintPath: includeTreePath,
	}
}

func getDeploymentErrorSpecificMessage(err error, fallbackMessage string) string {
	message := fallbackMessage

	runErr, isRunErr := err.(*errors.RunError)
	if isRunErr &&
		(runErr.ReasonCode == ErrorReasonCodeRemovedResourceHasDependents ||
			runErr.ReasonCode == ErrorReasonCodeRemovedChildHasDependents) {
		message = runErr.Error()
	}

	return message
}
