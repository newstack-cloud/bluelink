package pluginutils

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/stretchr/testify/suite"
)

type ChangesTestSuite struct {
	suite.Suite
}

type getCurrentResourceStateSpecDataTestCase struct {
	name             string
	inputChanges     *provider.Changes
	expectedSpecData *core.MappingNode
	expectedEmpty    bool
}

func (s *ChangesTestSuite) Test_get_current_resource_state_spec_data() {
	testCases := []getCurrentResourceStateSpecDataTestCase{
		{
			name:         "nil changes",
			inputChanges: nil,
			expectedSpecData: &core.MappingNode{
				Fields: map[string]*core.MappingNode{},
			},
			expectedEmpty: true,
		},
		{
			name: "no current resource state",
			inputChanges: &provider.Changes{
				AppliedResourceInfo: provider.ResourceInfo{},
			},
			expectedSpecData: &core.MappingNode{
				Fields: map[string]*core.MappingNode{},
			},
			expectedEmpty: true,
		},
		{
			name: "valid current resource state",
			inputChanges: &provider.Changes{
				AppliedResourceInfo: provider.ResourceInfo{
					CurrentResourceState: &state.ResourceState{
						SpecData: &core.MappingNode{
							Fields: map[string]*core.MappingNode{
								"field1": core.MappingNodeFromString("value1"),
							},
						},
					},
				},
			},
			expectedSpecData: &core.MappingNode{
				Fields: map[string]*core.MappingNode{
					"field1": core.MappingNodeFromString("value1"),
				},
			},
			expectedEmpty: false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			result := GetCurrentResourceStateSpecData(tc.inputChanges)
			s.Assert().Equal(tc.expectedSpecData, result)
			if tc.expectedEmpty {
				s.Assert().Len(result.Fields, 0)
			} else {
				s.Assert().NotEmpty(result.Fields)
			}
		})
	}
}

type getSpecDataFromResourceInfoTestCase struct {
	name             string
	resourceInfo     *provider.ResourceInfo
	expectedSpecData *core.MappingNode
	expectedEmpty    bool
}

func (s *ChangesTestSuite) Test_get_spec_data_from_resource_info() {
	testCases := []getSpecDataFromResourceInfoTestCase{
		{
			name:         "nil resource info",
			resourceInfo: nil,
			expectedSpecData: &core.MappingNode{
				Fields: map[string]*core.MappingNode{},
			},
			expectedEmpty: true,
		},
		{
			name:         "no current resource state",
			resourceInfo: &provider.ResourceInfo{},
			expectedSpecData: &core.MappingNode{
				Fields: map[string]*core.MappingNode{},
			},
			expectedEmpty: true,
		},
		{
			name: "valid current resource state",
			resourceInfo: &provider.ResourceInfo{
				CurrentResourceState: &state.ResourceState{
					SpecData: &core.MappingNode{
						Fields: map[string]*core.MappingNode{
							"field1": core.MappingNodeFromString("value1"),
						},
					},
				},
			},
			expectedSpecData: &core.MappingNode{
				Fields: map[string]*core.MappingNode{
					"field1": core.MappingNodeFromString("value1"),
				},
			},
			expectedEmpty: false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			result := GetCurrentStateSpecDataFromResourceInfo(tc.resourceInfo)
			s.Assert().Equal(tc.expectedSpecData, result)
			if tc.expectedEmpty {
				s.Assert().Len(result.Fields, 0)
			} else {
				s.Assert().NotEmpty(result.Fields)
			}
		})
	}
}

type getResolvedResourceSpecDataTestCase struct {
	name             string
	inputChanges     *provider.Changes
	expectedSpecData *core.MappingNode
	expectedEmpty    bool
}

func (s *ChangesTestSuite) Test_get_resolved_resource_spec_data() {
	testCases := []getResolvedResourceSpecDataTestCase{
		{
			name:         "nil changes",
			inputChanges: nil,
			expectedSpecData: &core.MappingNode{
				Fields: map[string]*core.MappingNode{},
			},
			expectedEmpty: true,
		},
		{
			name: "no resolved resource",
			inputChanges: &provider.Changes{
				AppliedResourceInfo: provider.ResourceInfo{},
			},
			expectedSpecData: &core.MappingNode{
				Fields: map[string]*core.MappingNode{},
			},
			expectedEmpty: true,
		},
		{
			name: "valid resolved resource",
			inputChanges: &provider.Changes{
				AppliedResourceInfo: provider.ResourceInfo{
					ResourceWithResolvedSubs: &provider.ResolvedResource{
						Spec: &core.MappingNode{
							Fields: map[string]*core.MappingNode{
								"field1": core.MappingNodeFromString("value1"),
							},
						},
					},
				},
			},
			expectedSpecData: &core.MappingNode{
				Fields: map[string]*core.MappingNode{
					"field1": core.MappingNodeFromString("value1"),
				},
			},
			expectedEmpty: false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			result := GetResolvedResourceSpecData(tc.inputChanges)
			s.Assert().Equal(tc.expectedSpecData, result)
			if tc.expectedEmpty {
				s.Assert().Len(result.Fields, 0)
			} else {
				s.Assert().NotEmpty(result.Fields)
			}
		})
	}
}

func TestChangesTestSuite(t *testing.T) {
	suite.Run(t, new(ChangesTestSuite))
}
