package objectstore

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/statestore"
)

// Completes the removal of an instance whose removal started but was interrupted, and does
// nothing for an instance that is live or absent. A removal only starts once a destroy has
// removed everything the instance deployed, so what is left is state, this includes
// the ID records the interruption left behind, the name record, and the marked instance object.
//
// The name record and the instance object are only deleted at the versions read here, so one
// that another process replaced in the meantime is left alone.
func finishRemoval(
	ctx context.Context,
	svc Service,
	keys statestore.KeyBuilder,
	instanceID string,
) error {
	var marked statestore.PersistedInstanceState
	markedETag, found, err := readVersionedEntity(ctx, svc, keys.Instance(instanceID), &marked)
	if err != nil || !found || !marked.Removing {
		return err
	}

	if err := deleteRecordsNaming(
		ctx, svc, instanceID, entityRecordKeys(keys, marked.EntityIDs()),
	); err != nil {
		return err
	}

	if marked.InstanceName != "" {
		if err := deleteNameRecordNaming(
			ctx, svc, keys.InstanceByName(marked.InstanceName), instanceID,
		); err != nil {
			return err
		}
	}

	return deleteAtVersion(ctx, svc, keys.Instance(instanceID), markedETag)
}

func deleteRecordsNaming(ctx context.Context, svc Service, instanceID string, recordKeys []string) error {
	for _, key := range recordKeys {
		var record statestore.EntityRecord
		found, err := readEntity(ctx, svc, key, &record)
		if err != nil {
			return err
		}

		if !found || record.InstanceID != instanceID {
			continue
		}

		if err := deleteIgnoringNotFound(ctx, svc, key); err != nil {
			return err
		}
	}
	return nil
}

func deleteNameRecordNaming(ctx context.Context, svc Service, key string, instanceID string) error {
	var record statestore.InstanceNameRecord
	etag, found, err := readVersionedEntity(ctx, svc, key, &record)
	if err != nil || !found || record.ID != instanceID {
		return err
	}
	return deleteAtVersion(ctx, svc, key, etag)
}

// An object that is gone or was replaced since it was read is left as it is.
func deleteAtVersion(ctx context.Context, svc Service, key string, etag string) error {
	err := svc.Delete(ctx, key, &DeleteOptions{IfMatch: etag})
	if err != nil && !isObjectNotFound(err) && !isPreconditionFailed(err) {
		return err
	}

	return nil
}

func entityRecordKeys(keys statestore.KeyBuilder, ids statestore.InstanceEntityIDs) []string {
	recordKeys := make([]string, 0, len(ids.Resources)+len(ids.Links))
	for _, resourceID := range ids.Resources {
		recordKeys = append(recordKeys, keys.ResourceRecord(resourceID))
	}

	for _, linkID := range ids.Links {
		recordKeys = append(recordKeys, keys.LinkRecord(linkID))
	}

	return recordKeys
}

func deleteIgnoringNotFound(ctx context.Context, svc Service, key string) error {
	if err := svc.Delete(ctx, key, nil); err != nil && !isObjectNotFound(err) {
		return err
	}
	return nil
}
