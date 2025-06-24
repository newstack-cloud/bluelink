package docgen

import "github.com/newstack-cloud/bluelink/libs/blueprint/core"

// PluginDocs is a struct that holds the JSON representation of the plugin
// documentation.
type PluginDocs struct {
	ID               string                   `json:"id"`
	DisplayName      string                   `json:"displayName"`
	Version          string                   `json:"version"`
	ProtocolVersions []string                 `json:"protocolVersions"`
	Description      string                   `json:"description"`
	Author           string                   `json:"author"`
	Repository       string                   `json:"repository"`
	Config           *PluginDocsVersionConfig `json:"config"`

	// Required for providers, should be empty for transformers.
	Resources []*PluginDocsResource `json:"resources,omitempty"`
	// Required for providers, should be empty for transformers.
	Links []*PluginDocsLink `json:"links,omitempty"`
	// Required for providers, should be empty for transformers.
	DataSources []*PluginDocsDataSource `json:"dataSources,omitempty"`
	// Required for providers, should be empty for transformers.
	CustomVarTypes []*PluginDocsCustomVarType `json:"customVarTypes,omitempty"`
	// Required for providers, should be empty for transformers.
	Functions []*PluginDocsFunction `json:"functions,omitempty"`

	// Required for transformers, should be empty for providers.
	TransformName string `json:"transformName,omitempty"`
	// Required for transformers, should be empty for providers.
	AbstractResources []*PluginDocsResource `json:"abstractResources,omitempty"`
}

type PluginDocsVersionConfig struct {
	Fields                map[string]*PluginDocsVersionConfigField `json:"fields"`
	AllowAdditionalFields bool                                     `json:"allowAdditionalFields"`
}

type PluginDocsVersionConfigField struct {
	Type          string              `json:"type"`
	Label         string              `json:"label"`
	Description   string              `json:"description"`
	Required      bool                `json:"required"`
	Default       *core.ScalarValue   `json:"default,omitempty"`
	AllowedValues []*core.ScalarValue `json:"allowedValues,omitempty"`
	Secret        bool                `json:"secret"`
	Examples      []*core.ScalarValue `json:"examples,omitempty"`
}

type PluginDocsResource struct {
	Type          string                 `json:"type"`
	Label         string                 `json:"label"`
	Summary       string                 `json:"summary"`
	Description   string                 `json:"description"`
	Specification *PluginDocResourceSpec `json:"specification"`
	Examples      []string               `json:"examples"`
	CanLinkTo     []string               `json:"canLinkTo"`
}

type PluginDocResourceSpec struct {
	Schema  *PluginDocResourceSpecSchema `json:"schema"`
	IDField string                       `json:"idField"`
}

type PluginDocResourceSpecSchema struct {
	Type         string              `json:"type"`
	Label        string              `json:"label"`
	Description  string              `json:"description"`
	Nullable     bool                `json:"nullable"`
	Computed     bool                `json:"computed"`
	MustRecreate bool                `json:"mustRecreate"`
	Default      *core.MappingNode   `json:"default,omitempty"`
	Examples     []*core.MappingNode `json:"examples,omitempty"`

	// Required for "object" types, should be empty for other types.
	Attributes map[string]*PluginDocResourceSpecSchema `json:"attributes,omitempty"`
	// Required for "object" types, should be empty for other types.
	// This is a list of required attributes.
	Required []string `json:"required,omitempty"`

	// Required for "map" types, should be empty for other types.
	MapValues *PluginDocResourceSpecSchema `json:"mapValues,omitempty"`

	// Required for "array" types, should be empty for other types.
	Items *PluginDocResourceSpecSchema `json:"listValues,omitempty"`

	// Required for "union" types, should be empty for other types.
	OneOf []*PluginDocResourceSpecSchema `json:"oneOf,omitempty"`
}

type PluginDocsLink struct {
	Type                  string                                         `json:"type"`
	Summary               string                                         `json:"summary"`
	Description           string                                         `json:"description"`
	AnnotationDefinitions map[string]*PluginDocsLinkAnnotationDefinition `json:"annotationDefinitions"`
}

type PluginDocsLinkAnnotationDefinition struct {
	Name          string              `json:"name"`
	Label         string              `json:"label"`
	Type          string              `json:"type"`
	Description   string              `json:"description"`
	Default       *core.ScalarValue   `json:"default,omitempty"`
	AllowedValues []*core.ScalarValue `json:"allowedValues,omitempty"`
	Examples      []*core.ScalarValue `json:"examples,omitempty"`
	Required      bool                `json:"required"`
}

type PluginDocsDataSource struct {
	Type          string                    `json:"type"`
	Label         string                    `json:"label"`
	Summary       string                    `json:"summary"`
	Description   string                    `json:"description"`
	Specification *PluginDocsDataSourceSpec `json:"specification"`
	Examples      []string                  `json:"examples,omitempty"`
}

type PluginDocsDataSourceSpec struct {
	Fields map[string]*PluginDocsDataSourceFieldSpec `json:"fields"`
}

type PluginDocsDataSourceFieldSpec struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Nullable    bool   `json:"nullable"`
	Filterable  bool   `json:"filterable"`
}

type PluginDocsCustomVarType struct {
	Type        string                                    `json:"type"`
	Label       string                                    `json:"label"`
	Summary     string                                    `json:"summary"`
	Description string                                    `json:"description"`
	Options     map[string]*PluginDocsCustomVarTypeOption `json:"options"`
	Examples    []string                                  `json:"examples,omitempty"`
}

type PluginDocsCustomVarTypeOption struct {
	Label       string `json:"label"`
	Description string `json:"value"`
}

type PluginDocsFunction struct {
	FunctionDefinition
	Name        string `json:"name"`
	Summary     string `json:"summary"`
	Description string `json:"description"`
}
