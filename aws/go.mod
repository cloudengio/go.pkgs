module cloudeng.io/aws

go 1.27

require (
	cloudeng.io/cicd v0.0.0-20260527194618-4cb6d4558850
	cloudeng.io/cmdutil v0.0.0-20260527194618-4cb6d4558850
	cloudeng.io/errors v0.0.14-0.20260312171538-61fcde6ce278
	cloudeng.io/file v0.0.0-20260824023931-9b6c51abac7f
	cloudeng.io/logging v0.0.0-20260824023931-9b6c51abac7f
	cloudeng.io/path v0.0.10-0.20260312171538-61fcde6ce278
	cloudeng.io/sync v0.0.12-0.20260804222138-e9281ed260ba
	cloudeng.io/text v0.0.16-0.20260624171915-da98fe9dec2b
	github.com/alexbacchin/ssm-session-client v1.3.0
	github.com/aws/aws-sdk-go-v2 v1.45.1
	github.com/aws/aws-sdk-go-v2/config v1.33.2
	github.com/aws/aws-sdk-go-v2/credentials v1.20.2
	github.com/aws/aws-sdk-go-v2/feature/dsql/auth v1.2.1
	github.com/aws/aws-sdk-go-v2/service/dsql v1.20.0
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.328.0
	github.com/aws/aws-sdk-go-v2/service/kms v1.58.0
	github.com/aws/aws-sdk-go-v2/service/s3 v1.110.0
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.47.0
	github.com/aws/aws-sdk-go-v2/service/sesv2 v1.71.0
	github.com/aws/aws-sdk-go-v2/service/sts v1.48.0
	github.com/aws/smithy-go v1.28.1
	github.com/jackc/pgx/v5 v5.10.0
	github.com/orlangure/gnomock v0.32.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	cloudeng.io/algo v0.0.0-20260903161432-3e39c500cdbf // indirect
	cloudeng.io/os v0.0.0-20260825050644-3d0fba22c536 // indirect
	cloudeng.io/sys v0.0.0-20260825050644-3d0fba22c536 // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.20 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.19.1 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.5.1 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.8.1 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.5.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.19 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.11.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.14.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.20.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.8.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssm v1.76.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.36.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.41.0 // indirect
	github.com/aws/session-manager-plugin v0.0.0-20260615221425-930a08e65d3a // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cihub/seelog v0.0.0-20170130134532-f561c5e57575 // indirect
	github.com/containerd/errdefs v1.0.0 // indirect
	github.com/containerd/errdefs/pkg v0.3.0 // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/docker/docker v28.5.2+incompatible // indirect
	github.com/docker/go-connections v0.8.1 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/felixge/httpsnoop v1.1.0 // indirect
	github.com/fsnotify/fsnotify v1.10.1 // indirect
	github.com/go-logr/logr v1.4.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.27.5 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/moby/sys/sequential v0.6.0 // indirect
	github.com/moby/term v0.5.2 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/stretchr/objx v0.5.3 // indirect
	github.com/stretchr/testify v1.12.1 // indirect
	github.com/twinj/uuid v1.0.0 // indirect
	github.com/xtaci/smux v1.5.57 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.71.0 // indirect
	go.opentelemetry.io/otel v1.46.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.25.0 // indirect
	go.opentelemetry.io/otel/metric v1.46.0 // indirect
	go.opentelemetry.io/otel/trace v1.46.0 // indirect
	go.opentelemetry.io/proto/otlp v1.9.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.28.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/crypto v0.56.0 // indirect
	golang.org/x/net v0.58.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/term v0.45.0 // indirect
	golang.org/x/text v0.41.0 // indirect
	golang.org/x/time v0.15.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260715232425-e75dac1f907d // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260831171406-18b4a7587f8a // indirect
	google.golang.org/grpc v1.83.2 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
	gotest.tools/v3 v3.5.2 // indirect
)

replace github.com/twinj/uuid => github.com/twinj/uuid v0.0.0-20151029044442-89173bcdda19

replace cloudeng.io/cicd => ../cicd

replace cloudeng.io/cmdutil => ../cmdutil

replace cloudeng.io/errors => ../errors

replace cloudeng.io/file => ../file

replace cloudeng.io/logging => ../logging

replace cloudeng.io/path => ../path

replace cloudeng.io/text => ../text

replace cloudeng.io/os => ../os

replace cloudeng.io/sync => ../sync

replace cloudeng.io/algo => ../algo
