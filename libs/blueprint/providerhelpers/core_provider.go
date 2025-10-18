package providerhelpers

import (
	"context"
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/corefunctions"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

type coreProvider struct {
	functions    map[string]provider.Function
	functionList []string
}

// NewCoreProvider returns a new instance of the core provider
// that contains all the core functions as per the blueprint
// specification along with a stub resource used by host applications
// to load a blueprint container without requiring a specific blueprint
// document.
// The stub resource is primarily useful for loading a blueprint container
// to destroy a blueprint instance without requiring the user to provide a
// blueprint document.
func NewCoreProvider(
	linkStateRetriever corefunctions.LinkStateRetriever,
	blueprintInstanceIDRetriever corefunctions.BlueprintInstanceIDRetriever,
	resolveWorkingDir corefunctions.WorkingDirResolver,
	fileSourceRegistry provider.FileSourceRegistry,
	clock core.Clock,
) provider.Provider {
	functions := map[string]provider.Function{
		"fromjson":      corefunctions.NewFromJSONFunction(),
		"fromjson_g":    corefunctions.NewFromJSON_G_Function(),
		"jsondecode":    corefunctions.NewJSONDecodeFunction(),
		"len":           corefunctions.NewLenFunction(),
		"substr":        corefunctions.NewSubstrFunction(),
		"substr_g":      corefunctions.NewSubstr_G_Function(),
		"replace":       corefunctions.NewReplaceFunction(),
		"replace_g":     corefunctions.NewReplace_G_Function(),
		"trim":          corefunctions.NewTrimFunction(),
		"trimprefix":    corefunctions.NewTrimPrefixFunction(),
		"trimprefix_g":  corefunctions.NewTrimPrefix_G_Function(),
		"trimsuffix":    corefunctions.NewTrimSuffixFunction(),
		"trimsuffix_g":  corefunctions.NewTrimSuffix_G_Function(),
		"split":         corefunctions.NewSplitFunction(),
		"split_g":       corefunctions.NewSplit_G_Function(),
		"join":          corefunctions.NewJoinFunction(),
		"index":         corefunctions.NewIndexFunction(),
		"last_index":    corefunctions.NewLastIndexFunction(),
		"to_upper":      corefunctions.NewToUpperFunction(),
		"to_lower":      corefunctions.NewToLowerFunction(),
		"has_prefix":    corefunctions.NewHasPrefixFunction(),
		"has_prefix_g":  corefunctions.NewHasPrefix_G_Function(),
		"has_suffix":    corefunctions.NewHasSuffixFunction(),
		"has_suffix_g":  corefunctions.NewHasSuffix_G_Function(),
		"contains":      corefunctions.NewContainsFunction(),
		"contains_g":    corefunctions.NewContains_G_Function(),
		"list":          corefunctions.NewListFunction(),
		"object":        corefunctions.NewObjectFunction(),
		"keys":          corefunctions.NewKeysFunction(),
		"vals":          corefunctions.NewValsFunction(),
		"first":         corefunctions.NewFirstFunction(),
		"lookup":        corefunctions.NewLookupFunction(),
		"if":            corefunctions.NewIfFunction(),
		"map":           corefunctions.NewMapFunction(),
		"filter":        corefunctions.NewFilterFunction(),
		"reduce":        corefunctions.NewReduceFunction(),
		"sort":          corefunctions.NewSortFunction(),
		"flatmap":       corefunctions.NewFlatMapFunction(),
		"compose":       corefunctions.NewComposeFunction(),
		"pipe":          corefunctions.NewPipeFunction(),
		"getattr":       corefunctions.NewGetAttrFunction(),
		"_getattr_exec": corefunctions.NewGetAttrExecFunction(),
		"getelem":       corefunctions.NewGetElemFunction(),
		"_getelem_exec": corefunctions.NewGetElemExecFunction(),
		"link": corefunctions.NewLinkFunction(
			linkStateRetriever,
			blueprintInstanceIDRetriever,
		),
		"and":           corefunctions.NewAndFunction(),
		"or":            corefunctions.NewOrFunction(),
		"not":           corefunctions.NewNotFunction(),
		"eq":            corefunctions.NewEqFunction(),
		"gt":            corefunctions.NewGtFunction(),
		"ge":            corefunctions.NewGeFunction(),
		"lt":            corefunctions.NewLtFunction(),
		"le":            corefunctions.NewLeFunction(),
		"cwd":           corefunctions.NewCWDFunction(resolveWorkingDir),
		"datetime":      corefunctions.NewDateTimeFunction(clock),
		"base64encode":  corefunctions.NewBase64EncodeFunction(),
		"base64decode":  corefunctions.NewBase64DecodeFunction(),
		"min":           corefunctions.NewMinFunction(),
		"max":           corefunctions.NewMaxFunction(),
		"abs":           corefunctions.NewAbsFunction(),
		"file":          corefunctions.NewFileFunction(fileSourceRegistry),
		"utf8":          corefunctions.NewUTF8Function(),
		"sha256":        corefunctions.NewSHA256Function(),
		"md5":           corefunctions.NewMD5Function(),
		"sha1":          corefunctions.NewSHA1Function(),
		"uuid":          corefunctions.NewUUIDFunction(),
		"cidrsubnet":    corefunctions.NewCIDRSubnetFunction(),
		"http_resource": corefunctions.NewHTTPResourceFunction(),
	}
	return &coreProvider{
		functions:    functions,
		functionList: funcMapKeys(functions),
	}
}

func (p *coreProvider) Namespace(ctx context.Context) (string, error) {
	return "core", nil
}

func (p *coreProvider) ConfigDefinition(ctx context.Context) (*core.ConfigDefinition, error) {
	return &core.ConfigDefinition{
		Fields: map[string]*core.ConfigFieldDefinition{},
	}, nil
}

func (p *coreProvider) Resource(ctx context.Context, resourceType string) (provider.Resource, error) {
	if resourceType == stubResourceType {
		return newStubResource(), nil
	}

	return nil, fmt.Errorf(
		"resource type %q not found in core provider, "+
			"only functions and stub resources are made available by the core provider",
		resourceType,
	)
}

func (p *coreProvider) DataSource(ctx context.Context, dataSourceType string) (provider.DataSource, error) {
	return nil, fmt.Errorf(
		"data source type %q not found in core provider, "+
			"only functions and stub resources are made available by the core provider",
		dataSourceType,
	)
}

func (p *coreProvider) Link(ctx context.Context, resourceTypeA string, resourceTypeB string) (provider.Link, error) {
	return nil, fmt.Errorf(
		"link between resource types %q and %q not found in core provider, "+
			"only functions and stub resources are made available by the core provider",
		resourceTypeA,
		resourceTypeB,
	)
}

func (p *coreProvider) CustomVariableType(ctx context.Context, customVariableType string) (provider.CustomVariableType, error) {
	return nil, fmt.Errorf(
		"custom variable type %q not found in core provider, "+
			"only functions and stub resources are made available by the core provider",
		customVariableType,
	)
}

func (p *coreProvider) ListResourceTypes(ctx context.Context) ([]string, error) {
	return []string{
		stubResourceType,
	}, nil
}

func (p *coreProvider) ListLinkTypes(ctx context.Context) ([]string, error) {
	return []string{}, nil
}

func (p *coreProvider) ListDataSourceTypes(ctx context.Context) ([]string, error) {
	return []string{}, nil
}

func (p *coreProvider) ListFunctions(ctx context.Context) ([]string, error) {
	return p.functionList, nil
}

func (p *coreProvider) ListCustomVariableTypes(ctx context.Context) ([]string, error) {
	return []string{}, nil
}

func (p *coreProvider) Function(ctx context.Context, functionName string) (provider.Function, error) {
	function, ok := p.functions[functionName]
	if !ok {
		return nil, fmt.Errorf(
			"function %q not found in core provider",
			functionName,
		)
	}
	return function, nil
}

// The core provider does not provide a retry policy,
// as the core provider only provides functions, there will be no support
// for retries.
// All core functions apart from the `link` function do not perform any IO
// operations. The state container implementation that powers the `link` function
// should be responsible for retrying transient errors.
func (p *coreProvider) RetryPolicy(ctx context.Context) (*provider.RetryPolicy, error) {
	return nil, nil
}

func funcMapKeys(m map[string]provider.Function) []string {
	keys := []string{}
	for key := range m {
		keys = append(keys, key)
	}
	return keys
}
