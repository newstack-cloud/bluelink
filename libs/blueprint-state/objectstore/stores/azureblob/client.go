package azureblob

import (
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

// ClientOptions configures the account and service URL used when
// constructing an azblob.Client for the objectstore Service.
type ClientOptions struct {
	// ServiceURL is the account-qualified blob service URL, e.g.
	// "https://<account>.blob.core.windows.net" against real Azure or
	// "http://localhost:10000/devstoreaccount1" against Azurite.
	ServiceURL string
	// AccountName is the storage account name used for shared-key
	// signing, e.g. "devstoreaccount1" for Azurite.
	AccountName string
	// AccountKey is the base64-encoded shared key for the account.
	AccountKey string
	// Client lets callers override the azblob.ClientOptions — for
	// custom retry policies, HTTP transports, etc.
	Client *azblob.ClientOptions
}

// NewClient builds an *azblob.Client bound to a storage account via
// shared-key credentials. The caller still owns credential sourcing;
// this helper is intentionally thin so it can be reused against Azurite
// and any Azure-compatible emulator.
func NewClient(opts ClientOptions) (*azblob.Client, error) {
	cred, err := azblob.NewSharedKeyCredential(opts.AccountName, opts.AccountKey)
	if err != nil {
		return nil, err
	}
	return azblob.NewClientWithSharedKeyCredential(opts.ServiceURL, cred, opts.Client)
}
