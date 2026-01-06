package shared

import (
	"context"
	"net/url"
	"path"
	"path/filepath"
	"strconv"

	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
	"github.com/newstack-cloud/deploy-cli-sdk/consts"
)

// BuildResourceActions converts resource reconciliation results to action payloads.
func BuildResourceActions(resources []container.ResourceReconcileResult) []types.ResourceReconcileActionPayload {
	actions := make([]types.ResourceReconcileActionPayload, 0, len(resources))
	for _, r := range resources {
		actions = append(actions, types.ResourceReconcileActionPayload{
			ResourceID:    r.ResourceID,
			ChildPath:     r.ChildPath,
			Action:        string(r.RecommendedAction),
			ExternalState: r.ExternalState,
			NewStatus:     strconv.Itoa(int(r.NewStatus)),
		})
	}
	return actions
}

// BuildLinkActions converts link reconciliation results to action payloads.
func BuildLinkActions(links []container.LinkReconcileResult) []types.LinkReconcileActionPayload {
	actions := make([]types.LinkReconcileActionPayload, 0, len(links))
	for _, l := range links {
		actions = append(actions, types.LinkReconcileActionPayload{
			LinkID:              l.LinkID,
			ChildPath:           l.ChildPath,
			Action:              string(l.RecommendedAction),
			NewStatus:           strconv.Itoa(int(l.NewStatus)),
			LinkDataUpdates:     l.LinkDataUpdates,
			IntermediaryActions: BuildIntermediaryActions(l.IntermediaryChanges),
		})
	}
	return actions
}

// BuildIntermediaryActions converts intermediary reconciliation results to action payloads.
func BuildIntermediaryActions(
	changes map[string]*container.IntermediaryReconcileResult,
) map[string]*types.IntermediaryReconcileActionPayload {
	if len(changes) == 0 {
		return nil
	}

	actions := make(map[string]*types.IntermediaryReconcileActionPayload, len(changes))
	for name, intResult := range changes {
		actions[name] = &types.IntermediaryReconcileActionPayload{
			Action:        string(container.ReconciliationActionAcceptExternal),
			ExternalState: intResult.ExternalState,
			NewStatus:     "created",
		}
	}
	return actions
}

// BuildDocumentInfo creates BlueprintDocumentInfo based on the source type.
func BuildDocumentInfo(source string, blueprintFile string) (types.BlueprintDocumentInfo, error) {
	switch source {
	case consts.BlueprintSourceHTTPS:
		return BuildHTTPSDocumentInfo(blueprintFile)
	case consts.BlueprintSourceS3:
		return BuildObjectStorageDocumentInfo(blueprintFile, "s3"), nil
	case consts.BlueprintSourceGCS:
		return BuildObjectStorageDocumentInfo(blueprintFile, "gcs"), nil
	case consts.BlueprintSourceAzureBlob:
		return BuildObjectStorageDocumentInfo(blueprintFile, "azureblob"), nil
	default:
		return BuildLocalFileDocumentInfo(blueprintFile)
	}
}

// BuildLocalFileDocumentInfo creates document info for local file sources.
func BuildLocalFileDocumentInfo(blueprintFile string) (types.BlueprintDocumentInfo, error) {
	absPath, err := filepath.Abs(blueprintFile)
	if err != nil {
		return types.BlueprintDocumentInfo{}, err
	}
	return types.BlueprintDocumentInfo{
		FileSourceScheme: "file",
		Directory:        filepath.Dir(absPath),
		BlueprintFile:    filepath.Base(absPath),
	}, nil
}

// BuildHTTPSDocumentInfo creates document info for HTTPS sources.
func BuildHTTPSDocumentInfo(blueprintFile string) (types.BlueprintDocumentInfo, error) {
	parsedURL, err := url.Parse(blueprintFile)
	if err != nil {
		return types.BlueprintDocumentInfo{}, err
	}

	basePath := path.Dir(parsedURL.Path)
	if basePath == "/" {
		basePath = ""
	}

	return types.BlueprintDocumentInfo{
		FileSourceScheme: "https",
		Directory:        basePath,
		BlueprintFile:    path.Base(parsedURL.Path),
		BlueprintLocationMetadata: map[string]any{
			"host": parsedURL.Host,
		},
	}, nil
}

// GetEffectiveInstanceID returns the instance ID, falling back to instance name if ID is empty.
func GetEffectiveInstanceID(instanceID, instanceName string) string {
	if instanceID != "" {
		return instanceID
	}
	return instanceName
}

// InstanceResolver provides the fields needed to resolve instance identifiers.
type InstanceResolver interface {
	GetInstanceID() string
	GetInstanceName() string
	GetEngine() InstanceLookup
}

// InstanceLookup is the interface for looking up blueprint instances.
type InstanceLookup interface {
	GetBlueprintInstance(ctx context.Context, instanceIDOrName string) (*state.InstanceState, error)
}

// ResolveInstanceIdentifiers looks up instance identifiers, returning the resolved ID and name.
// If an instance ID is already provided, it's returned as-is.
// If only a name is provided, it looks up the instance to get its ID.
// If the instance doesn't exist, empty strings are returned (indicating new deployment).
func ResolveInstanceIdentifiers(resolver InstanceResolver) (instanceID, instanceName string) {
	if resolver.GetInstanceID() != "" {
		return resolver.GetInstanceID(), resolver.GetInstanceName()
	}

	if resolver.GetInstanceName() == "" {
		return "", ""
	}

	instance, err := resolver.GetEngine().GetBlueprintInstance(context.TODO(), resolver.GetInstanceName())
	if err != nil || instance == nil {
		return "", ""
	}

	return instance.InstanceID, resolver.GetInstanceName()
}
