package plugintestutils

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/stretchr/testify/suite"
)

type CompareTagsSuite struct {
	suite.Suite
}

func (s *CompareTagsSuite) Test_compare_matching_tags() {
	tag1 := &core.MappingNode{
		Fields: map[string]*core.MappingNode{
			"key":   core.MappingNodeFromString("myKey"),
			"value": core.MappingNodeFromString("myValue"),
		},
	}
	tag2 := &core.MappingNode{
		Fields: map[string]*core.MappingNode{
			"key":   core.MappingNodeFromString("myKey2"),
			"value": core.MappingNodeFromString("myValue2"),
		},
	}

	tags1 := []*core.MappingNode{
		tag1,
		tag2,
	}

	tags2 := []*core.MappingNode{
		tag2,
		tag1,
	}

	CompareTags(s.T(), tags1, tags2, nil)
}

func (s *CompareTagsSuite) Test_compare_matching_tags_empty_tag_field_names() {
	tag1 := &core.MappingNode{
		Fields: map[string]*core.MappingNode{
			"key":   core.MappingNodeFromString("myKey10"),
			"value": core.MappingNodeFromString("myValue10"),
		},
	}
	tag2 := &core.MappingNode{
		Fields: map[string]*core.MappingNode{
			"key":   core.MappingNodeFromString("myKey20"),
			"value": core.MappingNodeFromString("myValue20"),
		},
	}

	tags1 := []*core.MappingNode{
		tag1,
		tag2,
	}

	tags2 := []*core.MappingNode{
		tag2,
		tag1,
	}

	CompareTags(s.T(), tags1, tags2, &TagFieldNames{})
}

func (s *CompareTagsSuite) Test_compare_tags_with_custom_field_names() {
	tag1 := &core.MappingNode{
		Fields: map[string]*core.MappingNode{
			"customKey":   core.MappingNodeFromString("myKey"),
			"customValue": core.MappingNodeFromString("myValue"),
		},
	}
	tag2 := &core.MappingNode{
		Fields: map[string]*core.MappingNode{
			"customKey":   core.MappingNodeFromString("myKey2"),
			"customValue": core.MappingNodeFromString("myValue2"),
		},
	}

	tags1 := []*core.MappingNode{
		tag1,
		tag2,
	}

	tags2 := []*core.MappingNode{
		tag2,
		tag1,
	}

	customFieldNames := &TagFieldNames{
		KeyFieldName:   "customKey",
		ValueFieldName: "customValue",
	}

	CompareTags(s.T(), tags1, tags2, customFieldNames)
}

func TestCompareTagsSuite(t *testing.T) {
	suite.Run(t, new(CompareTagsSuite))
}
