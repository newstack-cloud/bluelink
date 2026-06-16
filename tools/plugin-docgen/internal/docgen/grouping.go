package docgen

import (
	"encoding/json"
	"sort"
	"strings"
	"unicode"

	"github.com/spf13/afero"
)

// Used for the group key of links that connect two
// resources belonging to different services (e.g. "from:lambda").
const fromGroupKeyPrefix = "from:"

// GroupConfig holds optional, plugin-author-supplied overrides for the
// service-like groups that documentation elements are organised into.
// It is typically loaded from a JSON file passed to the doc generator and is
// keyed by the service group key (the {service} segment of a type string).
type GroupConfig struct {
	Groups map[string]GroupOverride `json:"groups"`
}

// GroupOverride supplies a human-friendly label and description for a group key.
type GroupOverride struct {
	Label       string `json:"label"`
	Description string `json:"description"`
}

// LoadGroupConfig loads an optional group config/overrides file from the given
// path. An empty path or an empty/whitespace-only file yields a nil config
// (so the doc generator falls back to derived, title-cased labels). A malformed
// file returns an error.
func LoadGroupConfig(fs afero.Fs, path string) (*GroupConfig, error) {
	if strings.TrimSpace(path) == "" {
		return nil, nil
	}

	data, err := afero.ReadFile(fs, path)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(string(data)) == "" {
		return nil, nil
	}

	config := &GroupConfig{}
	if err := json.Unmarshal(data, config); err != nil {
		return nil, err
	}

	return config, nil
}

// extractServiceGroupKey returns the service segment of a type string in the
// form {provider}/{service}/{resource}. It returns an empty string when the
// type does not have at least three segments.
func extractServiceGroupKey(itemType string) string {
	parts := strings.Split(itemType, "/")
	if len(parts) < 3 {
		return ""
	}

	return parts[1]
}

// Derives the group key for a link type ({typeA}::{typeB}).
// Links between two resources of the same service are grouped under that
// service; links that cross services are grouped under "from:{serviceA}".
// An empty string is returned when the key cannot be derived.
func linkGroupKey(linkType string) string {
	info, err := extractLinkTypeInfo(linkType)
	if err != nil {
		return ""
	}

	serviceA := extractServiceGroupKey(info.resourceTypeA)
	if serviceA == "" {
		return ""
	}

	if serviceA == extractServiceGroupKey(info.resourceTypeB) {
		return serviceA
	}

	return fromGroupKeyPrefix + serviceA
}

type orderedGroupKeys struct {
	seen map[string]struct{}
	keys []string
}

func newOrderedGroupKeys() *orderedGroupKeys {
	return &orderedGroupKeys{seen: map[string]struct{}{}}
}

func (o *orderedGroupKeys) add(key string) {
	if key == "" {
		return
	}

	if _, ok := o.seen[key]; ok {
		return
	}

	o.seen[key] = struct{}{}
	o.keys = append(o.keys, key)
}

func buildProviderServiceGroups(docs *PluginDocs, config *GroupConfig) []*PluginDocsServiceGroup {
	ordered := newOrderedGroupKeys()
	for _, resource := range docs.Resources {
		ordered.add(resource.Group)
	}

	for _, dataSource := range docs.DataSources {
		ordered.add(dataSource.Group)
	}

	for _, customVarType := range docs.CustomVarTypes {
		ordered.add(customVarType.Group)
	}

	for _, link := range docs.Links {
		ordered.add(link.Group)
	}

	return buildServiceGroups(ordered.keys, config)
}

func buildTransformerServiceGroups(docs *PluginDocs, config *GroupConfig) []*PluginDocsServiceGroup {
	ordered := newOrderedGroupKeys()
	for _, resource := range docs.AbstractResources {
		ordered.add(resource.Group)
	}

	for _, link := range docs.AbstractLinks {
		ordered.add(link.Group)
	}

	return buildServiceGroups(ordered.keys, config)
}

func buildServiceGroups(keys []string, config *GroupConfig) []*PluginDocsServiceGroup {
	if len(keys) == 0 {
		return nil
	}

	sort.Strings(keys)

	groups := make([]*PluginDocsServiceGroup, 0, len(keys))
	for _, key := range keys {
		groups = append(groups, &PluginDocsServiceGroup{
			Key:         key,
			Label:       resolveGroupLabel(key, config),
			Description: resolveGroupDescription(key, config),
		})
	}

	return groups
}

func resolveGroupLabel(key string, config *GroupConfig) string {
	if serviceKey, ok := strings.CutPrefix(key, fromGroupKeyPrefix); ok {
		return "From " + resolveServiceLabel(serviceKey, config)
	}

	return resolveServiceLabel(key, config)
}

func resolveServiceLabel(serviceKey string, config *GroupConfig) string {
	if override, ok := lookupGroupOverride(serviceKey, config); ok && override.Label != "" {
		return override.Label
	}

	return titleCaseServiceLabel(serviceKey)
}

func resolveGroupDescription(key string, config *GroupConfig) string {
	if strings.HasPrefix(key, fromGroupKeyPrefix) {
		return ""
	}

	if override, ok := lookupGroupOverride(key, config); ok {
		return override.Description
	}

	return ""
}

func lookupGroupOverride(key string, config *GroupConfig) (GroupOverride, bool) {
	if config == nil || config.Groups == nil {
		return GroupOverride{}, false
	}

	override, ok := config.Groups[key]
	return override, ok
}

// Produces a fallback label from a service key by
// upper-casing the first letter of each "-"/"_"-separated word. This is a best
// effort fallback; acronyms (e.g. "iam" -> "Iam") are exactly the case plugin
// authors should override via a group config file.
func titleCaseServiceLabel(key string) string {
	words := strings.FieldsFunc(key, func(r rune) bool {
		return r == '-' || r == '_'
	})
	for i, word := range words {
		runes := []rune(word)
		runes[0] = unicode.ToUpper(runes[0])
		words[i] = string(runes)
	}

	return strings.Join(words, " ")
}
