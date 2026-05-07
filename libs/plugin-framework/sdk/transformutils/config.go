package transformutils

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
)

// ResourceConfigVariable reads a resource-scoped or service-scoped
// transformer config variable, preferring the resource-scoped form.
//
// For prefix "aws.dynamodb", resourceName "ordersTable", key "billingMode":
//  1. tries "aws.dynamodb.ordersTable.billingMode"
//  2. falls back to "aws.dynamodb.billingMode"
//  3. returns (nil, error) if neither is set; caller decides default.
func ResourceConfigVariable(
	transformCtx transform.Context,
	prefix string,
	resourceName string,
	key string,
) (*core.ScalarValue, error) {
	resourceKey := fmt.Sprintf("%s.%s.%s", prefix, resourceName, key)
	serviceKey := fmt.Sprintf("%s.%s", prefix, key)

	if value, ok := transformCtx.TransformerConfigVariable(resourceKey); ok {
		return value, nil
	}

	if value, ok := transformCtx.TransformerConfigVariable(serviceKey); ok {
		return value, nil
	}

	return nil, fmt.Errorf("config variable not found: %s", resourceKey)
}

// ResourceConfigVariableMap reads a resource-scoped or service-scoped
// prefix-keyed sub-map of transformer config variables, preferring the
// resource-scoped form. Sub-keys are single-segment; entries with deeper
// nesting under the same prefix are ignored.
//
// For prefix "aws.config", resourceName "primaryConfig", key "regionKMSKeys":
//  1. collects every "aws.config.primaryConfig.regionKMSKeys.<region>" entry
//  2. if none, collects every "aws.config.regionKMSKeys.<region>" entry
//  3. returns (nil, error) if neither has entries; caller decides default.
func ResourceConfigVariableMap(
	transformCtx transform.Context,
	prefix string,
	resourceName string,
	key string,
) (map[string]*core.ScalarValue, error) {
	resourcePrefix := fmt.Sprintf("%s.%s.%s.", prefix, resourceName, key)
	servicePrefix := fmt.Sprintf("%s.%s.", prefix, key)

	all := transformCtx.TransformerConfigVariables()

	if subMap := singleSegmentSubMap(all, resourcePrefix); len(subMap) > 0 {
		return subMap, nil
	}

	if subMap := singleSegmentSubMap(all, servicePrefix); len(subMap) > 0 {
		return subMap, nil
	}

	return nil, fmt.Errorf("config variable map not found: %s*", resourcePrefix)
}

// ResourceConfigVariableSeq reads a resource-scoped or service-scoped
// indexed sub-object sequence of transformer config variables, preferring
// the resource-scoped form. For each contiguous numeric index starting
// at 0, collects every "<prefix>.<resourceName>.<key>.<i>.<subkey>" into
// one map; sub-keys are single-segment. Stops at the first index with no
// sub-keys; if the sequence does not start at 0 it is treated as absent.
//
// For prefix "aws.sns", resourceName "ordersTopic", key "statusLogging":
//  1. collects every "aws.sns.ordersTopic.statusLogging.0.<subkey>",
//     "aws.sns.ordersTopic.statusLogging.1.<subkey>", … until a gap
//  2. if the resource form yields no entries, repeats with
//     "aws.sns.statusLogging.<i>.<subkey>"
//  3. returns (nil, error) if neither yields entries; caller decides default.
func ResourceConfigVariableSeq(
	transformCtx transform.Context,
	prefix string,
	resourceName string,
	key string,
) ([]map[string]*core.ScalarValue, error) {
	resourcePrefix := fmt.Sprintf("%s.%s.%s.", prefix, resourceName, key)
	servicePrefix := fmt.Sprintf("%s.%s.", prefix, key)

	all := transformCtx.TransformerConfigVariables()

	if seq := contiguousIndexedSeq(all, resourcePrefix); len(seq) > 0 {
		return seq, nil
	}

	if seq := contiguousIndexedSeq(all, servicePrefix); len(seq) > 0 {
		return seq, nil
	}

	return nil, fmt.Errorf(
		"config variable sequence not found: %s<index>.<subkey>",
		resourcePrefix,
	)
}

// AppEnv is a helper function to read the "appEnv" context variable,
// which is commonly used to determine the deployment environment
// (e.g., "dev", "staging", "prod").
// It returns an empty string if the variable is not set.
func AppEnv(transformCtx transform.Context) string {
	if value, ok := transformCtx.ContextVariable("appEnv"); ok {
		return core.StringValueFromScalar(value)
	}

	return ""
}

func singleSegmentSubMap(
	all map[string]*core.ScalarValue,
	prefix string,
) map[string]*core.ScalarValue {
	out := map[string]*core.ScalarValue{}
	for k, v := range all {
		if !strings.HasPrefix(k, prefix) {
			continue
		}
		suffix := k[len(prefix):]
		if suffix == "" || strings.Contains(suffix, ".") {
			continue
		}
		out[suffix] = v
	}
	return out
}

func contiguousIndexedSeq(
	all map[string]*core.ScalarValue,
	prefix string,
) []map[string]*core.ScalarValue {
	byIndex := map[int]map[string]*core.ScalarValue{}
	for k, v := range all {
		if !strings.HasPrefix(k, prefix) {
			continue
		}
		suffix := k[len(prefix):]
		dotIdx := strings.Index(suffix, ".")
		if dotIdx <= 0 {
			continue
		}
		index, err := strconv.Atoi(suffix[:dotIdx])
		if err != nil || index < 0 {
			continue
		}
		subKey := suffix[dotIdx+1:]
		if subKey == "" || strings.Contains(subKey, ".") {
			continue
		}
		if byIndex[index] == nil {
			byIndex[index] = map[string]*core.ScalarValue{}
		}
		byIndex[index][subKey] = v
	}

	if len(byIndex) == 0 {
		return nil
	}

	out := []map[string]*core.ScalarValue{}
	for i := 0; ; i += 1 {
		bucket, ok := byIndex[i]
		if !ok {
			break
		}
		out = append(out, bucket)
	}
	return out
}
