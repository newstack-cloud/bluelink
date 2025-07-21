package pluginutils

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/stretchr/testify/suite"
)

type ResourceHelpersTestSuite struct {
	suite.Suite
}

func (s *ResourceHelpersTestSuite) Test_get_resource_name() {
	resourceName := GetResourceName(&provider.ResourceInfo{
		ResourceName: "testResource",
	})
	s.Assert().Equal("testResource", resourceName)

	resourceName = GetResourceName(nil)
	s.Assert().Equal("unknown", resourceName)
}

func (s *ResourceHelpersTestSuite) Test_get_instance_id() {
	instanceID := GetInstanceID(&provider.ResourceInfo{
		InstanceID: "testInstance",
	})
	s.Assert().Equal("testInstance", instanceID)

	instanceID = GetInstanceID(nil)
	s.Assert().Equal("unknown", instanceID)
}

func (s *ResourceHelpersTestSuite) Test_is_resource_new() {
	changes1 := &provider.Changes{
		NewFields: []provider.FieldChange{
			{
				FieldPath: "newField",
				NewValue:  core.MappingNodeFromString("newValue"),
			},
		},
	}
	s.Assert().True(IsResourceNew(changes1))

	changes2 := &provider.Changes{
		ModifiedFields: []provider.FieldChange{
			{
				FieldPath: "modifiedField",
				PrevValue: core.MappingNodeFromString("oldValue"),
				NewValue:  core.MappingNodeFromString("modifiedValue"),
			},
		},
		UnchangedFields: []string{"field2"},
	}
	s.Assert().False(IsResourceNew(changes2))

	changes3 := &provider.Changes{
		RemovedFields: []string{"removedField"},
	}
	s.Assert().False(IsResourceNew(changes3))

	changes4 := &provider.Changes{
		UnchangedFields: []string{"field1"},
		NewFields: []provider.FieldChange{
			{
				FieldPath: "newField",
				NewValue:  core.MappingNodeFromString("newValue"),
			},
		},
	}
	s.Assert().False(IsResourceNew(changes4))

	changes5 := &provider.Changes{
		ModifiedFields: []provider.FieldChange{
			{
				FieldPath: "modifiedField",
				PrevValue: core.MappingNodeFromString("oldValue"),
				NewValue:  core.MappingNodeFromString("modifiedValue"),
			},
		},
		UnchangedFields: []string{"field2"},
		MustRecreate:    true,
	}
	s.Assert().True(IsResourceNew(changes5))
}

func TestResourceHelpersTestSuite(t *testing.T) {
	suite.Run(t, new(ResourceHelpersTestSuite))
}
