package providerv1

import (
	"context"
	"fmt"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/providerserverv1"
)

type linkTypeInfo struct {
	resourceTypeA string
	resourceTypeB string
}

func extractLinkTypeInfo(linkType *providerserverv1.LinkType) (*linkTypeInfo, error) {
	parts := strings.Split(linkType.Type, "::")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid link type: %s", linkType.Type)
	}

	return &linkTypeInfo{
		resourceTypeA: parts[0],
		resourceTypeB: parts[1],
	}, nil
}

type linkContextFromVarMaps struct {
	providerConfigVars map[string]map[string]*core.ScalarValue
	contextVars        map[string]*core.ScalarValue
}

func createLinkContextFromVarMaps(
	providerConfigVars map[string]*core.ScalarValue,
	contextVars map[string]*core.ScalarValue,
) (provider.LinkContext, error) {
	expandedProviderConfigVars, err := expandProviderConfigVars(providerConfigVars)
	if err != nil {
		return nil, err
	}

	return &linkContextFromVarMaps{
		providerConfigVars: expandedProviderConfigVars,
		contextVars:        contextVars,
	}, nil
}

func (l *linkContextFromVarMaps) ProviderConfigVariable(namespace, name string) (*core.ScalarValue, bool) {
	namespaceConfig, ok := l.providerConfigVars[namespace]
	if !ok {
		return nil, false
	}

	v, ok := namespaceConfig[name]
	return v, ok
}

func (l *linkContextFromVarMaps) ProviderConfigVariables(namespace string) map[string]*core.ScalarValue {
	namespaceConfig, ok := l.providerConfigVars[namespace]
	if !ok {
		return nil
	}

	return namespaceConfig
}

func (l *linkContextFromVarMaps) AllProviderConfigVariables() map[string]map[string]*core.ScalarValue {
	return l.providerConfigVars
}

func (l *linkContextFromVarMaps) ContextVariable(name string) (*core.ScalarValue, bool) {
	v, ok := l.contextVars[name]
	return v, ok
}

func (l *linkContextFromVarMaps) ContextVariables() map[string]*core.ScalarValue {
	return l.contextVars
}

func expandProviderConfigVars(
	providerConfigVars map[string]*core.ScalarValue,
) (map[string]map[string]*core.ScalarValue, error) {
	expandedProviderConfigVars := make(map[string]map[string]*core.ScalarValue)

	for key, value := range providerConfigVars {
		keyParts := strings.Split(key, "::")
		if len(keyParts) != 2 {
			return nil, fmt.Errorf("invalid provider config variable key: %s", key)
		}
		namespace := keyParts[0]
		varName := keyParts[1]

		if _, hasNamespace := expandedProviderConfigVars[namespace]; !hasNamespace {
			expandedProviderConfigVars[namespace] = make(map[string]*core.ScalarValue)
		}
		expandedProviderConfigVars[namespace][varName] = value
	}

	return expandedProviderConfigVars, nil
}

type linkUpdateResourceFunc func(
	ctx context.Context,
	input *provider.LinkUpdateResourceInput,
) (*provider.LinkUpdateResourceOutput, error)

func selectLinkUpdateResourceFunc(
	link provider.Link,
	linkResource provider.LinkPriorityResource,
) linkUpdateResourceFunc {
	switch linkResource {
	case provider.LinkPriorityResourceA:
		return link.UpdateResourceA
	case provider.LinkPriorityResourceB:
		return link.UpdateResourceB
	default:
		return func(
			ctx context.Context,
			input *provider.LinkUpdateResourceInput,
		) (*provider.LinkUpdateResourceOutput, error) {
			return nil, fmt.Errorf("unknown link resource type: %d", linkResource)
		}
	}
}
