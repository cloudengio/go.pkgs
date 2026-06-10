module cloudeng.io/vms/dockervm/dockerapi

go 1.26.2

require (
	cloudeng.io/cicd v0.0.0-20260527194618-4cb6d4558850
	cloudeng.io/os v0.0.0-20260527194618-4cb6d4558850
	cloudeng.io/vms v0.0.0-20260527194618-4cb6d4558850
	github.com/containerd/errdefs v1.0.0
	github.com/docker/go-sdk/client v0.1.0-alpha013
	github.com/moby/moby/api v1.54.2
	github.com/moby/moby/client v0.4.1
)

require (
	cloudeng.io/errors v0.0.14-0.20260312171538-61fcde6ce278 // indirect
	cloudeng.io/sync v0.0.11 // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/caarlos0/env/v11 v11.4.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/containerd/errdefs/pkg v0.3.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/docker/go-connections v0.7.0 // indirect
	github.com/docker/go-sdk/config v0.1.0-alpha013 // indirect
	github.com/docker/go-sdk/context v0.1.0-alpha013 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.69.0 // indirect
	go.opentelemetry.io/otel v1.44.0 // indirect
	go.opentelemetry.io/otel/metric v1.44.0 // indirect
	go.opentelemetry.io/otel/trace v1.44.0 // indirect
	golang.org/x/sync v0.21.0 // indirect
	golang.org/x/sys v0.46.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace cloudeng.io/vms => ../..

replace cloudeng.io/errors => ../../../errors

replace cloudeng.io/os => ../../../os
