package changes

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

// Sensitivity is declared on the field a plugin author thinks of as secret,
// which for a map of secrets is the map. The leaves beneath it hold the actual
// values, so they are the ones that must come out marked, otherwise,
// a consumer redacting on the flag doesn't redact the actual secret values.
func (s *ResourceChangeGeneratorTestSuite) Test_marks_values_inside_a_sensitive_map_as_sensitive() {
	resourceChanges, err := s.resourceChangeGenerator.GenerateChanges(
		context.Background(),
		sensitiveContainerResourceInfo(),
		&internal.ExampleSensitiveContainerResource{},
		[]string{},
		nil,
	)
	s.Require().NoError(err)

	secret := findFieldChange(resourceChanges.NewFields, "spec.secureValues.apiToken")
	s.Require().NotNil(secret, "expected a change for the entry inside the sensitive map")
	s.True(secret.Sensitive, "an entry of a sensitive map holds a secret value")

	nested := findFieldChange(resourceChanges.NewFields, "spec.credentials.username")
	s.Require().NotNil(nested, "expected a change for the field inside the sensitive object")
	s.True(nested.Sensitive, "a field of a sensitive object holds a secret value")
}

// Inheritance must not spread sensitivity to fields that are not beneath a
// sensitive container, or everything ends up redacted and the flag stops
// meaning anything.
func (s *ResourceChangeGeneratorTestSuite) Test_leaves_fields_outside_a_sensitive_container_unmarked() {
	resourceChanges, err := s.resourceChangeGenerator.GenerateChanges(
		context.Background(),
		sensitiveContainerResourceInfo(),
		&internal.ExampleSensitiveContainerResource{},
		[]string{},
		nil,
	)
	s.Require().NoError(err)

	plain := findFieldChange(resourceChanges.NewFields, "spec.values.logLevel")
	s.Require().NotNil(plain, "expected a change for the entry inside the plain map")
	s.False(plain.Sensitive)

	name := findFieldChange(resourceChanges.NewFields, "spec.name")
	s.Require().NotNil(name)
	s.False(name.Sensitive)
}

func findFieldChange(fields []provider.FieldChange, path string) *provider.FieldChange {
	for i := range fields {
		if fields[i].FieldPath == path {
			return &fields[i]
		}
	}
	return nil
}

func sensitiveContainerResourceInfo() *provider.ResourceInfo {
	return &provider.ResourceInfo{
		ResourceName: "configStore",
		InstanceID:   "test-instance-1",
		CurrentResourceState: &state.ResourceState{
			ResourceID:    "test-resource-1",
			Name:          "configStore",
			Type:          "example/sensitiveContainer",
			Status:        core.ResourceStatusCreated,
			PreciseStatus: core.PreciseResourceStatusCreated,
			SpecData: &core.MappingNode{
				Fields: map[string]*core.MappingNode{},
			},
		},
		ResourceWithResolvedSubs: &provider.ResolvedResource{
			Type: &schema.ResourceTypeWrapper{Value: "example/sensitiveContainer"},
			Spec: &core.MappingNode{
				Fields: map[string]*core.MappingNode{
					"name": core.MappingNodeFromString("app-config"),
					"values": {
						Fields: map[string]*core.MappingNode{
							"logLevel": core.MappingNodeFromString("info"),
						},
					},
					"secureValues": {
						Fields: map[string]*core.MappingNode{
							"apiToken": core.MappingNodeFromString("super-secret"),
						},
					},
					"credentials": {
						Fields: map[string]*core.MappingNode{
							"username": core.MappingNodeFromString("admin"),
						},
					},
				},
			},
		},
	}
}
