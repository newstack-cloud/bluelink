package transformerv1

import (
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/errors"
)

const (
	// ErrorReasonCodeCrossAbstractConcreteBoundaryLink indicates that
	// a link crosses the abstract-concrete boundary in the blueprint.
	ErrorReasonCodeCrossAbstractConcreteBoundaryLink errors.ErrorReasonCode = "cross_abstract_concrete_boundary_link"
	// ErrorReasonCodeNoSuchAbstractLinkDefinition indicates that a link references
	// an abstract link type for which there is no definition in the transformer plugin.
	ErrorReasonCodeNoSuchAbstractLinkDefinition errors.ErrorReasonCode = "no_such_abstract_link_definition"
	// ErrorReasonCodeMissingAnnotationResource indicates that an annotation resource
	// referenced in the abstract link definition is missing from the blueprint.
	ErrorReasonCodeMissingAnnotationResource errors.ErrorReasonCode = "missing_annotation_resource"
)

func errAbstractResourceTypeNotFound(abstractResourceType string) error {
	return fmt.Errorf(
		"abstract resource type not implemented in transformer plugin: %s",
		abstractResourceType,
	)
}

func errAbstractLinkNotFound(linkType string, transformName string) error {
	return fmt.Errorf(
		"no abstract link definition found for link type: %s in transformer plugin: %s",
		linkType,
		transformName,
	)
}
