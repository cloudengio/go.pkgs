module cloudeng.io/vms/dockervm/dockerapi

go 1.26.2

require (
	cloudeng.io/algo v0.0.0-20260824023931-9b6c51abac7f
	cloudeng.io/cicd v0.0.0-20260824023931-9b6c51abac7f
	cloudeng.io/os v0.0.0-20260527194618-4cb6d4558850
	cloudeng.io/vms v0.0.0-20260527194618-4cb6d4558850
	github.com/containerd/errdefs v1.0.0
	github.com/docker/go-sdk/client v0.1.0-alpha013
	github.com/moby/moby/api v1.55.0
	github.com/moby/moby/client v0.5.1
)

require (
	cloudeng.io/errors v0.0.14-0.20260312171538-61fcde6ce278 // indirect
	cloudeng.io/sync v0.0.12-0.20260804222138-e9281ed260ba // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/caarlos0/env/v11 v11.4.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/containerd/errdefs/pkg v0.3.0 // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/docker/go-connections v0.8.1 // indirect
	github.com/docker/go-sdk/config v0.1.0-alpha013 // indirect
	github.com/docker/go-sdk/context v0.1.0-alpha013 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/felixge/httpsnoop v1.1.0 // indirect
	github.com/go-logr/logr v1.4.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.1 // indirect
	github.com/stretchr/testify v1.12.1 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.70.0 // indirect
	go.opentelemetry.io/otel v1.45.0 // indirect
	go.opentelemetry.io/otel/metric v1.45.0 // indirect
	go.opentelemetry.io/otel/trace v1.45.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
)

replace cloudeng.io/vms => ../..

replace cloudeng.io/errors => ../../../errors

replace cloudeng.io/os => ../../../os

replace cloudeng.io/algo => ../../../algo
