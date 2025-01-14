module github.com/confluentinc/terraform-provider-confluent

go 1.22.7

require (
	github.com/confluentinc/ccloud-sdk-go-v2/apikeys v0.3.0
	github.com/confluentinc/ccloud-sdk-go-v2/byok v0.0.2
	github.com/confluentinc/ccloud-sdk-go-v2/certificate-authority v0.0.0-20240921001517-750d06dd7c27
	github.com/confluentinc/ccloud-sdk-go-v2/cmk v0.21.0
	github.com/confluentinc/ccloud-sdk-go-v2/connect v0.2.0
	github.com/confluentinc/ccloud-sdk-go-v2/connect-custom-plugin v0.0.2
	github.com/confluentinc/ccloud-sdk-go-v2/data-catalog v0.2.0
	github.com/confluentinc/ccloud-sdk-go-v2/flink v0.9.0
	github.com/confluentinc/ccloud-sdk-go-v2/flink-artifact v0.4.0
	github.com/confluentinc/ccloud-sdk-go-v2/flink-gateway v0.17.0
	github.com/confluentinc/ccloud-sdk-go-v2/iam v0.14.0
	github.com/confluentinc/ccloud-sdk-go-v2/identity-provider v0.2.0
	github.com/confluentinc/ccloud-sdk-go-v2/kafka-quotas v0.4.0
	github.com/confluentinc/ccloud-sdk-go-v2/kafkarest v0.14.0
	github.com/confluentinc/ccloud-sdk-go-v2/ksql v0.1.0
	github.com/confluentinc/ccloud-sdk-go-v2/mds v0.3.0
	github.com/confluentinc/ccloud-sdk-go-v2/networking v0.12.0
	github.com/confluentinc/ccloud-sdk-go-v2/networking-access-point v0.5.0
	github.com/confluentinc/ccloud-sdk-go-v2/networking-dnsforwarder v0.4.0
	github.com/confluentinc/ccloud-sdk-go-v2/networking-gateway v0.2.0
	github.com/confluentinc/ccloud-sdk-go-v2/networking-ip v0.2.0
	github.com/confluentinc/ccloud-sdk-go-v2/networking-privatelink v0.2.0
	github.com/confluentinc/ccloud-sdk-go-v2/org v0.9.0
	github.com/confluentinc/ccloud-sdk-go-v2/provider-integration v0.1.0
	github.com/confluentinc/ccloud-sdk-go-v2/schema-registry v0.4.0
	github.com/confluentinc/ccloud-sdk-go-v2/srcm v0.7.0
	github.com/confluentinc/ccloud-sdk-go-v2/sso v0.0.1
	github.com/dghubble/sling v1.4.1
	github.com/docker/go-connections v0.5.0
	github.com/hashicorp/go-cty v1.4.1-0.20200414143053-d3edf31b6320
	github.com/hashicorp/go-retryablehttp v0.7.7
	github.com/hashicorp/terraform-plugin-log v0.4.0
	github.com/hashicorp/terraform-plugin-sdk/v2 v2.16.0
	github.com/samber/lo v1.20.0
	github.com/testcontainers/testcontainers-go v0.32.0
	github.com/walkerus/go-wiremock v1.2.0
)

require (
	github.com/confluentinc/ccloud-sdk-go-v2-internal/networking-dnsforwarder v0.0.5 // indirect
	github.com/containerd/errdefs v0.1.0 // indirect
	github.com/sergi/go-diff v1.3.2-0.20230802210424-5b0b94c5c0d3 // indirect
    github.com/stretchr/testify v1.10.0 // indirect
)

require (
	dario.cat/mergo v1.0.0 // indirect
	github.com/Azure/go-ansiterm v0.0.0-20230124172434-306776ec8161 // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/Microsoft/hcsshim v0.12.2 // indirect
	github.com/agext/levenshtein v1.2.2 // indirect
	github.com/apparentlymart/go-textseg/v13 v13.0.0 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/containerd/containerd v1.7.18 // indirect
	github.com/containerd/log v0.1.0 // indirect
	github.com/cpuguy83/dockercfg v0.3.1 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/docker/docker v27.1.1+incompatible // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/fatih/color v1.16.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-git/go-git/v5 v5.13.0 // indirect
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/uuid v1.6.0
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-checkpoint v0.5.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-hclog v1.6.3 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-plugin v1.4.3 // indirect
	github.com/hashicorp/go-uuid v1.0.3 // indirect
	github.com/hashicorp/go-version v1.6.0 // indirect
	github.com/hashicorp/hc-install v0.3.2 // indirect
	github.com/hashicorp/hcl/v2 v2.16.2
	github.com/hashicorp/logutils v1.0.0 // indirect
	github.com/hashicorp/terraform-exec v0.16.1 // indirect
	github.com/hashicorp/terraform-json v0.13.0 // indirect
	github.com/hashicorp/terraform-plugin-go v0.9.0 // indirect
	github.com/hashicorp/terraform-registry-address v0.0.0-20210412075316-9b2996cce896 // indirect
	github.com/hashicorp/terraform-svchost v0.0.0-20200729002733-f050f53b9734 // indirect
	github.com/hashicorp/yamux v0.0.0-20181012175058-2f1d1f20f75d // indirect
	github.com/klauspost/compress v1.17.8 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/lufia/plan9stats v0.0.0-20240408141607-282e7b5d6b74 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/go-testing-interface v1.14.1 // indirect
	github.com/mitchellh/go-wordwrap v1.0.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/moby/patternmatcher v0.6.0 // indirect
	github.com/moby/sys/sequential v0.5.0 // indirect
	github.com/moby/sys/user v0.1.0 // indirect
	github.com/moby/term v0.5.0 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/nsf/jsondiff v0.0.0-20210926074059-1e845ec5d249 // indirect
	github.com/oklog/run v1.0.0 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/power-devops/perfstat v0.0.0-20240221224432-82ca36839d55 // indirect
	github.com/rogpeppe/go-internal v1.11.0 // indirect
	github.com/shirou/gopsutil/v3 v3.24.3 // indirect
	github.com/shoenig/go-m1cpu v0.1.6 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/tklauser/go-sysconf v0.3.13 // indirect
	github.com/tklauser/numcpus v0.7.0 // indirect
	github.com/vmihailenco/msgpack v4.0.4+incompatible // indirect
	github.com/vmihailenco/msgpack/v4 v4.3.12 // indirect
	github.com/vmihailenco/tagparser v0.1.2 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	github.com/zclconf/go-cty v1.12.1
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.50.0 // indirect
	go.opentelemetry.io/otel v1.25.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.24.0 // indirect
	go.opentelemetry.io/otel/metric v1.25.0 // indirect
	go.opentelemetry.io/otel/sdk v1.24.0 // indirect
	go.opentelemetry.io/otel/trace v1.25.0 // indirect
	go.opentelemetry.io/proto/otlp v1.1.0 // indirect
	golang.org/x/crypto v0.31.0 // indirect
	golang.org/x/exp v0.0.0-20240409090435-93d18d7e34b8 // indirect
	golang.org/x/net v0.33.0 // indirect
	golang.org/x/oauth2 v0.17.0 // indirect
	golang.org/x/sys v0.28.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	golang.org/x/time v0.5.0 // indirect
	google.golang.org/appengine v1.6.8 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240412170617-26222e5d3d56 // indirect
	google.golang.org/grpc v1.63.2 // indirect
	google.golang.org/protobuf v1.33.0 // indirect
)
