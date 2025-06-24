package plugintestutils

import (
	"fmt"
	"slices"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/stretchr/testify/assert"
)

// TagFieldNames holds the field names used to extract key and value from tag slice
// MappingNodes.
// By default, if empty or the values are not set when passed to CompareTags,
// it will use "key" and "value" as the field names.
type TagFieldNames struct {
	KeyFieldName   string
	ValueFieldName string
}

// CompareTags compares two slices of tag MappingNodes in an order-independent way.
// It extracts key-value pairs represented as an object with
// attributes holding a key and a value that are both expected to be strings.
//
// If nil is passed for tagFieldNames, it defaults to using "key" and "value" as the field names.
func CompareTags(
	t assert.TestingT,
	expectedTags, actualTags []*core.MappingNode,
	tagFieldNames *TagFieldNames,
) {
	finalTagFieldNames := createFinalTagFields(tagFieldNames)

	assert.Equal(t, len(expectedTags), len(actualTags), "tag count should match")

	compareExpectedTags := tagsToStringSlice(expectedTags, finalTagFieldNames)
	compareActualTags := tagsToStringSlice(actualTags, finalTagFieldNames)

	slices.Sort(compareExpectedTags)
	slices.Sort(compareActualTags)

	assert.Equal(t, compareExpectedTags, compareActualTags, "tags should match regardless of order")
}

func tagsToStringSlice(tags []*core.MappingNode, tagFieldNames *TagFieldNames) []string {
	orderedTags := make([]string, len(tags))
	for i, tag := range tags {
		key := tag.Fields[tagFieldNames.KeyFieldName]
		value := tag.Fields[tagFieldNames.ValueFieldName]
		orderedTags[i] = fmt.Sprintf(
			"%s=%s",
			core.StringValue(key),
			core.StringValue(value),
		)
	}
	return orderedTags
}

func createFinalTagFields(tagFieldNames *TagFieldNames) *TagFieldNames {
	if tagFieldNames == nil {
		return &TagFieldNames{
			KeyFieldName:   "key",
			ValueFieldName: "value",
		}
	}

	newTagFieldNames := &TagFieldNames{
		KeyFieldName:   tagFieldNames.KeyFieldName,
		ValueFieldName: tagFieldNames.ValueFieldName,
	}

	if tagFieldNames.KeyFieldName == "" {
		newTagFieldNames.KeyFieldName = "key"
	}
	if tagFieldNames.ValueFieldName == "" {
		newTagFieldNames.ValueFieldName = "value"
	}

	return newTagFieldNames
}
