package deployconfig

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
)

// Context variable names the Celerity transformer reads. The deploy target
// name in particular is required as the transformer resolves its emit pipeline
// from it and fails outright when it is missing.
const (
	deployTargetContextVar = "deployTarget"
	appNameContextVar      = "celerity.appName"
	appEnvContextVar       = "appEnv"
)

// deployTargetToProvider maps deploy target names to their provider namespace.
var deployTargetToProvider = map[string]string{
	"aws":               "aws",
	"aws-serverless":    "aws",
	"gcloud":            "gcloud",
	"gcloud-serverless": "gcloud",
	"azure":             "azure",
	"azure-serverless":  "azure",
}

// transformerConfigPrefixes are the deploy target config key prefixes that
// address resource-specific configuration handled by the Celerity transformer
// rather than the provider, keyed by provider namespace.
var transformerConfigPrefixes = map[string][]string{
	"aws": {
		"aws.sqs.", "aws.sns.", "aws.dynamodb.", "aws.s3.",
		"aws.lambda.", "aws.apigateway.", "aws.ecs.", "aws.eks.",
	},
	"gcloud": {
		"gcloud.pubsub.", "gcloud.cloudfunctions.", "gcloud.cloudrun.",
		"gcloud.firestore.", "gcloud.storage.",
	},
	"azure": {
		"azure.servicebus.", "azure.functions.", "azure.container.",
		"azure.cosmosdb.", "azure.storage.",
	},
}

// appDeployConfig is the authoring-level Celerity deploy configuration.
type appDeployConfig struct {
	AppName            string         `json:"appName"`
	DeployTarget       deployTarget   `json:"deployTarget"`
	ContextVariables   map[string]any `json:"contextVariables,omitempty"`
	BlueprintVariables map[string]any `json:"blueprintVariables,omitempty"`
}

type deployTarget struct {
	Name   string         `json:"name"`
	AppEnv string         `json:"appEnv,omitempty"`
	Config map[string]any `json:"config,omitempty"`
}

// CelerityAppConfigFile is the conventional authoring-level Celerity deploy
// configuration file name, found in the project root.
const CelerityAppConfigFile = "app.deploy.jsonc"

// The prefix shared by CelerityAppConfigFile and any
// per-environment variants of it.
const celerityAppConfigPrefix = "app.deploy."

// CelerityAppSource reads the authoring-level Celerity deploy configuration
// and converts it to the canonical Bluelink format.
//
// The conversion mirrors the Celerity CLI's pre-command step so that the
// language server derives the same parameters the CLI sends to the deploy
// engine. It is covered by a golden fixture test that pairs an input file with
// the output the CLI produces for it.
type CelerityAppSource struct{}

// Recognises matches the conventional name and its per-environment variants, so
// that an explicitly named app.deploy.staging.jsonc is still converted.
func (s *CelerityAppSource) Recognises(path string) bool {
	return hasBaseNamePrefix(path, celerityAppConfigPrefix)
}

func (s *CelerityAppSource) Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading app deploy config %s: %w", path, err)
	}

	appConfig := &appDeployConfig{}
	if err := json.Unmarshal([]byte(stripJSONCComments(string(data))), appConfig); err != nil {
		return nil, fmt.Errorf("parsing app deploy config %s: %w", path, err)
	}

	return convertAppConfig(appConfig, path)
}

func convertAppConfig(appConfig *appDeployConfig, path string) (*Config, error) {
	if appConfig.DeployTarget.Name == "" {
		return nil, fmt.Errorf("app deploy config %s is missing deployTarget.name", path)
	}

	providerNamespace, supported := deployTargetToProvider[appConfig.DeployTarget.Name]
	if !supported {
		return nil, fmt.Errorf(
			"app deploy config %s has unsupported deploy target %q",
			path,
			appConfig.DeployTarget.Name,
		)
	}

	providerConfig, transformerConfig := splitTargetConfig(
		appConfig.DeployTarget.Config,
		providerNamespace,
	)

	return &Config{
		Providers:          map[string]map[string]*core.ScalarValue{providerNamespace: providerConfig},
		Transformers:       map[string]map[string]*core.ScalarValue{"celerity": transformerConfig},
		ContextVariables:   buildContextVariables(appConfig),
		BlueprintVariables: toScalarMap(appConfig.BlueprintVariables),
	}, nil
}

// Assembles the context variables the transformer reads.
// A missing appName is tolerated rather than rejected, the language server
// would otherwise fall back to no diagnostics at all over a value that only
// affects generated resource naming.
func buildContextVariables(appConfig *appDeployConfig) map[string]*core.ScalarValue {
	contextVariables := toScalarMap(appConfig.ContextVariables)
	contextVariables[deployTargetContextVar] = core.ScalarFromString(appConfig.DeployTarget.Name)

	if appConfig.AppName != "" {
		contextVariables[appNameContextVar] = core.ScalarFromString(appConfig.AppName)
	}

	if appConfig.DeployTarget.AppEnv != "" {
		contextVariables[appEnvContextVar] = core.ScalarFromString(appConfig.DeployTarget.AppEnv)
	}

	return contextVariables
}

// Divides deploy target config between the provider and the
// transformer. Transformer keys keep their full dotted form because that is
// what the transformer's config definition declares, while provider keys have
// the namespace prefix stripped to match the provider's bare field names.
func splitTargetConfig(
	config map[string]any,
	providerNamespace string,
) (providerConfig, transformerConfig map[string]*core.ScalarValue) {
	providerConfig = map[string]*core.ScalarValue{}
	transformerConfig = map[string]*core.ScalarValue{}

	prefixes := transformerConfigPrefixes[providerNamespace]
	for key, value := range config {
		scalar, isScalar := toScalar(value)
		if !isScalar {
			continue
		}

		if hasAnyPrefix(key, prefixes) {
			transformerConfig[key] = scalar
			continue
		}

		providerConfig[strings.TrimPrefix(key, providerNamespace+".")] = scalar
	}

	return providerConfig, transformerConfig
}

func hasAnyPrefix(key string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}

func toScalarMap(values map[string]any) map[string]*core.ScalarValue {
	scalars := map[string]*core.ScalarValue{}
	for key, value := range values {
		if scalar, isScalar := toScalar(value); isScalar {
			scalars[key] = scalar
		}
	}
	return scalars
}

// Converts a decoded JSON value to a scalar. Objects, arrays and null
// have no scalar equivalent and are reported as unconvertible so callers can
// drop them rather than fail the whole configuration.
func toScalar(value any) (*core.ScalarValue, bool) {
	switch typedValue := value.(type) {
	case string:
		return core.ScalarFromString(typedValue), true
	case bool:
		return core.ScalarFromBool(typedValue), true
	case float64:
		if typedValue == math.Trunc(typedValue) && math.Abs(typedValue) < math.MaxInt64 {
			return core.ScalarFromInt(int(typedValue)), true
		}
		return core.ScalarFromFloat(typedValue), true
	default:
		return nil, false
	}
}
