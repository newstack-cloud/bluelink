package awskms

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kms"
)

// ClientOptions configures client-level overrides when constructing a KMS
// client. Credential resolution stays with the caller through aws.Config.
type ClientOptions struct {
	// Endpoint overrides the default KMS endpoint, for a local emulator or a VPC
	// endpoint. Empty means use the default.
	Endpoint string
}

// NewClient builds a KMS client from an aws.Config and the given overrides,
// mirroring the shape used by the objectstore S3 store so a deployment can
// share credential-loading conventions between the two.
func NewClient(conf aws.Config, opts ClientOptions) *kms.Client {
	return kms.NewFromConfig(conf, func(o *kms.Options) {
		if opts.Endpoint != "" {
			o.BaseEndpoint = aws.String(opts.Endpoint)
		}
	})
}
