// Copyright 2021 Confluent Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package provider

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/dghubble/sling"
	"github.com/google/uuid"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	flinkgatewayinternalv1 "github.com/confluentinc/ccloud-sdk-go-v2-internal/flink-gateway/v1"
	apikeysv2 "github.com/confluentinc/ccloud-sdk-go-v2/apikeys/v2"
	byokv1 "github.com/confluentinc/ccloud-sdk-go-v2/byok/v1"
	camv1 "github.com/confluentinc/ccloud-sdk-go-v2/cam/v1"
	ccpmv1 "github.com/confluentinc/ccloud-sdk-go-v2/ccpm/v1"
	certificateauthorityv2 "github.com/confluentinc/ccloud-sdk-go-v2/certificate-authority/v2"
	cmkv2 "github.com/confluentinc/ccloud-sdk-go-v2/cmk/v2"
	connectcustompluginv1 "github.com/confluentinc/ccloud-sdk-go-v2/connect-custom-plugin/v1"
	connectv1 "github.com/confluentinc/ccloud-sdk-go-v2/connect/v1"
	datacatalogv1 "github.com/confluentinc/ccloud-sdk-go-v2/data-catalog/v1"
	endpointv1 "github.com/confluentinc/ccloud-sdk-go-v2/endpoint/v1"
	flinkartifactv1 "github.com/confluentinc/ccloud-sdk-go-v2/flink-artifact/v1"
	flinkgatewayv1 "github.com/confluentinc/ccloud-sdk-go-v2/flink-gateway/v1"
	flinkv2 "github.com/confluentinc/ccloud-sdk-go-v2/flink/v2"
	iamipfilteringv2 "github.com/confluentinc/ccloud-sdk-go-v2/iam-ip-filtering/v2"
	iamv1 "github.com/confluentinc/ccloud-sdk-go-v2/iam/v1"
	iamv2 "github.com/confluentinc/ccloud-sdk-go-v2/iam/v2"
	identityproviderv2 "github.com/confluentinc/ccloud-sdk-go-v2/identity-provider/v2"
	kafkaquotasv1 "github.com/confluentinc/ccloud-sdk-go-v2/kafka-quotas/v1"
	kafkarestv3 "github.com/confluentinc/ccloud-sdk-go-v2/kafkarest/v3"
	ksqlv2 "github.com/confluentinc/ccloud-sdk-go-v2/ksql/v2"
	mdsv2 "github.com/confluentinc/ccloud-sdk-go-v2/mds/v2"
	networkingaccesspointv1 "github.com/confluentinc/ccloud-sdk-go-v2/networking-access-point/v1"
	networkingdnsforwarderv1 "github.com/confluentinc/ccloud-sdk-go-v2/networking-dnsforwarder/v1"
	networkinggatewayv1 "github.com/confluentinc/ccloud-sdk-go-v2/networking-gateway/v1"
	networkingipv1 "github.com/confluentinc/ccloud-sdk-go-v2/networking-ip/v1"
	networkingprivatelinkv1 "github.com/confluentinc/ccloud-sdk-go-v2/networking-privatelink/v1"
	networkingv1 "github.com/confluentinc/ccloud-sdk-go-v2/networking/v1"
	orgv2 "github.com/confluentinc/ccloud-sdk-go-v2/org/v2"
	providerintegrationv1 "github.com/confluentinc/ccloud-sdk-go-v2/provider-integration/v1"
	providerintegrationv2 "github.com/confluentinc/ccloud-sdk-go-v2/provider-integration/v2"
	schemaregistryv1 "github.com/confluentinc/ccloud-sdk-go-v2/schema-registry/v1"
	srcmv3 "github.com/confluentinc/ccloud-sdk-go-v2/srcm/v3"
	ssov2 "github.com/confluentinc/ccloud-sdk-go-v2/sso/v2"
	tableflowv1 "github.com/confluentinc/ccloud-sdk-go-v2/tableflow/v1"
)

func (c *Client) apiKeysV2ApiContext(ctx context.Context) context.Context {
	if c.oauthToken != nil && c.stsToken != nil {
		if err := c.fetchOrOverrideSTSOAuthTokenFromApiContext(ctx); err != nil {
			tflog.Error(ctx, fmt.Sprintf("Failed to get OAuth token for API Key client: %v", err))
		}
		return context.WithValue(ctx, apikeysv2.ContextAccessToken, c.stsToken.AccessToken)
	}

	if c.cloudApiKey != "" && c.cloudApiSecret != "" {
		return context.WithValue(ctx, apikeysv2.ContextBasicAuth, apikeysv2.BasicAuth{
			UserName: c.cloudApiKey,
			Password: c.cloudApiSecret,
		})
	}

	tflog.Warn(ctx, "Could not find Cloud API Key or OAuth Token for API Key client")
	return ctx
}

func (c *Client) byokV1ApiContext(ctx context.Context) context.Context {
	if c.oauthToken != nil && c.stsToken != nil {
		if err := c.fetchOrOverrideSTSOAuthTokenFromApiContext(ctx); err != nil {
			tflog.Error(ctx, fmt.Sprintf("Failed to get OAuth token for BYOK client: %v", err))
		}
		return context.WithValue(ctx, byokv1.ContextAccessToken, c.stsToken.AccessToken)
	}

	if c.cloudApiKey != "" && c.cloudApiSecret != "" {
		return context.WithValue(ctx, byokv1.ContextBasicAuth, byokv1.BasicAuth{
			UserName: c.cloudApiKey,
			Password: c.cloudApiSecret,
		})
	}

	tflog.Warn(ctx, "Could not find Cloud API Key or OAuth Token for BYOK client")
	return ctx
}

func (c *KafkaRestClient) kafkaRestApiContextWithClusterApiKey(ctx context.Context, kafkaApiKey string, kafkaApiSecret string) context.Context {
	// This is for API key sync status check function only, so we can skip the OAuth token option here.
	if kafkaApiKey != "" && kafkaApiSecret != "" {
		return context.WithValue(ctx, kafkarestv3.ContextBasicAuth, kafkarestv3.BasicAuth{
			UserName: kafkaApiKey,
			Password: kafkaApiSecret,
		})
	}

	tflog.Warn(ctx, "Could not find Cloud API Key for Kafka Rest API client")
	return ctx
}

func (c *Client) connectCustomPluginV1ApiContext(ctx context.Context) context.Context {
	if c.oauthToken != nil && c.stsToken != nil {
		if err := c.fetchOrOverrideSTSOAuthTokenFromApiContext(ctx); err != nil {
			tflog.Error(ctx, fmt.Sprintf("Failed to get OAuth token for Custom Code Logging client: %v", err))
		}
		return context.WithValue(ctx, connectcustompluginv1.ContextAccessToken, c.stsToken.AccessToken)
	}

	if c.cloudApiKey != "" && c.cloudApiSecret != "" {
		return context.WithValue(ctx, connectcustompluginv1.ContextBasicAuth, connectcustompluginv1.BasicAuth{
			UserName: c.cloudApiKey,
			Password: c.cloudApiSecret,
		})
	}

	tflog.Warn(ctx, "Could not find Cloud API Key or OAuth Token for Custom Code Logging client")
	return ctx
}

func (c *Client) ccpmV1ApiContext(ctx context.Context) context.Context {
	if c.oauthToken != nil && c.stsToken != nil {
		if err := c.fetchOrOverrideSTSOAuthTokenFromApiContext(ctx); err != nil {
			tflog.Error(ctx, fmt.Sprintf("Failed to get OAuth token for Custom Code Logging client: %v", err))
		}
		return context.WithValue(ctx, connectcustompluginv1.ContextAccessToken, c.stsToken.AccessToken)
	}

	if c.cloudApiKey != "" && c.cloudApiSecret != "" {
		return context.WithValue(ctx, ccpmv1.ContextBasicAuth, ccpmv1.BasicAuth{
			UserName: c.cloudApiKey,
			Password: c.cloudApiSecret,
		})
	}

	tflog.Warn(ctx, "Could not find Cloud API Key or OAuth Token for Custom Code Logging client")
	return ctx
}

func (c *Client) cmkV2ApiContext(ctx context.Context) context.Context {
	if c.oauthToken != nil && c.stsToken != nil {
		if err := c.fetchOrOverrideSTSOAuthTokenFromApiContext(ctx); err != nil {
			tflog.Error(ctx, fmt.Sprintf("Failed to get OAuth token for Kafka Cluster client: %v", err))
		}
		return context.WithValue(ctx, cmkv2.ContextAccessToken, c.stsToken.AccessToken)
	}

	if c.cloudApiKey != "" && c.cloudApiSecret != "" {
		return context.WithValue(ctx, cmkv2.ContextBasicAuth, cmkv2.BasicAuth{
			UserName: c.cloudApiKey,
			Password: c.cloudApiSecret,
		})
	}

	tflog.Warn(ctx, "Could not find Cloud API Key or OAuth Token for Kafka Cluster client")
	return ctx
}

func (c *Client) iamV2ApiContext(ctx context.Context) context.Context {
	if c.oauthToken != nil && c.stsToken != nil {
		if err := c.fetchOrOverrideSTSOAuthTokenFromApiContext(ctx); err != nil {
			tflog.Error(ctx, fmt.Sprintf("Failed to get OAuth token for IAM client: %v", err))
		}
		return context.WithValue(ctx, iamv2.ContextAccessToken, c.stsToken.AccessToken)
	}

	if c.cloudApiKey != "" && c.cloudApiSecret != "" {
		return context.WithValue(ctx, iamv2.ContextBasicAuth, iamv2.BasicAuth{
			UserName: c.cloudApiKey,
			Password: c.cloudApiSecret,
		})
	}

	tflog.Warn(ctx, "Could not find Cloud API Key or OAuth Token for IAM client")
	return ctx
}

func (c *Client) iamIpFilteringV2ApiContext(ctx context.Context) context.Context {
	if c.oauthToken != nil && c.stsToken != nil {
		if err := c.fetchOrOverrideSTSOAuthTokenFromApiContext(ctx); err != nil {
			tflog.Error(ctx, fmt.Sprintf("Failed to get OAuth token for IAM IP client: %v", err))
		}
		return context.WithValue(ctx, iamipfilteringv2.ContextAccessToken, c.stsToken.AccessToken)
	}

	if c.cloudApiKey != "" && c.cloudApiSecret != "" {
		return context.WithValue(ctx, iamipfilteringv2.ContextBasicAuth, iamipfilteringv2.BasicAuth{
			UserName: c.cloudApiKey,
			Password: c.cloudApiSecret,
		})
	}

	tflog.Warn(ctx, "Could not find Cloud API Key or OAuth Token for IAM IP client")
	return ctx
}

func (c *Client) certificateAuthorityV2ApiContext(ctx context.Context) context.Context {
	if c.oauthToken != nil && c.stsToken != nil {
		if err := c.fetchOrOverrideSTSOAuthTokenFromApiContext(ctx); err != nil {
			tflog.Error(ctx, fmt.Sprintf("Failed to get OAuth token for Certificate Authorities client: %v", err))
		}
		return context.WithValue(ctx, certificateauthorityv2.ContextAccessToken, c.stsToken.AccessToken)
	}

	if c.cloudApiKey != "" && c.cloudApiSecret != "" {
		return context.WithValue(ctx, certificateauthorityv2.ContextBasicAuth, certificateauthorityv2.BasicAuth{
			UserName: c.cloudApiKey,
			Password: c.cloudApiSecret,
		})
	}

	tflog.Warn(ctx, "Could not find Cloud API Key or OAuth Token for Certificate Authorities client")
	return ctx
}

func (c *Client) camV1ApiContext(ctx context.Context) context.Context {
	if c.cloudApiKey != "" && c.cloudApiSecret != "" {
		return context.WithValue(context.Background(), camv1.ContextBasicAuth, camv1.BasicAuth{
			UserName: c.cloudApiKey,
			Password: c.cloudApiSecret,
		})
	}
	tflog.Warn(ctx, "Could not find Cloud API Key")
	return ctx
}

func (c *Client) ssoV2ApiContext(ctx context.Context) context.Context {
	if c.oauthToken != nil && c.stsToken != nil {
		if err := c.fetchOrOverrideSTSOAuthTokenFromApiContext(ctx); err != nil {
			tflog.Error(ctx, fmt.Sprintf("Failed to get OAuth token for SSO client: %v", err))
		}
		return context.WithValue(ctx, ssov2.ContextAccessToken, c.stsToken.AccessToken)
	}

	if c.cloudApiKey != "" && c.cloudApiSecret != "" {
		return context.WithValue(ctx, ssov2.ContextBasicAuth, ssov2.BasicAuth{
			UserName: c.cloudApiKey,
			Password: c.cloudApiSecret,
		})
	}

	tflog.Warn(ctx, "Could not find Cloud API Key or OAuth Token for SSO client")
	return ctx
}

func (c *Client) providerIntegrationV1ApiContext(ctx context.Context) context.Context {
	if c.oauthToken != nil && c.stsToken != nil {
		if err := c.fetchOrOverrideSTSOAuthTokenFromApiContext(ctx); err != nil {
			tflog.Error(ctx, fmt.Sprintf("Failed to get OAuth token for Provider Integration client: %v", err))
		}
		return context.WithValue(ctx, providerintegrationv1.ContextAccessToken, c.stsToken.AccessToken)
	}

	if c.cloudApiKey != "" && c.cloudApiSecret != "" {
		return context.WithValue(ctx, providerintegrationv1.ContextBasicAuth, providerintegrationv1.BasicAuth{
			UserName: c.cloudApiKey,
			Password: c.cloudApiSecret,
		})
	}

	tflog.Warn(ctx, "Could not find Cloud API Key or OAuth Token for Provider Integration client")
	return ctx
}

func (c *Client) providerIntegrationV2ApiContext(ctx context.Context) context.Context {
	if c.oauthToken != nil && c.stsToken != nil {
		if err := c.fetchOrOverrideSTSOAuthTokenFromApiContext(ctx); err != nil {
			tflog.Error(ctx, fmt.Sprintf("Failed to get OAuth token for Provider Integration v2 client: %v", err))
		}
		return context.WithValue(ctx, providerintegrationv2.ContextAccessToken, c.stsToken.AccessToken)
	}

	if c.cloudApiKey != "" && c.cloudApiSecret != "" {
		return context.WithValue(ctx, providerintegrationv2.ContextBasicAuth, providerintegrationv2.BasicAuth{
			UserName: c.cloudApiKey,
			Password: c.cloudApiSecret,
		})
	}

	tflog.Warn(ctx, "Could not find Cloud API Key or OAuth Token for Provider Integration v2 client")
	return ctx
}

func (c *Client) iamV1ApiContext(ctx context.Context) context.Context {
	if c.oauthToken != nil && c.stsToken != nil {
		if err := c.fetchOrOverrideSTSOAuthTokenFromApiContext(ctx); err != nil {
			tflog.Error(ctx, fmt.Sprintf("Failed to get OAuth token for IAM v1 client: %v", err))
		}
		return context.WithValue(ctx, iamv1.ContextAccessToken, c.stsToken.AccessToken)
	}

	if c.cloudApiKey != "" && c.cloudApiSecret != "" {
		return context.WithValue(ctx, iamv1.ContextBasicAuth, iamv1.BasicAuth{
			UserName: c.cloudApiKey,
			Password: c.cloudApiSecret,
		})
	}

	tflog.Warn(ctx, "Could not find Cloud API Key or OAuth Token for IAM v1 client")
	return ctx
}

func (c *Client) mdsV2ApiContext(ctx context.Context) context.Context {
	if c.oauthToken != nil && c.stsToken != nil {
		if err := c.fetchOrOverrideSTSOAuthTokenFromApiContext(ctx); err != nil {
			tflog.Error(ctx, fmt.Sprintf("Failed to get OAuth token for MDS client: %v", err))
		}
		return context.WithValue(ctx, mdsv2.ContextAccessToken, c.stsToken.AccessToken)
	}

	if c.cloudApiKey != "" && c.cloudApiSecret != "" {
		return context.WithValue(ctx, mdsv2.ContextBasicAuth, mdsv2.BasicAuth{
			UserName: c.cloudApiKey,
			Password: c.cloudApiSecret,
		})
	}

	tflog.Warn(ctx, "Could not find Cloud API Key or OAuth Token for MDS client")
	return ctx
}

func (c *Client) networkingV1ApiContext(ctx context.Context) context.Context {
	if c.oauthToken != nil && c.stsToken != nil {
		if err := c.fetchOrOverrideSTSOAuthTokenFromApiContext(ctx); err != nil {
			tflog.Error(ctx, fmt.Sprintf("Failed to get OAuth token for Networking client: %v", err))
		}
		return context.WithValue(ctx, networkingv1.ContextAccessToken, c.stsToken.AccessToken)
	}

	if c.cloudApiKey != "" && c.cloudApiSecret != "" {
		return context.WithValue(ctx, networkingv1.ContextBasicAuth, networkingv1.BasicAuth{
			UserName: c.cloudApiKey,
			Password: c.cloudApiSecret,
		})
	}

	tflog.Warn(ctx, "Could not find Cloud API Key or OAuth Token for Networking client")
	return ctx
}

func (c *Client) flinkArtifactV1ApiContext(ctx context.Context) context.Context {
	if c.oauthToken != nil && c.stsToken != nil {
		if err := c.fetchOrOverrideSTSOAuthTokenFromApiContext(ctx); err != nil {
			tflog.Error(ctx, fmt.Sprintf("Failed to get OAuth token for Flink artifact client: %v", err))
		}
		return context.WithValue(ctx, flinkartifactv1.ContextAccessToken, c.stsToken.AccessToken)
	}

	if c.cloudApiKey != "" && c.cloudApiSecret != "" {
		return context.WithValue(ctx, flinkartifactv1.ContextBasicAuth, flinkartifactv1.BasicAuth{
			UserName: c.cloudApiKey,
			Password: c.cloudApiSecret,
		})
	}

	tflog.Warn(ctx, "Could not find Cloud API Key or OAuth Token for Flink Artifact client")
	return ctx
}

func (c *Client) flinkV2ApiContext(ctx context.Context) context.Context {
	if c.oauthToken != nil && c.stsToken != nil {
		if err := c.fetchOrOverrideSTSOAuthTokenFromApiContext(ctx); err != nil {
			tflog.Error(ctx, fmt.Sprintf("Failed to get OAuth token for Flink client: %v", err))
		}
		return context.WithValue(ctx, flinkv2.ContextAccessToken, c.stsToken.AccessToken)
	}

	if c.cloudApiKey != "" && c.cloudApiSecret != "" {
		return context.WithValue(ctx, flinkv2.ContextBasicAuth, flinkv2.BasicAuth{
			UserName: c.cloudApiKey,
			Password: c.cloudApiSecret,
		})
	}

	tflog.Warn(ctx, "Could not find Cloud API Key or OAuth Token for Flink client")
	return ctx
}

func (c *Client) networkingAccessPointV1ApiContext(ctx context.Context) context.Context {
	if c.oauthToken != nil && c.stsToken != nil {
		if err := c.fetchOrOverrideSTSOAuthTokenFromApiContext(ctx); err != nil {
			tflog.Error(ctx, fmt.Sprintf("Failed to get OAuth token for network access point client: %v", err))
		}
		return context.WithValue(ctx, networkingaccesspointv1.ContextAccessToken, c.stsToken.AccessToken)
	}

	if c.cloudApiKey != "" && c.cloudApiSecret != "" {
		return context.WithValue(ctx, networkingaccesspointv1.ContextBasicAuth, networkingaccesspointv1.BasicAuth{
			UserName: c.cloudApiKey,
			Password: c.cloudApiSecret,
		})
	}

	tflog.Warn(ctx, "Could not find Cloud API Key or OAuth Token for Network Access Point client")
	return ctx
}

func (c *Client) networkingGatewayV1ApiContext(ctx context.Context) context.Context {
	if c.oauthToken != nil && c.stsToken != nil {
		if err := c.fetchOrOverrideSTSOAuthTokenFromApiContext(ctx); err != nil {
			tflog.Error(ctx, fmt.Sprintf("Failed to get OAuth token for network gateway client: %v", err))
		}
		return context.WithValue(ctx, networkinggatewayv1.ContextAccessToken, c.stsToken.AccessToken)
	}

	if c.cloudApiKey != "" && c.cloudApiSecret != "" {
		return context.WithValue(ctx, networkinggatewayv1.ContextBasicAuth, networkinggatewayv1.BasicAuth{
			UserName: c.cloudApiKey,
			Password: c.cloudApiSecret,
		})
	}

	tflog.Warn(ctx, "Could not find Cloud API Key or OAuth Token for Network Gateway client")
	return ctx
}

func (c *Client) networkingIpV1ApiContext(ctx context.Context) context.Context {
	if c.oauthToken != nil && c.stsToken != nil {
		if err := c.fetchOrOverrideSTSOAuthTokenFromApiContext(ctx); err != nil {
			tflog.Error(ctx, fmt.Sprintf("Failed to get OAuth token for network IP client: %v", err))
		}
		return context.WithValue(ctx, networkingipv1.ContextAccessToken, c.stsToken.AccessToken)
	}

	if c.cloudApiKey != "" && c.cloudApiSecret != "" {
		return context.WithValue(ctx, networkingipv1.ContextBasicAuth, networkingipv1.BasicAuth{
			UserName: c.cloudApiKey,
			Password: c.cloudApiSecret,
		})
	}

	tflog.Warn(ctx, "Could not find Cloud API Key or OAuth Token for Network IP client")
	return ctx
}

func (c *Client) networkingPrivatelinkV1ApiContext(ctx context.Context) context.Context {
	if c.oauthToken != nil && c.stsToken != nil {
		if err := c.fetchOrOverrideSTSOAuthTokenFromApiContext(ctx); err != nil {
			tflog.Error(ctx, fmt.Sprintf("Failed to get OAuth token for network private link client: %v", err))
		}
		return context.WithValue(ctx, networkingprivatelinkv1.ContextAccessToken, c.stsToken.AccessToken)
	}

	if c.cloudApiKey != "" && c.cloudApiSecret != "" {
		return context.WithValue(ctx, networkingprivatelinkv1.ContextBasicAuth, networkingprivatelinkv1.BasicAuth{
			UserName: c.cloudApiKey,
			Password: c.cloudApiSecret,
		})
	}

	tflog.Warn(ctx, "Could not find Cloud API Key or OAuth Token for Network Private Link client")
	return ctx
}

func (c *Client) networkingDnsforwarderV1ApiContext(ctx context.Context) context.Context {
	if c.oauthToken != nil && c.stsToken != nil {
		if err := c.fetchOrOverrideSTSOAuthTokenFromApiContext(ctx); err != nil {
			tflog.Error(ctx, fmt.Sprintf("Failed to get OAuth token for DNS client: %v", err))
		}
		return context.WithValue(ctx, networkingdnsforwarderv1.ContextAccessToken, c.stsToken.AccessToken)
	}

	if c.cloudApiKey != "" && c.cloudApiSecret != "" {
		return context.WithValue(ctx, networkingdnsforwarderv1.ContextBasicAuth, networkingdnsforwarderv1.BasicAuth{
			UserName: c.cloudApiKey,
			Password: c.cloudApiSecret,
		})
	}

	tflog.Warn(ctx, "Could not find Cloud API Key or OAuth Token for Network DNS Forwarder client")
	return ctx
}

func (c *Client) endpointV1ApiContext(ctx context.Context) context.Context {
	if c.oauthToken != nil && c.stsToken != nil {
		if err := c.fetchOrOverrideSTSOAuthTokenFromApiContext(ctx); err != nil {
			tflog.Error(ctx, fmt.Sprintf("Failed to get OAuth token for Endpoint client: %v", err))
		}
		return context.WithValue(ctx, endpointv1.ContextAccessToken, c.stsToken.AccessToken)
	}

	if c.cloudApiKey != "" && c.cloudApiSecret != "" {
		return context.WithValue(ctx, endpointv1.ContextBasicAuth, endpointv1.BasicAuth{
			UserName: c.cloudApiKey,
			Password: c.cloudApiSecret,
		})
	}

	tflog.Warn(ctx, "Could not find Cloud API Key or OAuth Token for Endpoint client")
	return ctx
}

func (c *Client) srcmV3ApiContext(ctx context.Context) context.Context {
	if c.oauthToken != nil && c.stsToken != nil {
		if err := c.fetchOrOverrideSTSOAuthTokenFromApiContext(ctx); err != nil {
			tflog.Error(ctx, fmt.Sprintf("Failed to get OAuth token for SRCM client: %v", err))
		}
		return context.WithValue(ctx, srcmv3.ContextAccessToken, c.stsToken.AccessToken)
	}

	if c.cloudApiKey != "" && c.cloudApiSecret != "" {
		return context.WithValue(ctx, srcmv3.ContextBasicAuth, srcmv3.BasicAuth{
			UserName: c.cloudApiKey,
			Password: c.cloudApiSecret,
		})
	}

	tflog.Warn(ctx, "Could not find Cloud API Key for Schema Registry Clusters client")
	return ctx
}

func (c *Client) connectV1ApiContext(ctx context.Context) context.Context {
	if c.oauthToken != nil && c.stsToken != nil {
		if err := c.fetchOrOverrideSTSOAuthTokenFromApiContext(ctx); err != nil {
			tflog.Error(ctx, fmt.Sprintf("Failed to get OAuth token for Connect client: %v", err))
		}
		return context.WithValue(ctx, connectv1.ContextAccessToken, c.stsToken.AccessToken)
	}

	if c.cloudApiKey != "" && c.cloudApiSecret != "" {
		return context.WithValue(ctx, connectv1.ContextBasicAuth, connectv1.BasicAuth{
			UserName: c.cloudApiKey,
			Password: c.cloudApiSecret,
		})
	}

	tflog.Warn(ctx, "Could not find Cloud API Key or OAuth Token for Connect client")
	return ctx
}

func (c *Client) orgV2ApiContext(ctx context.Context) context.Context {
	if c.oauthToken != nil && c.stsToken != nil {
		if err := c.fetchOrOverrideSTSOAuthTokenFromApiContext(ctx); err != nil {
			tflog.Error(ctx, fmt.Sprintf("Failed to get OAuth token for Organization client: %v", err))
		}
		return context.WithValue(ctx, orgv2.ContextAccessToken, c.stsToken.AccessToken)
	}

	if c.cloudApiKey != "" && c.cloudApiSecret != "" {
		return context.WithValue(ctx, orgv2.ContextBasicAuth, orgv2.BasicAuth{
			UserName: c.cloudApiKey,
			Password: c.cloudApiSecret,
		})
	}

	tflog.Warn(ctx, "Could not find Cloud API Key or OAuth Token for Organization client")
	return ctx
}

func (c *Client) ksqlV2ApiContext(ctx context.Context) context.Context {
	if c.oauthToken != nil && c.stsToken != nil {
		if err := c.fetchOrOverrideSTSOAuthTokenFromApiContext(ctx); err != nil {
			tflog.Error(ctx, fmt.Sprintf("Failed to get OAuth token for KSQL client: %v", err))
		}
		return context.WithValue(ctx, ksqlv2.ContextAccessToken, c.stsToken.AccessToken)
	}

	if c.cloudApiKey != "" && c.cloudApiSecret != "" {
		return context.WithValue(ctx, ksqlv2.ContextBasicAuth, ksqlv2.BasicAuth{
			UserName: c.cloudApiKey,
			Password: c.cloudApiSecret,
		})
	}

	tflog.Warn(ctx, "Could not find Cloud API Key or OAuth Token for KSQL client")
	return ctx
}

func (c *Client) identityProviderV2ApiContext(ctx context.Context) context.Context {
	if c.oauthToken != nil && c.stsToken != nil {
		if err := c.fetchOrOverrideSTSOAuthTokenFromApiContext(ctx); err != nil {
			tflog.Error(ctx, fmt.Sprintf("Failed to get OAuth token for Identity Provider client: %v", err))
		}
		return context.WithValue(ctx, identityproviderv2.ContextAccessToken, c.stsToken.AccessToken)
	}

	if c.cloudApiKey != "" && c.cloudApiSecret != "" {
		return context.WithValue(ctx, identityproviderv2.ContextBasicAuth, identityproviderv2.BasicAuth{
			UserName: c.cloudApiKey,
			Password: c.cloudApiSecret,
		})
	}

	tflog.Warn(ctx, "Could not find Cloud API Key or OAuth Token for Identity Provider client")
	return ctx
}

func (c *Client) kafkaQuotasV1ApiContext(ctx context.Context) context.Context {
	if c.oauthToken != nil && c.stsToken != nil {
		if err := c.fetchOrOverrideSTSOAuthTokenFromApiContext(ctx); err != nil {
			tflog.Error(ctx, fmt.Sprintf("Failed to get OAuth token for Kafka Quotas client: %v", err))
		}
		return context.WithValue(ctx, kafkaquotasv1.ContextAccessToken, c.stsToken.AccessToken)
	}

	if c.cloudApiKey != "" && c.cloudApiSecret != "" {
		return context.WithValue(ctx, kafkaquotasv1.ContextBasicAuth, kafkaquotasv1.BasicAuth{
			UserName: c.cloudApiKey,
			Password: c.cloudApiSecret,
		})
	}

	tflog.Warn(ctx, "Could not find Cloud API Key or OAuth Token for Kafka Quotas client")
	return ctx
}

func orgV2ApiContext(ctx context.Context, cloudApiKey, cloudApiSecret string) context.Context {
	if cloudApiKey != "" && cloudApiSecret != "" {
		return context.WithValue(ctx, orgv2.ContextBasicAuth, orgv2.BasicAuth{
			UserName: cloudApiKey,
			Password: cloudApiSecret,
		})
	}

	tflog.Warn(ctx, "Cloud API Key or Cloud API Secret is empty")
	return ctx
}

func getTimeoutFor(clusterType string) time.Duration {
	if clusterType == kafkaClusterTypeDedicated {
		return 72 * time.Hour
	} else {
		return 1 * time.Hour
	}
}

func stringToAclResourceType(aclResourceType string) (kafkarestv3.AclResourceType, error) {
	switch aclResourceType {
	case "UNKNOWN":
		return kafkarestv3.UNKNOWN, nil
	case "ANY":
		return kafkarestv3.ANY, nil
	case "TOPIC":
		return kafkarestv3.TOPIC, nil
	case "GROUP":
		return kafkarestv3.GROUP, nil
	case "CLUSTER":
		return kafkarestv3.CLUSTER, nil
	case "TRANSACTIONAL_ID":
		return kafkarestv3.TRANSACTIONAL_ID, nil
	case "DELEGATION_TOKEN":
		return kafkarestv3.DELEGATION_TOKEN, nil
	}
	return "", fmt.Errorf("unknown ACL resource type was found: %q", aclResourceType)
}

type Acl struct {
	ResourceType kafkarestv3.AclResourceType
	ResourceName string
	PatternType  string
	Principal    string
	Host         string
	Operation    string
	Permission   string
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

type KafkaRestClient struct {
	apiClient                     *kafkarestv3.APIClient
	externalAccessToken           *OAuthToken
	clusterId                     string
	clusterApiKey                 string
	clusterApiSecret              string
	restEndpoint                  string
	isMetadataSetInProviderBlock  bool
	isClusterIdSetInProviderBlock bool
}

type SchemaRegistryRestClient struct {
	apiClient                    *schemaregistryv1.APIClient
	externalAccessToken          *OAuthToken
	clusterId                    string
	clusterApiKey                string
	clusterApiSecret             string
	restEndpoint                 string
	isMetadataSetInProviderBlock bool
}

type CatalogRestClient struct {
	apiClient                    *datacatalogv1.APIClient
	externalAccessToken          *OAuthToken
	clusterId                    string
	clusterApiKey                string
	clusterApiSecret             string
	restEndpoint                 string
	isMetadataSetInProviderBlock bool
}

type FlinkRestClient struct {
	apiClientInternal            *flinkgatewayinternalv1.APIClient
	apiClient                    *flinkgatewayv1.APIClient
	externalAccessToken          *OAuthToken
	organizationId               string
	environmentId                string
	computePoolId                string
	principalId                  string
	flinkApiKey                  string
	flinkApiSecret               string
	restEndpoint                 string
	isMetadataSetInProviderBlock bool
}

type TableflowRestClient struct {
	apiClient                    *tableflowv1.APIClient
	oauthToken                   *OAuthToken
	stsToken                     *STSToken
	tableflowApiKey              string
	tableflowApiSecret           string
	isMetadataSetInProviderBlock bool
}

func (c *KafkaRestClient) apiContext(ctx context.Context) context.Context {
	if c.externalAccessToken != nil {
		currToken := c.externalAccessToken
		token, err := fetchExternalOAuthToken(ctx, currToken.TokenUrl, currToken.ClientId, currToken.ClientSecret, currToken.Scope, currToken.IdentityPoolId, currToken, currToken.HTTPClient)
		if err != nil {
			tflog.Error(ctx, fmt.Sprintf("Failed to get OAuth token for Kafka rest client: %v", err))
		}
		c.externalAccessToken = token
		return context.WithValue(ctx, kafkarestv3.ContextAccessToken, c.externalAccessToken.AccessToken)
	}

	if c.clusterApiKey != "" && c.clusterApiSecret != "" {
		return context.WithValue(ctx, kafkarestv3.ContextBasicAuth, kafkarestv3.BasicAuth{
			UserName: c.clusterApiKey,
			Password: c.clusterApiSecret,
		})
	}

	tflog.Warn(ctx, fmt.Sprintf("Could not find Kafka API Key or OAuth token for Kafka Cluster %q", c.clusterId), map[string]interface{}{kafkaClusterLoggingKey: c.clusterId})
	return ctx
}

func (c *SchemaRegistryRestClient) apiContext(ctx context.Context) context.Context {
	if c.externalAccessToken != nil {
		currToken := c.externalAccessToken
		token, err := fetchExternalOAuthToken(ctx, currToken.TokenUrl, currToken.ClientId, currToken.ClientSecret, currToken.Scope, currToken.IdentityPoolId, currToken, currToken.HTTPClient)
		if err != nil {
			tflog.Error(ctx, fmt.Sprintf("Failed to get OAuth token for Schema Registry rest client: %v", err))
		}
		c.externalAccessToken = token
		return context.WithValue(ctx, schemaregistryv1.ContextAccessToken, c.externalAccessToken.AccessToken)
	}

	if c.clusterApiKey != "" && c.clusterApiSecret != "" {
		return context.WithValue(ctx, schemaregistryv1.ContextBasicAuth, schemaregistryv1.BasicAuth{
			UserName: c.clusterApiKey,
			Password: c.clusterApiSecret,
		})
	}

	tflog.Warn(ctx, fmt.Sprintf("Could not find Schema Registry API Key or OAuth token for Stream Governance Cluster %q", c.clusterId))
	return ctx
}

func (c *SchemaRegistryRestClient) dataCatalogV1ApiContext(ctx context.Context) context.Context {
	if c.externalAccessToken != nil {
		currToken := c.externalAccessToken
		token, err := fetchExternalOAuthToken(ctx, currToken.TokenUrl, currToken.ClientId, currToken.ClientSecret, currToken.Scope, currToken.IdentityPoolId, currToken, currToken.HTTPClient)
		if err != nil {
			tflog.Error(ctx, fmt.Sprintf("Failed to get OAuth token for Data Catalog rest client: %v", err))
		}
		c.externalAccessToken = token
		return context.WithValue(ctx, datacatalogv1.ContextAccessToken, c.externalAccessToken.AccessToken)
	}

	if c.clusterApiKey != "" && c.clusterApiSecret != "" {
		return context.WithValue(ctx, datacatalogv1.ContextBasicAuth, datacatalogv1.BasicAuth{
			UserName: c.clusterApiKey,
			Password: c.clusterApiSecret,
		})
	}

	tflog.Warn(ctx, fmt.Sprintf("Could not find Schema Registry API Key or OAuth token for Stream Governance Cluster %q", c.clusterId))
	return ctx
}

func (c *CatalogRestClient) dataCatalogV1ApiContext(ctx context.Context) context.Context {
	if c.externalAccessToken != nil {
		currToken := c.externalAccessToken
		token, err := fetchExternalOAuthToken(ctx, currToken.TokenUrl, currToken.ClientId, currToken.ClientSecret, currToken.Scope, currToken.IdentityPoolId, currToken, currToken.HTTPClient)
		if err != nil {
			tflog.Error(ctx, fmt.Sprintf("Failed to get OAuth token for Stream Governance Cluster rest client: %v", err))
		}
		c.externalAccessToken = token
		return context.WithValue(ctx, datacatalogv1.ContextAccessToken, c.externalAccessToken.AccessToken)
	}

	if c.clusterApiKey != "" && c.clusterApiSecret != "" {
		return context.WithValue(ctx, datacatalogv1.ContextBasicAuth, datacatalogv1.BasicAuth{
			UserName: c.clusterApiKey,
			Password: c.clusterApiSecret,
		})
	}
	tflog.Warn(ctx, fmt.Sprintf("Could not find Catalog API Key or OAuth token for Stream Governance Cluster %q", c.clusterId))
	return ctx
}

func (c *FlinkRestClient) apiContext(ctx context.Context) context.Context {
	if c.externalAccessToken != nil {
		currToken := c.externalAccessToken
		token, err := fetchExternalOAuthToken(ctx, currToken.TokenUrl, currToken.ClientId, currToken.ClientSecret, currToken.Scope, currToken.IdentityPoolId, currToken, currToken.HTTPClient)
		if err != nil {
			tflog.Error(ctx, fmt.Sprintf("Failed to get OAuth token for Flink rest client: %v", err))
		}
		c.externalAccessToken = token
		return context.WithValue(ctx, flinkgatewayv1.ContextAccessToken, c.externalAccessToken.AccessToken)
	}

	if c.flinkApiKey != "" && c.flinkApiSecret != "" {
		return context.WithValue(ctx, flinkgatewayv1.ContextBasicAuth, flinkgatewayv1.BasicAuth{
			UserName: c.flinkApiKey,
			Password: c.flinkApiSecret,
		})
	}

	tflog.Warn(ctx, fmt.Sprintf("Could not find Flink API Key or OAuth token for Flink %q", c.restEndpoint))
	return ctx
}

func (c *FlinkRestClient) fgApiContext(ctx context.Context) context.Context {
	if c.externalAccessToken != nil {
		currToken := c.externalAccessToken
		token, err := fetchExternalOAuthToken(ctx, currToken.TokenUrl, currToken.ClientId, currToken.ClientSecret, currToken.Scope, currToken.IdentityPoolId, currToken, currToken.HTTPClient)
		if err != nil {
			tflog.Error(ctx, fmt.Sprintf("Failed to get OAuth token for Flink rest client: %v", err))
		}
		if token != nil {
			c.externalAccessToken = token
			return context.WithValue(ctx, flinkgatewayinternalv1.ContextAccessToken, c.externalAccessToken.AccessToken)
		}
		if currToken.AccessToken != "" {
			return context.WithValue(ctx, flinkgatewayinternalv1.ContextAccessToken, currToken.AccessToken)
		}
		tflog.Warn(ctx, fmt.Sprintf("Could not find Flink OAuth token for Flink %q", c.restEndpoint))
		return ctx
	}

	if c.flinkApiKey != "" && c.flinkApiSecret != "" {
		return context.WithValue(ctx, flinkgatewayinternalv1.ContextBasicAuth, flinkgatewayinternalv1.BasicAuth{
			UserName: c.flinkApiKey,
			Password: c.flinkApiSecret,
		})
	}

	tflog.Warn(ctx, fmt.Sprintf("Could not find Flink API Key or OAuth token for Flink %q", c.restEndpoint))
	return ctx
}

// TODO: Tableflow APIs don't support OAuth at this moment, following up in CLI-3534 for OAuth GA
func (c *TableflowRestClient) apiContext(ctx context.Context) context.Context {
	if c.tableflowApiKey != "" && c.tableflowApiSecret != "" {
		return context.WithValue(ctx, tableflowv1.ContextBasicAuth, tableflowv1.BasicAuth{
			UserName: c.tableflowApiKey,
			Password: c.tableflowApiSecret,
		})
	}
	tflog.Warn(ctx, fmt.Sprintf("Could not find Tableflow API Key for Tableflow client"))
	return ctx
}

type GenericOpenAPIError interface {
	Model() interface{}
}

func setStringAttributeInListBlockOfSizeOne(blockName, attributeName, attributeValue string, d *schema.ResourceData) error {
	return d.Set(blockName, []interface{}{map[string]interface{}{
		attributeName: attributeValue,
	}})
}

// createDescriptiveError will convert GenericOpenAPIError error into an error with a more descriptive error message.
// diag.FromErr(createDescriptiveError(err)) should be used instead of diag.FromErr(err) in this project
// since GenericOpenAPIError.Error() returns just HTTP status code and its generic name (i.e., "400 Bad Request")
func createDescriptiveError(err error, resp ...*http.Response) error {
	if err == nil {
		return nil
	}
	// At this point it's just status code and its generic name
	errorMessage := err.Error()
	// Add error.detail to the final error message
	if genericOpenAPIError, ok := err.(GenericOpenAPIError); ok {
		failure := genericOpenAPIError.Model()
		reflectedFailure := reflect.ValueOf(&failure).Elem().Elem()
		reflectedFailureValue := reflect.Indirect(reflectedFailure)
		if reflectedFailureValue.IsValid() {
			errs := reflectedFailureValue.FieldByName("Errors")
			kafkaRestOrConnectErr := reflectedFailureValue.FieldByName("Message")
			if errs.IsValid() && errs.Kind() == reflect.Slice && errs.Len() > 0 {
				nest := errs.Index(0)
				detailPtr := nest.FieldByName("Detail")
				if detailPtr.IsValid() && detailPtr.Kind() == reflect.Pointer && !detailPtr.IsNil() {
					errorMessage = fmt.Sprintf("%s: %s", errorMessage, reflect.Indirect(detailPtr))
				}
			} else if kafkaRestOrConnectErr.IsValid() && kafkaRestOrConnectErr.Kind() == reflect.Struct {
				detailPtr := kafkaRestOrConnectErr.FieldByName("value")
				if detailPtr.IsValid() && detailPtr.Kind() == reflect.Pointer && !detailPtr.IsNil() {
					errorMessage = fmt.Sprintf("%s: %s", errorMessage, reflect.Indirect(detailPtr))
				}
			} else if kafkaRestOrConnectErr.IsValid() && kafkaRestOrConnectErr.Kind() == reflect.Pointer &&
				!kafkaRestOrConnectErr.IsNil() {
				errorMessage = fmt.Sprintf("%s: %s", errorMessage, reflect.Indirect(kafkaRestOrConnectErr))
			}
		}
	}

	// If a *http.Response was provided, and we could not parse Error object,
	// read its *http.Response body to provide a more descriptive error message to avoid
	// https://github.com/confluentinc/terraform-provider-confluent/issues/53
	if errorMessage == err.Error() && len(resp) > 0 && resp[0] != nil && resp[0].Body != nil {
		defer resp[0].Body.Close()

		bodyBytes, readErr := io.ReadAll(resp[0].Body)
		if readErr == nil {
			// Check if the response looks like gzip (magic bytes 0x1f 0x8b)
			// This handles cases where Content-Encoding header is missing
			if len(bodyBytes) >= 2 && bodyBytes[0] == 0x1f && bodyBytes[1] == 0x8b {
				gzipReader, gzipErr := gzip.NewReader(bytes.NewReader(bodyBytes))
				if gzipErr == nil {
					defer gzipReader.Close()
					decompressedBytes, decompressErr := io.ReadAll(gzipReader)
					if decompressErr == nil {
						bodyBytes = decompressedBytes
					}
				}
			}

			errorMessage = fmt.Sprintf(
				"%s; could not parse error details; raw response body: %#v",
				errorMessage,
				string(bodyBytes),
			)
		}
	}

	return errors.New(errorMessage)
}

// Reports whether the response has http.StatusForbidden status due to an invalid Cloud API Key vs other reasons
// which is useful to distinguish from scenarios where http.StatusForbidden represents http.StatusNotFound for
// security purposes.
func ResponseHasStatusForbiddenDueToInvalidAPIKey(response *http.Response) bool {
	if ResponseHasExpectedStatusCode(response, http.StatusForbidden) {
		bodyBytes, err := io.ReadAll(response.Body)
		if err != nil {
			return false
		}
		bodyString := string(bodyBytes)
		// Search for a specific error message that indicates the invalid Cloud API Key has been used
		return strings.Contains(bodyString, "invalid API key")
	}
	return false
}

func ResponseHasExpectedStatusCode(response *http.Response, expectedStatusCode int) bool {
	return response != nil && response.StatusCode == expectedStatusCode
}

func isNonKafkaRestApiResourceNotFound(response *http.Response) bool {
	return ResponseHasExpectedStatusCode(response, http.StatusNotFound) ||
		(ResponseHasExpectedStatusCode(response, http.StatusForbidden) && !ResponseHasStatusForbiddenDueToInvalidAPIKey(response))
}

// APIF-2043: TEMPORARY METHOD
// Converts principal with a resourceID (User:sa-01234) to principal with an integer ID (User:6789)
func principalWithResourceIdToPrincipalWithIntegerId(c *Client, principalWithResourceId string) (string, error) {
	// There's input validation that principal attribute must start with "User:sa-" or "User:u-" or "User:pool-" r "User:group-" or "User:*"

	if principalWithResourceId == "User:*" {
		return principalWithResourceId, nil
	}

	// User:sa-abc123 -> sa-abc123
	resourceId := principalWithResourceId[5:]
	if strings.HasPrefix(principalWithResourceId, "User:sa-") {
		integerId, err := saResourceIdToSaIntegerId(c, resourceId)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s%d", principalPrefix, integerId), nil
	} else if strings.HasPrefix(principalWithResourceId, "User:u-") {
		integerId, err := userResourceIdToUserIntegerId(c, resourceId)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s%d", principalPrefix, integerId), nil
	} else if strings.HasPrefix(principalWithResourceId, "User:pool-") || strings.HasPrefix(principalWithResourceId, "User:group-") {
		return principalWithResourceId, nil
	}
	return "", fmt.Errorf("the principal must start with 'User:sa-' or 'User:u-' or 'User:pool-' or 'User:group-' or 'User:*'")
}

// APIF-2043: TEMPORARY METHOD
// Converts service account's resourceID (sa-abc123) to its integer ID (67890)
func saResourceIdToSaIntegerId(c *Client, saResourceId string) (int, error) {
	list, _, err := c.iamV1Client.ServiceAccountsV1Api.ListV1ServiceAccounts(c.iamV1ApiContext(context.Background())).Execute()
	if err != nil {
		return 0, err
	}
	for _, sa := range list.GetUsers() {
		if sa.GetResourceId() == saResourceId {
			if sa.HasId() {
				return int(sa.GetId()), nil
			} else {
				return 0, fmt.Errorf("the matching integer ID for a service account with resource ID=%s is nil", saResourceId)
			}
		}
	}
	return 0, fmt.Errorf("the service account with resource ID=%s was not found", saResourceId)
}

// APIF-2043: TEMPORARY METHOD
// Converts user's resourceID (u-abc123) to its integer ID (67890)
func userResourceIdToUserIntegerId(c *Client, userResourceId string) (int, error) {
	list, _, err := c.iamV1Client.UsersV1Api.ListV1Users(c.iamV1ApiContext(context.Background())).Execute()
	if err != nil {
		return 0, err
	}
	for _, user := range list.GetUsers() {
		if user.GetResourceId() == userResourceId {
			if user.HasId() {
				return int(user.GetId()), nil
			} else {
				return 0, fmt.Errorf("the matching integer ID for a user with resource ID=%s is nil", userResourceId)
			}
		}
	}
	return 0, fmt.Errorf("the user with resource ID=%s was not found", userResourceId)
}

func clusterCrnToRbacClusterCrn(clusterCrn string) (string, error) {
	// Converts
	// crn://confluent.cloud/organization=./environment=./cloud-cluster=lkc-198rjz/kafka=lkc-198rjz
	// to
	// crn://confluent.cloud/organization=./environment=./cloud-cluster=lkc-198rjz
	lastIndex := strings.LastIndex(clusterCrn, crnKafkaSuffix)
	if lastIndex == -1 {
		return "", fmt.Errorf("could not find %s in %s", crnKafkaSuffix, clusterCrn)
	}
	return clusterCrn[:lastIndex], nil
}

func convertToStringStringMap(data map[string]interface{}) map[string]string {
	stringMap := make(map[string]string)

	for key, value := range data {
		stringMap[key] = value.(string)
	}

	return stringMap
}

func convertToStringStringListMap(data []interface{}) map[string][]string {
	stringListMap := make(map[string][]string)
	for _, item := range data {
		kv := item.(map[string]interface{})
		key := kv[paramKey].(string)
		value := convertToStringSlice(kv[paramValue].(*schema.Set).List())
		stringListMap[key] = value
	}
	return stringListMap
}

func normalizeCrn(crn string) string {
	if v, err := url.PathUnescape(crn); err == nil {
		return v
	}
	return crn
}

func ptr(s string) *string {
	return &s
}

func kafkaClusterBlockV0() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			paramKafkaCluster: kafkaClusterIdSchema(),
		},
	}
}

func kafkaClusterBlockStateUpgradeV0(ctx context.Context, rawState map[string]interface{}, meta interface{}) (map[string]interface{}, error) {
	kafkaClusterIdString := rawState[paramKafkaCluster].(string)
	rawState[paramKafkaCluster] = []interface{}{map[string]interface{}{
		paramId: kafkaClusterIdString,
	}}
	return rawState, nil
}

func (c *Client) fetchOrOverrideExternalOAuthTokenFromApiContext(ctx context.Context) error {
	currToken := c.oauthToken
	token, err := fetchExternalOAuthToken(ctx, currToken.TokenUrl, currToken.ClientId, currToken.ClientSecret, currToken.Scope, currToken.IdentityPoolId, currToken, currToken.HTTPClient)
	if err != nil {
		return err
	}
	c.oauthToken = token
	return nil
}

func (c *Client) fetchOrOverrideSTSOAuthTokenFromApiContext(ctx context.Context) error {
	currExternalToken := c.oauthToken
	currSTSToken := c.stsToken

	// Check if the current STS OAuth token is still valid
	if valid := validateCurrentSTSOAuthToken(ctx, currSTSToken); valid {
		return nil
	}

	// Check if the current external OAuth token is still valid
	// If valid, request a new STS token based on the current external OAuth token
	if valid := validateCurrentExternalOAuthToken(ctx, currExternalToken); valid {
		stsToken, err := requestNewSTSOAuthToken(ctx, currExternalToken.AccessToken, currSTSToken.IdentityPoolId, currSTSToken.ExpiresInSeconds, currSTSToken.STSClient)
		if err != nil {
			return err
		}
		c.stsToken = stsToken
		return nil
	}

	// If invalid, request a new external OAuth token first
	// then request a new STS token based on the current external OAuth token
	externalToken, err := requestNewExternalOAuthToken(ctx, currExternalToken.TokenUrl, currExternalToken.ClientId, currExternalToken.ClientSecret, currExternalToken.Scope, currExternalToken.IdentityPoolId, currExternalToken.HTTPClient)
	if err != nil {
		return err
	}
	c.oauthToken = externalToken

	stsToken, err := requestNewSTSOAuthToken(ctx, externalToken.AccessToken, currSTSToken.IdentityPoolId, currSTSToken.ExpiresInSeconds, currSTSToken.STSClient)
	if err != nil {
		return err
	}
	c.stsToken = stsToken
	return nil
}

func (c *TableflowRestClient) fetchOrOverrideSTSOAuthTokenFromApiContext(ctx context.Context) error {
	currExternalToken := c.oauthToken
	currSTSToken := c.stsToken

	// Check if the current STS OAuth token is still valid
	if valid := validateCurrentSTSOAuthToken(ctx, currSTSToken); valid {
		return nil
	}

	// Check if the current external OAuth token is still valid
	// If valid, request a new STS token based on the current external OAuth token
	if valid := validateCurrentExternalOAuthToken(ctx, currExternalToken); valid {
		stsToken, err := requestNewSTSOAuthToken(ctx, currExternalToken.AccessToken, currSTSToken.IdentityPoolId, currSTSToken.ExpiresInSeconds, currSTSToken.STSClient)
		if err != nil {
			return err
		}
		c.stsToken = stsToken
		return nil
	}

	// If invalid, request a new external OAuth token first
	// then request a new STS token based on the current external OAuth token
	externalToken, err := requestNewExternalOAuthToken(ctx, currExternalToken.TokenUrl, currExternalToken.ClientId, currExternalToken.ClientSecret, currExternalToken.Scope, currExternalToken.IdentityPoolId, currExternalToken.HTTPClient)
	if err != nil {
		return err
	}
	c.oauthToken = externalToken

	stsToken, err := requestNewSTSOAuthToken(ctx, externalToken.AccessToken, currSTSToken.IdentityPoolId, currSTSToken.ExpiresInSeconds, currSTSToken.STSClient)
	if err != nil {
		return err
	}
	c.stsToken = stsToken
	return nil
}

// Extracts "foo" from "https://api.confluent.cloud/iam/v2/service-accounts?page_token=foo"
func extractPageToken(nextPageUrlString string) (string, error) {
	nextPageUrl, err := url.Parse(nextPageUrlString)
	if err != nil {
		return "", fmt.Errorf("could not parse %q into URL, %s", nextPageUrlString, createDescriptiveError(err))
	}
	pageToken := nextPageUrl.Query().Get(pageTokenQueryParameter)
	if pageToken == "" {
		return "", fmt.Errorf("could not parse the value for %q query parameter from %q", pageTokenQueryParameter, nextPageUrlString)
	}
	return pageToken, nil
}

func verifyListValues(values, acceptedValues []string, ignoreCase bool) error {
	for _, actualValue := range values {
		found := stringInSlice(actualValue, acceptedValues, ignoreCase)
		if !found {
			return fmt.Errorf("expected %s to be one of %v, got %s", actualValue, acceptedValues, actualValue)
		}
	}
	return nil
}

func stringInSlice(target string, slice []string, ignoreCase bool) bool {
	for _, v := range slice {
		if v == target || (ignoreCase && strings.EqualFold(v, target)) {
			return true
		}
	}
	return false
}

func convertToStringSlice(items []interface{}) []string {
	stringItems := make([]string, len(items))
	for i, item := range items {
		stringItems[i] = fmt.Sprint(item)
	}
	return stringItems
}

func clusterSettingsKeysValidate(v interface{}, path cty.Path) diag.Diagnostics {
	clusterSettingsMap := v.(map[string]interface{})

	if len(clusterSettingsMap) == 0 {
		return diag.Errorf("error creating / updating Cluster Config: %q block should not be empty", paramConfigs)
	}

	for clusterSetting := range clusterSettingsMap {
		if !stringInSlice(clusterSetting, editableClusterSettings, false) {
			return diag.Errorf("error creating / updating Cluster Config: %q cluster setting is read-only and cannot be updated. "+
				"Read %s for more details.", clusterSetting, docsClusterConfigUrl)
		}
	}
	return nil
}

func clusterLinkSettingsKeysValidate(v interface{}, path cty.Path) diag.Diagnostics {
	clusterSettingsMap := v.(map[string]interface{})

	for clusterSetting := range clusterSettingsMap {
		if !stringInSlice(clusterSetting, editableClusterLinkSettings, false) {
			return diag.Errorf("error creating / updating Cluster Link: %q cluster link setting is read-only and cannot be updated. "+
				"Read %s for more details.", clusterSetting, docsClusterLinkConfigUrl)
		}
	}
	return nil
}

// https://github.com/confluentinc/cli/blob/main/internal/connect/utils.go#L88C1-L125C2
func uploadFile(url, filePath string, formFields map[string]any, fileExtension, cloud string) error {
	// TODO: We have a task to export the method for general use in a more maintainable way (APIT-2912)
	// TODO: figure out a way to mock this function and delete this hack
	if url == tfCustomConnectorPluginTestUrl {
		return nil
	}
	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)

	for key, value := range formFields {
		if strValue, ok := value.(string); ok {
			_ = writer.WriteField(key, strValue)
		}
	}

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, file); err != nil {
		return err
	}

	if err := writer.Close(); err != nil {
		return err
	}

	client := &http.Client{
		Timeout: 20 * time.Minute,
	}

	var contentFormat string
	switch strings.ToLower(fileExtension) {
	case "zip":
		contentFormat = "application/zip"
	case "jar":
		contentFormat = "application/java-archive"
	}

	if cloud == "GCP" {
		_, err = sling.New().
			Client(client).
			Base(url).
			Set("Content-Type", contentFormat).
			Put("").
			Body(&buffer).
			ReceiveSuccess(nil)
	} else if cloud == "AZURE" {
		_, err = sling.New().
			Client(client).
			Base(url).
			Set("x-ms-blob-type", "BlockBlob").
			Set("Content-Type", contentFormat).
			Put("").
			Body(&buffer).
			ReceiveSuccess(nil)
	} else {
		_, err = sling.New().
			Client(client).
			Base(url).
			Set("Content-Type", writer.FormDataContentType()).
			Post("").
			Body(&buffer).
			ReceiveSuccess(nil)
	}
	if err != nil {
		return err
	}

	return nil
}

func extractCloudAndRegionName(resourceId string) (string, string, error) {
	parts := strings.Split(resourceId, ".")
	cloud := ""
	regionName := ""
	if len(parts) == 3 {
		// old version of API Key Mgmt API
		cloud = parts[1]
		regionName = parts[2]
	} else if len(parts) == 2 {
		// new version of API Key Mgmt API
		cloud = parts[0]
		regionName = parts[1]
	} else {
		return "", "", fmt.Errorf("error extracting cloud and region name: invalid format: expected " +
			"'<cloud>.<region name>' or '<environment>.<cloud>.<region name>'")
	}

	return cloud, regionName, nil
}

func extractOrgIdFromResourceName(resourceName string) (string, error) {
	// Match any string of non-slash characters after organization= until the next slash or the end of the string.
	re := regexp.MustCompile(`/organization=([^/]+)(/|$)`)
	match := re.FindStringSubmatch(resourceName)
	if len(match) > 1 {
		return match[1], nil
	} else {
		return "", fmt.Errorf("could not find organization ID in %v: %s", paramResourceName, resourceName)
	}
}

func generateFlinkStatementName() string {
	clientName := "tf"
	date := time.Now().Format("2006-01-02")
	localTime := time.Now().Format("150405")
	id := uuid.New().String()
	return fmt.Sprintf("%s-%s-%s-%s", clientName, date, localTime, id)
}

func parseStatementName(id string) (string, error) {
	parts := strings.Split(id, "/")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid ID format: expected '<Environment ID>/Compute Pool ID>/<Statement name>'")
	}
	return parts[2], nil
}

func canUpdateSchemaEntityType(oldEntityName, newEntityName string) bool {
	oldParts := strings.Split(oldEntityName, ":")
	newParts := strings.Split(newEntityName, ":")
	if 3 != len(oldParts) || 3 != len(newParts) {
		return false
	}
	if oldParts[0] != newParts[0] || oldParts[1] != newParts[1] {
		return false
	}
	return isNewSchemaIdGreaterThanOld(oldParts, newParts)
}

func canUpdateFieldOrRecordEntityType(oldEntityName, newEntityName string) bool {
	oldParts := strings.Split(oldEntityName, ":")
	newParts := strings.Split(newEntityName, ":")
	if 4 != len(oldParts) || 4 != len(newParts) {
		return false
	}
	if oldParts[0] != newParts[0] || oldParts[1] != newParts[1] || oldParts[3] != newParts[3] {
		return false
	}
	return isNewSchemaIdGreaterThanOld(oldParts, newParts)
}

func isNewSchemaIdGreaterThanOld(oldParts, newParts []string) bool {
	oldSchemaId, err := strconv.Atoi(oldParts[2])
	if err != nil {
		return false
	}
	newSchemaId, err := strconv.Atoi(newParts[2])
	if err != nil {
		return false
	}
	// Tags are propagated for new versions
	return newSchemaId > oldSchemaId
}

func canUpdateEntityName(entityType, oldEntityName, newEntityName string) bool {
	switch entityType {
	case schemaEntityType:
		return canUpdateSchemaEntityType(oldEntityName, newEntityName)
	case fieldEntityType, recordEntityType:
		return canUpdateFieldOrRecordEntityType(oldEntityName, newEntityName)
	default:
		return false
	}
}

func canUpdateEntityNameBusinessMetadata(entityType, oldEntityName, newEntityName string) bool {
	if oldEntityName == newEntityName {
		return true
	}
	switch entityType {
	case schemaEntityType:
		return canUpdateSchemaEntityType(oldEntityName, newEntityName)
	default:
		return false
	}
}

func convertConfigDataToAlterConfigBatchRequestData(configs []kafkarestv3.ConfigData) []kafkarestv3.AlterConfigBatchRequestDataData {
	configResult := make([]kafkarestv3.AlterConfigBatchRequestDataData, len(configs))

	for i, config := range configs {
		configResult[i] = kafkarestv3.AlterConfigBatchRequestDataData{
			Name:      config.Name,
			Value:     config.Value,
			Operation: *kafkarestv3.NewNullableString(kafkarestv3.PtrString("SET")),
		}
	}

	return configResult
}

func extractCredentialConfigs(configs []kafkarestv3.ConfigData) []kafkarestv3.AlterConfigBatchRequestDataData {
	credentialConfigKeys := []string{
		saslJaasConfigConfigKey,
		localSaslJaasConfigConfigKey,
		saslMechanismConfigKey,
		localSaslMechanismConfigKey,
	}

	var filteredConfigs []kafkarestv3.ConfigData
	for _, config := range configs {
		if stringInSlice(config.Name, credentialConfigKeys, false) {
			filteredConfigs = append(filteredConfigs, config)
		}
	}

	return convertConfigDataToAlterConfigBatchRequestData(filteredConfigs)
}

func validateAllOrNoneAttributesSetForResources(
	kafkaApiKey, kafkaApiSecret, kafkaID, kafkaRestEndpoint,
	schemaRegistryApiKey, schemaRegistryApiSecret, schemaRegistryClusterId, schemaRegistryRestEndpoint, catalogRestEndpoint,
	flinkApiKey, flinkApiSecret, flinkOrganizationId, flinkEnvironmentId, flinkComputePoolId, flinkRestEndpoint, flinkPrincipalId,
	tableflowApiKey, tableflowApiSecret string) (ResourceMetadataSetFlags, diag.Diagnostics) {
	var flags ResourceMetadataSetFlags
	// 3 or 4 attributes should be set or not set at the same time
	// Option #2: (kafka_api_key, kafka_api_secret, kafka_rest_endpoint)
	// Option #3 (primary): (kafka_api_key, kafka_api_secret, kafka_rest_endpoint, kafka_id)
	allKafkaAttributesAreSet := (kafkaApiKey != "") && (kafkaApiSecret != "") && (kafkaRestEndpoint != "")
	allKafkaAttributesAreNotSet := (kafkaApiKey == "") && (kafkaApiSecret == "") && (kafkaRestEndpoint == "")
	justOneOrTwoKafkaAttributesAreSet := !(allKafkaAttributesAreSet || allKafkaAttributesAreNotSet)
	if justOneOrTwoKafkaAttributesAreSet {
		return flags, diag.Errorf("(kafka_api_key, kafka_api_secret, kafka_rest_endpoint) or (kafka_api_key, kafka_api_secret, kafka_rest_endpoint, kafka_id) attributes should be set or not set in the provider block at the same time")
	}
	flags.isKafkaMetadataSet = allKafkaAttributesAreSet

	// All 4 attributes should be set or not set at the same time
	endpointIsSet := schemaRegistryRestEndpoint != "" || catalogRestEndpoint != ""
	allSchemaRegistryAttributesAreSet := (schemaRegistryApiKey != "") && (schemaRegistryApiSecret != "") && endpointIsSet && (schemaRegistryClusterId != "")
	allSchemaRegistryAttributesAreNotSet := (schemaRegistryApiKey == "") && (schemaRegistryApiSecret == "") && !endpointIsSet && (schemaRegistryClusterId == "")
	justSubsetOfSchemaRegistryAttributesAreSet := !(allSchemaRegistryAttributesAreSet || allSchemaRegistryAttributesAreNotSet)
	if justSubsetOfSchemaRegistryAttributesAreSet {
		return flags, diag.Errorf("All 4 schema_registry_api_key, schema_registry_api_secret, schema_registry_rest_endpoint, schema_registry_id attributes should be set or not set in the provider block at the same time")
	}
	flags.isSchemaRegistryMetadataSet = allSchemaRegistryAttributesAreSet
	flags.isCatalogMetadataSet = allSchemaRegistryAttributesAreSet

	// All 7 attributes should be set or not set at the same time
	allFlinkAttributesAreSet := (flinkApiKey != "") && (flinkApiSecret != "") && (flinkRestEndpoint != "") && (flinkOrganizationId != "") && (flinkEnvironmentId != "") && (flinkComputePoolId != "") && (flinkPrincipalId != "")
	allFlinkAttributesAreNotSet := (flinkApiKey == "") && (flinkApiSecret == "") && (flinkRestEndpoint == "") && (flinkOrganizationId == "") && (flinkEnvironmentId == "") && (flinkComputePoolId == "") && (flinkPrincipalId == "")
	justSubsetOfFlinkAttributesAreSet := !(allFlinkAttributesAreSet || allFlinkAttributesAreNotSet)
	if justSubsetOfFlinkAttributesAreSet {
		return flags, diag.Errorf("All 7 flink_api_key, flink_api_secret, flink_rest_endpoint, organization_id, environment_id, flink_compute_pool_id, flink_principal_id attributes should be set or not set in the provider block at the same time")
	}
	flags.isFlinkMetadataSet = allFlinkAttributesAreSet

	allTableflowAttributesAreSet := (tableflowApiKey != "") && (tableflowApiSecret != "")
	allTableflowAttributesAreNotSet := (tableflowApiKey == "") && (tableflowApiSecret == "")
	justOneTableflowAttributeSet := !(allTableflowAttributesAreSet || allTableflowAttributesAreNotSet)
	if justOneTableflowAttributeSet {
		return flags, diag.Errorf("Both tableflow_api_key and tableflow_api_secret should be set or not set in the provider block at the same time")
	}
	flags.isTableflowMetadataSet = allTableflowAttributesAreSet

	return flags, nil
}

func validateAllOrNoneAttributesSetForResourcesWithOAuth(
	kafkaID, kafkaRestEndpoint,
	srID, srRestEndpoint, catalogRestEndpoint,
	flinkOrganizationId, flinkEnvironmentId, flinkComputePoolId, flinkRestEndpoint, flinkPrincipalId string) (ResourceMetadataSetFlags, diag.Diagnostics) {
	var flags ResourceMetadataSetFlags
	// When OAuth is enabled, the Kafka ID and rest endpoint should be set or not set at the same time
	allKafkaAttributesAreSet := (kafkaID != "") && (kafkaRestEndpoint != "")
	allKafkaAttributesAreNotSet := (kafkaID == "") && (kafkaRestEndpoint == "")
	justOneOfKafkaAttributesAreSet := !(allKafkaAttributesAreSet || allKafkaAttributesAreNotSet)
	if justOneOfKafkaAttributesAreSet {
		return flags, diag.Errorf("(kafka_rest_endpoint, kafka_id) attributes should both be set or not set in the provider block at the same time with OAuth enabled.")
	}
	flags.isKafkaMetadataSet = allKafkaAttributesAreSet

	// When OAuth is enabled, the Schema Registry ID and rest endpoint (Schema Registry or Catalog) should be set or not set at the same time
	endpointIsSet := srRestEndpoint != "" || catalogRestEndpoint != ""
	allSchemaRegistryAttributesAreSet := endpointIsSet && (srID != "")
	allSchemaRegistryAttributesAreNotSet := !endpointIsSet && (srID == "")
	justOneOfSchemaRegistryAttributesAreSet := !(allSchemaRegistryAttributesAreSet || allSchemaRegistryAttributesAreNotSet)
	if justOneOfSchemaRegistryAttributesAreSet {
		return flags, diag.Errorf("(either schema_registry_rest_endpoint or catalog_rest_endpoint) and schema_registry_id attributes should both be set or not set in the provider block at the same time with OAuth enabled")
	}
	flags.isSchemaRegistryMetadataSet = allSchemaRegistryAttributesAreSet
	flags.isCatalogMetadataSet = allSchemaRegistryAttributesAreSet

	// When OAuth is enabled, all Flink related attributes below should be set or not set at the same time
	allFlinkAttributesAreSet := (flinkRestEndpoint != "") && (flinkOrganizationId != "") && (flinkEnvironmentId != "") && (flinkComputePoolId != "") && (flinkPrincipalId != "")
	allFlinkAttributesAreNotSet := (flinkRestEndpoint == "") && (flinkOrganizationId == "") && (flinkEnvironmentId == "") && (flinkComputePoolId == "") && (flinkPrincipalId == "")
	justSubsetOfFlinkAttributesAreSet := !(allFlinkAttributesAreSet || allFlinkAttributesAreNotSet)
	if justSubsetOfFlinkAttributesAreSet {
		return flags, diag.Errorf("All 5 (flink_rest_endpoint, organization_id, environment_id, flink_compute_pool_id, flink_principal_id) attributes should be set or not set in the provider block at the same time with OAuth enabled")
	}
	flags.isFlinkMetadataSet = allFlinkAttributesAreSet

	// Tableflow doesn't support OAuth authentication as of this implementation
	// So the `flags.areTableflowAllSet` is always false
	return flags, nil
}
