package internal

import "github.com/newstack-cloud/bluelink/libs/blueprint/schema"

type BlueprintSpecMock struct {
	Blueprint *schema.Blueprint
}

func NewBlueprintSpecMock(blueprint *schema.Blueprint) *BlueprintSpecMock {
	return &BlueprintSpecMock{
		Blueprint: blueprint,
	}
}

func (b *BlueprintSpecMock) ResourceSchema(resourceName string) *schema.Resource {
	if b.Blueprint.Resources == nil {
		return nil
	}

	return b.Blueprint.Resources.Values[resourceName]
}

func (b *BlueprintSpecMock) Schema() *schema.Blueprint {
	return b.Blueprint
}
