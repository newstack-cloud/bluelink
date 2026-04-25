package azureblob

import (
	"context"
	"io"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint-state/objectstore"
)

// Service is the Azure Blob Storage-backed objectstore.Service
// implementation. It maps SDK errors to objectstore.Error with a
// ReasonCode so the shared state layer can reason about not-found vs.
// precondition-failed vs. auth / rate limit without depending on Azure
// types.
//
// The ETag string in the objectstore.Service contract carries the blob's
// Azure ETag verbatim (quoted form as returned by the service) for this
// backend — callers treat the value as opaque, passing it back through
// PutOptions.IfMatch to drive conditional writes via the
// ModifiedAccessConditions.IfMatch header.
type Service struct {
	client    *azblob.Client
	container string
}

// NewService constructs an Azure Blob Service bound to the given
// container. The caller owns the azblob.Client lifecycle (credentials,
// service URL, retry policy etc.).
func NewService(client *azblob.Client, container string) *Service {
	return &Service{client: client, container: container}
}

func (s *Service) Get(ctx context.Context, key string) ([]byte, string, error) {
	resp, err := s.client.DownloadStream(ctx, s.container, key, nil)
	if err != nil {
		return nil, "", mapErr(err, "get "+key)
	}
	defer resp.Body.Close()

	data, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, "", readErr
	}
	return data, etagString(resp.ETag), nil
}

func (s *Service) Put(
	ctx context.Context,
	key string,
	data []byte,
	opts *objectstore.PutOptions,
) (string, error) {
	uploadOpts := &azblob.UploadBufferOptions{
		AccessConditions: uploadAccessConditions(opts),
	}
	resp, err := s.client.UploadBuffer(ctx, s.container, key, data, uploadOpts)
	if err != nil {
		return "", mapErr(err, "put "+key)
	}
	return etagString(resp.ETag), nil
}

func (s *Service) Delete(ctx context.Context, key string) error {
	if _, err := s.client.DeleteBlob(ctx, s.container, key, nil); err != nil {
		return mapErr(err, "delete "+key)
	}
	return nil
}

func (s *Service) Head(ctx context.Context, key string) (*objectstore.ObjectInfo, error) {
	blobClient := s.client.ServiceClient().
		NewContainerClient(s.container).
		NewBlobClient(key)
	resp, err := blobClient.GetProperties(ctx, nil)
	if err != nil {
		return nil, mapErr(err, "head "+key)
	}
	return &objectstore.ObjectInfo{
		Key:  key,
		Size: derefInt64(resp.ContentLength),
		ETag: etagString(resp.ETag),
	}, nil
}

func (s *Service) List(ctx context.Context, prefix string) ([]*objectstore.ObjectInfo, error) {
	listOpts := &azblob.ListBlobsFlatOptions{}
	if prefix != "" {
		listOpts.Prefix = to.Ptr(prefix)
	}
	pager := s.client.NewListBlobsFlatPager(s.container, listOpts)

	var infos []*objectstore.ObjectInfo
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, mapErr(err, "list "+prefix)
		}
		infos = append(infos, blobItemsToInfos(page.Segment.BlobItems)...)
	}
	return infos, nil
}

func blobItemsToInfos(items []*container.BlobItem) []*objectstore.ObjectInfo {
	infos := make([]*objectstore.ObjectInfo, 0, len(items))
	for _, item := range items {
		info := &objectstore.ObjectInfo{
			Key: derefString(item.Name),
		}
		if item.Properties != nil {
			info.Size = derefInt64(item.Properties.ContentLength)
			info.ETag = etagString(item.Properties.ETag)
		}
		infos = append(infos, info)
	}
	return infos
}

// Applies IfMatch / IfNoneMatch semantics from the
// objectstore.PutOptions onto the blob.AccessConditions. Azure uses
// string ETag CAS; the "*" IfNoneMatch sentinel maps to azcore.ETagAny.
func uploadAccessConditions(opts *objectstore.PutOptions) *blob.AccessConditions {
	if opts == nil || (opts.IfMatch == "" && opts.IfNoneMatch == "") {
		return nil
	}

	conds := &blob.ModifiedAccessConditions{}
	if opts.IfNoneMatch == "*" {
		conds.IfNoneMatch = to.Ptr(azcore.ETagAny)
	} else if opts.IfNoneMatch != "" {
		conds.IfNoneMatch = to.Ptr(azcore.ETag(opts.IfNoneMatch))
	}

	if opts.IfMatch != "" {
		conds.IfMatch = to.Ptr(azcore.ETag(opts.IfMatch))
	}

	return &blob.AccessConditions{ModifiedAccessConditions: conds}
}

// Classifies an SDK error onto a ReasonCode so the
// shared state layer can match on a stable surface. Both the 409
// BlobAlreadyExists (create-if-absent conflict) and the 412
// ConditionNotMet (IfMatch stale) paths collapse to precondition_failed
// — the distinction isn't important for CAS callers.
func mapErr(err error, ctx string) error {
	if err == nil {
		return nil
	}
	switch {
	case bloberror.HasCode(
		err,
		bloberror.BlobNotFound,
		bloberror.ContainerNotFound,
	):
		return objectstore.NewObjectNotFound(ctx)
	case bloberror.HasCode(
		err,
		bloberror.ConditionNotMet,
		bloberror.TargetConditionNotMet,
		bloberror.SourceConditionNotMet,
		bloberror.BlobAlreadyExists,
	):
		return objectstore.NewPreconditionFailed(ctx)
	case bloberror.HasCode(
		err,
		bloberror.AuthenticationFailed,
		bloberror.AuthorizationFailure,
		bloberror.InsufficientAccountPermissions,
		bloberror.InvalidAuthenticationInfo,
	):
		return objectstore.NewAuthFailed(ctx)
	case bloberror.HasCode(
		err,
		bloberror.ServerBusy,
		bloberror.OperationTimedOut,
	):
		return objectstore.NewRateLimited(ctx)
	}
	return err
}

func etagString(etag *azcore.ETag) string {
	if etag == nil {
		return ""
	}
	return string(*etag)
}

func derefInt64(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}

func derefString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
