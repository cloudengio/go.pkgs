module cloudeng.io/aws

go 1.26.2

require (
	cloudeng.io/cicd v0.0.0-20260527194618-4cb6d4558850
	cloudeng.io/cmdutil v0.0.0-20260527194618-4cb6d4558850
	cloudeng.io/errors v0.0.14-0.20260312171538-61fcde6ce278
	cloudeng.io/file v0.0.0-20260527194618-4cb6d4558850
	cloudeng.io/logging v0.0.0-20260606211206-13a5cf17eb80
	cloudeng.io/path v0.0.10-0.20260312171538-61fcde6ce278
	cloudeng.io/text v0.0.16-0.20260312171538-61fcde6ce278
	github.com/alexbacchin/ssm-session-client v1.1.0
	github.com/aws/aws-sdk-go-v2 v1.42.0
	github.com/aws/aws-sdk-go-v2/config v1.32.25
	github.com/aws/aws-sdk-go-v2/credentials v1.19.24
	github.com/aws/aws-sdk-go-v2/feature/dsql/auth v1.1.29
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.307.0
	github.com/aws/aws-sdk-go-v2/service/kms v1.53.4
	github.com/aws/aws-sdk-go-v2/service/s3 v1.103.3
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.42.3
	github.com/aws/aws-sdk-go-v2/service/sesv2 v1.62.4
	github.com/aws/aws-sdk-go-v2/service/sts v1.43.3
	github.com/aws/smithy-go v1.27.2
	github.com/jackc/pgx/v5 v5.10.0
	github.com/orlangure/gnomock v0.32.0
)

require (
	github.com/aws/aws-sdk-go-v2/service/ssm v1.69.3 // indirect
	github.com/aws/session-manager-plugin v0.0.0-20260423192734-dcff8da8cdec // indirect
	github.com/cihub/seelog v0.0.0-20170130134532-f561c5e57575 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/fsnotify/fsnotify v1.10.1 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/moby/term v0.5.2 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/stretchr/objx v0.5.3 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	github.com/xtaci/smux v1.5.57 // indirect
	golang.org/x/crypto v0.53.0 // indirect
	golang.org/x/term v0.44.0 // indirect
	gotest.tools/v3 v3.5.2 // indirect
)

require (
	cloudeng.io/algo v0.0.0-20260606211206-13a5cf17eb80 // indirect
	cloudeng.io/os v0.0.0-20260606211206-13a5cf17eb80 // indirect
	cloudeng.io/sync v0.0.11
	cloudeng.io/sys v0.0.0-20260606211206-13a5cf17eb80 // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.13 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.29 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.29 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.29 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.4.30 // indirect
	github.com/aws/aws-sdk-go-v2/service/dsql v1.14.6
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.12 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.9.22 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.29 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.19.29 // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.2.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.31.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.36.6 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/containerd/errdefs v1.0.0 // indirect
	github.com/containerd/errdefs/pkg v0.3.0 // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/docker/docker v28.5.2+incompatible // indirect
	github.com/docker/go-connections v0.7.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.27.5 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/moby/sys/sequential v0.6.0 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/twinj/uuid v1.0.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.69.0 // indirect
	go.opentelemetry.io/otel v1.44.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.25.0 // indirect
	go.opentelemetry.io/otel/metric v1.44.0 // indirect
	go.opentelemetry.io/otel/trace v1.44.0 // indirect
	go.opentelemetry.io/proto/otlp v1.9.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.28.0 // indirect
	golang.org/x/net v0.56.0 // indirect
	golang.org/x/sync v0.21.0 // indirect
	golang.org/x/sys v0.46.0 // indirect
	golang.org/x/text v0.38.0 // indirect
	golang.org/x/time v0.15.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260319201613-d00831a3d3e7 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260610202329-623566214e0c // indirect
	google.golang.org/grpc v1.81.1 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
	gopkg.in/yaml.v3 v3.0.1
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
