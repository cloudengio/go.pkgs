module cloudeng.io/google

go 1.27

require (
	cloudeng.io/algo v0.0.0-20260825050644-3d0fba22c536
	cloudeng.io/file v0.0.0-20260527194618-4cb6d4558850
	golang.org/x/oauth2 v0.36.0
	google.golang.org/api v0.293.0
)

require (
	cloud.google.com/go/auth v0.23.2 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.8 // indirect
	cloud.google.com/go/compute/metadata v0.9.0 // indirect
	cloudeng.io/errors v0.0.14-0.20260312171538-61fcde6ce278 // indirect
	cloudeng.io/sync v0.0.12-0.20260804222138-e9281ed260ba // indirect
	cloudeng.io/sys v0.0.0-20260825050644-3d0fba22c536 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/felixge/httpsnoop v1.1.0 // indirect
	github.com/go-logr/logr v1.4.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/s2a-go v0.1.9 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.21 // indirect
	github.com/googleapis/gax-go/v2 v2.24.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.70.0 // indirect
	go.opentelemetry.io/otel v1.46.0 // indirect
	go.opentelemetry.io/otel/metric v1.46.0 // indirect
	go.opentelemetry.io/otel/trace v1.46.0 // indirect
	golang.org/x/crypto v0.55.0 // indirect
	golang.org/x/net v0.58.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.41.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260825221802-da73d73af1c5 // indirect
	google.golang.org/grpc v1.83.2 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)

replace cloudeng.io/algo => ../algo

replace cloudeng.io/file => ../file

replace cloudeng.io/sync => ../sync

replace cloudeng.io/sys => ../sys

replace cloudeng.io/errors => ../errors
