package plugintestsuites

import (
	"os"

	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/internal/testutils"
	"gopkg.in/yaml.v3"
)

func createTransformInput() (*transform.SpecTransformerTransformInput, error) {
	blueprint, err := loadTestBlueprint()
	if err != nil {
		return nil, err
	}

	return &transform.SpecTransformerTransformInput{
		InputBlueprint:     blueprint,
		TransformerContext: testutils.CreateTestTransformerContext("celerity"),
	}, nil
}

func loadTestBlueprint() (*schema.Blueprint, error) {
	blueprintBytes, err := os.ReadFile("__testdata/transform/blueprint.yml")
	if err != nil {
		return nil, err
	}

	blueprint := &schema.Blueprint{}
	err = yaml.Unmarshal(blueprintBytes, blueprint)
	if err != nil {
		return nil, err
	}

	return blueprint, nil
}
