package pluginutils

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// AWSTag represents an AWS-style tag with pointer fields as commonly used by AWS SDK.
type AWSTag struct {
	Key   *string
	Value *string
}

// ToAWSTags converts BluelinkTags to a slice of AWS-style tags.
// Returns nil if tags is nil.
func ToAWSTags(tags *provider.BluelinkTags) []AWSTag {
	if tags == nil {
		return nil
	}

	prefix := tags.Prefix
	if prefix == "" {
		prefix = "bluelink:"
	}

	return []AWSTag{
		{Key: strPtr(fmt.Sprintf("%sinstance-id", prefix)), Value: strPtr(tags.InstanceID)},
		{Key: strPtr(fmt.Sprintf("%sinstance-name", prefix)), Value: strPtr(tags.InstanceName)},
		{Key: strPtr(fmt.Sprintf("%sresource-name", prefix)), Value: strPtr(tags.ResourceName)},
		{Key: strPtr(fmt.Sprintf("%sresource-type", prefix)), Value: strPtr(tags.ResourceType)},
		{Key: strPtr(fmt.Sprintf("%sprovisioned-by", prefix)), Value: strPtr(tags.ProvisionedBy)},
		{Key: strPtr(fmt.Sprintf("%sdeploy-engine-version", prefix)), Value: strPtr(tags.DeployEngineVersion)},
		{Key: strPtr(fmt.Sprintf("%sprovider-plugin-id", prefix)), Value: strPtr(tags.ProviderPluginID)},
		{Key: strPtr(fmt.Sprintf("%sprovider-plugin-version", prefix)), Value: strPtr(tags.ProviderPluginVersion)},
	}
}

// MergeAWSTags merges Bluelink tags with user-defined tags.
// User tags take precedence on key conflicts.
func MergeAWSTags(bluelinkTags, userTags []AWSTag) []AWSTag {
	tagMap := make(map[string]AWSTag)

	// Add Bluelink tags first
	for _, tag := range bluelinkTags {
		if tag.Key != nil {
			tagMap[*tag.Key] = tag
		}
	}

	// User tags override Bluelink tags on conflict
	for _, tag := range userTags {
		if tag.Key != nil {
			tagMap[*tag.Key] = tag
		}
	}

	result := make([]AWSTag, 0, len(tagMap))
	for _, tag := range tagMap {
		result = append(result, tag)
	}

	return result
}

// ToGCPLabels converts BluelinkTags to GCP-compatible labels.
// GCP labels have restrictions: lowercase, max 63 chars, valid chars [a-z0-9_-].
// Returns nil if tags is nil.
func ToGCPLabels(tags *provider.BluelinkTags) map[string]string {
	if tags == nil {
		return nil
	}

	prefix := normalizeGCPLabelKey(tags.Prefix)
	if prefix == "" {
		prefix = "bluelink-"
	}

	return map[string]string{
		fmt.Sprintf("%sinstance-id", prefix):             normalizeGCPLabelValue(tags.InstanceID),
		fmt.Sprintf("%sinstance-name", prefix):           normalizeGCPLabelValue(tags.InstanceName),
		fmt.Sprintf("%sresource-name", prefix):           normalizeGCPLabelValue(tags.ResourceName),
		fmt.Sprintf("%sresource-type", prefix):           normalizeGCPLabelValue(tags.ResourceType),
		fmt.Sprintf("%sprovisioned-by", prefix):          normalizeGCPLabelValue(tags.ProvisionedBy),
		fmt.Sprintf("%sdeploy-engine-version", prefix):   normalizeGCPLabelValue(tags.DeployEngineVersion),
		fmt.Sprintf("%sprovider-plugin-id", prefix):      normalizeGCPLabelValue(tags.ProviderPluginID),
		fmt.Sprintf("%sprovider-plugin-version", prefix): normalizeGCPLabelValue(tags.ProviderPluginVersion),
	}
}

// MergeGCPLabels merges Bluelink labels with user-defined labels.
// User labels take precedence on key conflicts.
func MergeGCPLabels(bluelinkLabels, userLabels map[string]string) map[string]string {
	return core.MergeNativeMaps(bluelinkLabels, userLabels)
}

// ToK8sLabels converts BluelinkTags to Kubernetes-compatible labels.
// K8s labels have restrictions on key format and value length.
// Returns nil if tags is nil.
func ToK8sLabels(tags *provider.BluelinkTags) map[string]string {
	if tags == nil {
		return nil
	}

	// K8s label keys can have a prefix/name format
	// We use "bluelink.dev/" as the official Bluelink domain prefix
	prefix := "bluelink.dev/"
	if tags.Prefix != "" {
		prefix = normalizeK8sLabelKey(tags.Prefix)
	}

	return map[string]string{
		fmt.Sprintf("%sinstance-id", prefix):             normalizeK8sLabelValue(tags.InstanceID),
		fmt.Sprintf("%sinstance-name", prefix):           normalizeK8sLabelValue(tags.InstanceName),
		fmt.Sprintf("%sresource-name", prefix):           normalizeK8sLabelValue(tags.ResourceName),
		fmt.Sprintf("%sresource-type", prefix):           normalizeK8sLabelValue(tags.ResourceType),
		fmt.Sprintf("%sprovisioned-by", prefix):          normalizeK8sLabelValue(tags.ProvisionedBy),
		fmt.Sprintf("%sdeploy-engine-version", prefix):   normalizeK8sLabelValue(tags.DeployEngineVersion),
		fmt.Sprintf("%sprovider-plugin-id", prefix):      normalizeK8sLabelValue(tags.ProviderPluginID),
		fmt.Sprintf("%sprovider-plugin-version", prefix): normalizeK8sLabelValue(tags.ProviderPluginVersion),
	}
}

// MergeK8sLabels merges Bluelink labels with user-defined labels.
// User labels take precedence on key conflicts.
func MergeK8sLabels(bluelinkLabels, userLabels map[string]string) map[string]string {
	return core.MergeNativeMaps(bluelinkLabels, userLabels)
}

// normalizeGCPLabelKey normalizes a key for GCP label requirements.
// GCP label keys must be lowercase, start with a letter, and contain only
// lowercase letters, digits, underscores, and hyphens. Max 63 chars.
func normalizeGCPLabelKey(key string) string {
	// Convert to lowercase
	key = strings.ToLower(key)

	// Replace colons and other invalid chars with hyphens
	key = strings.ReplaceAll(key, ":", "-")
	key = strings.ReplaceAll(key, "/", "-")

	// Remove any remaining invalid characters
	key = removeInvalidGCPChars(key)

	// Ensure it starts with a letter
	if len(key) > 0 && !unicode.IsLetter(rune(key[0])) {
		key = fmt.Sprintf("l%s", key)
	}

	// Truncate to 63 chars
	if len(key) > 63 {
		key = key[:63]
	}

	return key
}

// normalizeGCPLabelValue normalizes a value for GCP label requirements.
// GCP label values must be lowercase, and contain only lowercase letters,
// digits, underscores, and hyphens. Max 63 chars. Can be empty.
func normalizeGCPLabelValue(value string) string {
	// Convert to lowercase
	value = strings.ToLower(value)

	// Replace invalid chars with hyphens
	value = strings.ReplaceAll(value, ":", "-")
	value = strings.ReplaceAll(value, "/", "-")

	// Remove any remaining invalid characters
	value = removeInvalidGCPChars(value)

	// Truncate to 63 chars
	if len(value) > 63 {
		value = value[:63]
	}

	return value
}

// removeInvalidGCPChars removes characters that are not valid for GCP labels.
var gcpValidCharsRegex = regexp.MustCompile(`[^a-z0-9_-]`)

func removeInvalidGCPChars(s string) string {
	return gcpValidCharsRegex.ReplaceAllString(s, "")
}

// normalizeK8sLabelKey normalizes a key for Kubernetes label requirements.
// K8s label keys can have an optional prefix (DNS subdomain) followed by a slash
// and a name segment. The name segment must be 63 chars or less.
func normalizeK8sLabelKey(key string) string {
	// For simplicity, we just ensure valid characters
	// K8s allows alphanumeric, -, _, and . in the name segment
	// The prefix must be a valid DNS subdomain

	// Replace colons with dots for domain-style prefixes
	key = strings.ReplaceAll(key, ":", ".")

	// Remove trailing dots
	key = strings.TrimRight(key, ".")

	// Ensure it ends with a slash if it looks like a prefix
	if !strings.HasSuffix(key, "/") && strings.Contains(key, ".") {
		key = fmt.Sprintf("%s/", key)
	}

	return key
}

// normalizeK8sLabelValue normalizes a value for Kubernetes label requirements.
// K8s label values must be 63 chars or less, begin and end with alphanumeric,
// and contain only alphanumeric, -, _, and .
func normalizeK8sLabelValue(value string) string {
	// Replace invalid chars
	value = strings.ReplaceAll(value, ":", "-")
	value = strings.ReplaceAll(value, "/", "-")

	// Remove any remaining invalid characters
	value = removeInvalidK8sChars(value)

	// Truncate to 63 chars
	if len(value) > 63 {
		value = value[:63]
	}

	// Ensure it starts and ends with alphanumeric
	value = strings.Trim(value, "-_.")

	return value
}

// removeInvalidK8sChars removes characters that are not valid for K8s label values.
var k8sValidCharsRegex = regexp.MustCompile(`[^a-zA-Z0-9._-]`)

func removeInvalidK8sChars(s string) string {
	return k8sValidCharsRegex.ReplaceAllString(s, "")
}

// strPtr is a helper function to create a string pointer.
func strPtr(s string) *string {
	return &s
}
