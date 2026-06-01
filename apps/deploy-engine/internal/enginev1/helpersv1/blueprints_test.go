package helpersv1

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/stretchr/testify/assert"
)

func Test_GetFormat(t *testing.T) {
	tests := []struct {
		name              string
		blueprintFileName string
		expected          schema.SpecFormat
	}{
		{
			name:              "yaml extension",
			blueprintFileName: "project.blueprint.yaml",
			expected:          schema.YAMLSpecFormat,
		},
		{
			name:              "yml extension",
			blueprintFileName: "project.blueprint.yml",
			expected:          schema.YAMLSpecFormat,
		},
		{
			name:              "bp extension",
			blueprintFileName: "project.bp",
			expected:          schema.BlueprintLangSpecFormat,
		},
		{
			name:              "blueprint extension",
			blueprintFileName: "project.blueprint",
			expected:          schema.BlueprintLangSpecFormat,
		},
		{
			name:              "json extension falls back to JWCC",
			blueprintFileName: "project.blueprint.json",
			expected:          schema.JWCCSpecFormat,
		},
		{
			name:              "jsonc extension falls back to JWCC",
			blueprintFileName: "project.blueprint.jsonc",
			expected:          schema.JWCCSpecFormat,
		},
		{
			name:              "unknown extension falls back to JWCC",
			blueprintFileName: "project.blueprint.txt",
			expected:          schema.JWCCSpecFormat,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, GetFormat(tc.blueprintFileName))
		})
	}
}
