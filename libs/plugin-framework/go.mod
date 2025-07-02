module github.com/newstack-cloud/bluelink/libs/plugin-framework

go 1.23

toolchain go1.23.4

replace github.com/newstack-cloud/bluelink/libs/blueprint => ../blueprint

require (
	github.com/google/uuid v1.6.0
	github.com/newstack-cloud/bluelink/libs/blueprint v0.25.0
	github.com/newstack-cloud/bluelink/libs/common v0.3.2
	github.com/spf13/afero v1.11.0
	github.com/stretchr/testify v1.10.0
	google.golang.org/grpc v1.64.0
	google.golang.org/protobuf v1.34.1
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/coreos/go-json v0.0.0-20231102161613-e49c8866685a // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/matoous/go-nanoid/v2 v2.1.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/tailscale/hujson v0.0.0-20250226034555-ec1d1c113d33 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/net v0.22.0 // indirect
	golang.org/x/sys v0.18.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240318140521-94a12d6c2237 // indirect
)
