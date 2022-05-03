module github.com/confluentinc/terraform-provider-ccloud

go 1.15

require (
	github.com/antihax/optional v1.0.0
	github.com/confluentinc/ccloud-sdk-go-v2-internal/cmk v0.0.9
	github.com/confluentinc/ccloud-sdk-go-v2/apikeys v0.1.0
	github.com/confluentinc/ccloud-sdk-go-v2/iam v0.7.0
	github.com/confluentinc/ccloud-sdk-go-v2/kafkarest v0.3.0
	github.com/confluentinc/ccloud-sdk-go-v2/mds v0.3.0
	github.com/confluentinc/ccloud-sdk-go-v2/networking v0.2.0
	github.com/confluentinc/ccloud-sdk-go-v2/org v0.4.0
	github.com/docker/go-connections v0.4.0
	github.com/hashicorp/go-retryablehttp v0.7.0
	github.com/hashicorp/terraform-plugin-log v0.3.0
	github.com/hashicorp/terraform-plugin-sdk/v2 v2.12.0
	github.com/stretchr/testify v1.7.0
	github.com/testcontainers/testcontainers-go v0.13.0
	github.com/walkerus/go-wiremock v1.2.0
)

replace (
	github.com/opencontainers/image-spec => github.com/opencontainers/image-spec v1.0.2
	github.com/opencontainers/runc => github.com/opencontainers/runc v1.0.3
	github.com/containerd/imgcrypt => github.com/containerd/imgcrypt v1.1.4
	github.com/buger/jsonparser => github.com/buger/jsonparser v1.0.0
	github.com/Microsoft/hcsshim => github.com/Microsoft/hcsshim v0.8.24
	github.com/dgrijalva/jwt-go => github.com/golang-jwt/jwt v3.2.1+incompatible
	github.com/containernetworking/cni => github.com/containernetworking/cni v0.8.1
	github.com/satori/go.uuid v1.2.0 => github.com/satori/go.uuid v1.2.1-0.20181016170032-d91630c85102
)
