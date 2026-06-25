package docgen

import (
	"fmt"
	"sort"
)

// Marks each link that is activated by a reference
// placed at a wiring-slot field (ActivatesLinkOnReference) rather than by a
// linkSelector. A link A::B qualifies when resource A has at least one wiring
// slot, placing a reference to a B-typed resource there activates the link.
func correlateReferenceActivation(
	resources []*PluginDocsResource,
	links []*PluginDocsLink,
) {
	slotsByResourceType := map[string][]string{}
	for _, resource := range resources {
		if resource == nil || resource.Specification == nil {
			continue
		}
		slots := []string{}
		collectActivatingSlots(resource.Specification.Schema, "", &slots)
		if len(slots) > 0 {
			sort.Strings(slots)
			slotsByResourceType[resource.Type] = slots
		}
	}

	for _, link := range links {
		linkTypeParts, err := extractLinkTypeInfo(link.Type)
		if err != nil {
			continue
		}
		slots, ok := slotsByResourceType[linkTypeParts.resourceTypeA]
		if !ok {
			continue
		}
		link.ReferenceActivation = &PluginDocsLinkReferenceActivation{
			ResourceType: linkTypeParts.resourceTypeA,
			FieldPaths:   slots,
		}
	}
}

func collectActivatingSlots(
	schema *PluginDocResourceSpecSchema,
	path string,
	out *[]string,
) {
	if schema == nil {
		return
	}

	if schema.ActivatesLinkOnReference && path != "" {
		*out = append(*out, path)
	}

	for attrName, attr := range schema.Attributes {
		attrPath := attrName
		if path != "" {
			attrPath = fmt.Sprintf("%s.%s", path, attrName)
		}
		collectActivatingSlots(attr, attrPath, out)
	}

	collectActivatingSlots(schema.Items, fmt.Sprintf("%s[]", path), out)

	for _, oneOf := range schema.OneOf {
		collectActivatingSlots(oneOf, path, out)
	}
}
