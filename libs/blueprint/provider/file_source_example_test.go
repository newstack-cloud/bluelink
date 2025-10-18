package provider_test

import (
	"context"
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// Example showing how to create a custom file source for S3.
// This demonstrates how host applications (like the deploy engine)
// can extend the file() function to support remote sources.
func ExampleFileSource_s3() {
	// Create a new file source registry
	registry := provider.NewFileSourceRegistry()

	// Create and register a custom S3 file source
	s3Source := &S3FileSource{
		// In a real implementation, you'd configure AWS credentials here
	}
	err := registry.Register(s3Source)
	if err != nil {
		panic(err)
	}

	// Now the registry can read from S3 URIs
	data, err := registry.ReadFile(context.Background(), "s3://my-bucket/path/to/file.txt")
	if err != nil {
		// Handle error
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Read %d bytes from S3\n", len(data))
	}
}

// Example showing how to register multiple remote sources
func ExampleFileSourceRegistry() {
	registry := provider.NewFileSourceRegistry()

	// Register S3 source
	s3Source := &S3FileSource{}
	registry.Register(s3Source)

	// Register GCS source
	gcsSource := &GCSFileSource{}
	registry.Register(gcsSource)

	// Register HTTP source
	httpSource := &HTTPFileSource{}
	registry.Register(httpSource)

	// Now the file() function can handle:
	// - Local files: file("path/to/local.txt")
	// - S3 files: file("s3://bucket/key")
	// - GCS files: file("gs://bucket/object")
	// - HTTP files: file("https://example.com/file.txt")

	// When used in FunctionCallContext, these will be available to the file() function
	fmt.Println("Multiple file sources registered")
	// Output: Multiple file sources registered
}

// S3FileSource is an example implementation for AWS S3
type S3FileSource struct {
	// Add AWS SDK client, credentials, etc.
}

func (s *S3FileSource) Scheme() string {
	return "s3"
}

func (s *S3FileSource) ReadFile(ctx context.Context, path string) ([]byte, error) {
	// In a real implementation:
	// 1. Parse the S3 URI to extract bucket and key
	// 2. Use AWS SDK to download the object
	// 3. Return the object data
	return nil, fmt.Errorf("S3 implementation not shown in example")
}

// GCSFileSource is an example implementation for Google Cloud Storage
type GCSFileSource struct {
	// Add GCS client, credentials, etc.
}

func (s *GCSFileSource) Scheme() string {
	return "gs"
}

func (s *GCSFileSource) ReadFile(ctx context.Context, path string) ([]byte, error) {
	// Similar to S3, but using GCS SDK
	return nil, fmt.Errorf("GCS implementation not shown in example")
}

// HTTPFileSource is an example implementation for HTTP/HTTPS
type HTTPFileSource struct {
	// Add HTTP client configuration
}

func (s *HTTPFileSource) Scheme() string {
	return "https"
}

func (s *HTTPFileSource) ReadFile(ctx context.Context, path string) ([]byte, error) {
	// Use http.Get() to fetch the file
	return nil, fmt.Errorf("HTTP implementation not shown in example")
}

// Example showing how to inject custom file source registry into function calls
func ExampleFunctionCallContext_withCustomFileSources() {
	// This would be done in the host application (e.g., deploy engine)
	// when initializing the blueprint execution context

	// 1. Create a custom file source registry
	fileRegistry := provider.NewFileSourceRegistry()

	// 2. Register custom file sources
	fileRegistry.Register(&S3FileSource{})
	fileRegistry.Register(&GCSFileSource{})

	// 3. When creating FunctionCallContext, pass the registry
	// (This would be done in the subengine or container package)
	// The context would then make it available to all function calls

	fmt.Println("Custom file sources configured")
	// Output: Custom file sources configured
}
