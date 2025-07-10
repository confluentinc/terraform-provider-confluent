# Live Testing Template

## Adding New Live Tests

When adding new live tests, follow this template structure:

### 1. File Structure
```go
//go:build live_test && (all || YOUR_GROUP_HERE)

// Copyright 2021 Confluent Inc. All Rights Reserved.
// ... (standard license header)

package provider

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccYourResourceLive(t *testing.T) {
	// Skip this test unless explicitly enabled
	if os.Getenv("TF_ACC_PROD") == "" {
		t.Skip("Skipping live test. Set TF_ACC_PROD=1 to run this test.")
	}

	// Read credentials from environment variables
	apiKey := os.Getenv("CONFLUENT_CLOUD_API_KEY")
	apiSecret := os.Getenv("CONFLUENT_CLOUD_API_SECRET")
	endpoint := os.Getenv("CONFLUENT_CLOUD_ENDPOINT")

	// Validate required environment variables
	if apiKey == "" || apiSecret == "" {
		t.Fatal("CONFLUENT_CLOUD_API_KEY and CONFLUENT_CLOUD_API_SECRET must be set for live tests")
	}

	// Use timestamped names to avoid conflicts
	resourceName := fmt.Sprintf("tf-provider-live-test-%d", time.Now().Unix())

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckYourResourceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckYourResourceLiveConfig(endpoint, resourceName, apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckYourResourceExists("confluent_your_resource.test"),
					resource.TestCheckResourceAttr("confluent_your_resource.test", "display_name", resourceName),
					// Add more checks as needed
				),
			},
			{
				ResourceName:      "confluent_your_resource.test",
				ImportState:       true,
				ImportStateVerify: true,
				// Add ImportStateIdFunc if needed for complex import formats
			},
		},
	})
}

func testAccCheckYourResourceLiveConfig(endpoint, resourceName, apiKey, apiSecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint         = "%s"
		cloud_api_key    = "%s"
		cloud_api_secret = "%s"
	}

	resource "confluent_your_resource" "test" {
		display_name = "%s"
		# Add other required fields
	}
	`, endpoint, apiKey, apiSecret, resourceName)
}
```

### 2. Resource Group Tags

Use these build tags based on your resource type:

| Resource Group | Build Tag | Resources |
|---|---|---|
| **Core** | `core` | environments, service_accounts, api_keys |
| **Kafka** | `kafka` | kafka_clusters, kafka_topics, kafka_acls, kafka_client_quotas |
| **Connect** | `connect` | connectors, connect_artifacts, custom_connector_plugins |
| **Schema Registry** | `schema_registry` | schemas, schema_exporters, subject_configs |
| **Networking** | `networking` | networks, private_links, dns_forwarders, gateways |
| **Flink** | `flink` | flink_artifacts, flink_compute_pools, flink_connections, flink_statements |
| **RBAC** | `rbac` | role_bindings, identity_pools, identity_providers, group_mappings |
| **Data Catalog** | `data_catalog` | tags, tag_bindings, business_metadata, catalog_integrations |
| **Tableflow** | `tableflow` | tableflow_topics and related resources |

### 3. Multi-Group Resources

If your resource spans multiple groups, use multiple tags:
```go
//go:build live_test && (all || core || kafka)
```

### 4. Testing Your New Test

```bash
# Test specific group
make live-test-YOUR_GROUP

# Test multiple groups
make live-test GROUPS="core,kafka"

# Test all
make live-test
```

### 5. Best Practices

1. **Always use timestamped names** to avoid resource conflicts
2. **Test the full lifecycle**: Create → Read → Update → Import → Destroy
3. **Use real API endpoints** and credentials from environment variables
4. **Add proper cleanup** in destroy functions
5. **Include meaningful assertions** in your checks
6. **Handle import state formatting** correctly for complex resources

### 6. Environment Variables Required

All live tests expect these environment variables:
- `TF_ACC_PROD=1` (to enable live tests)
- `CONFLUENT_CLOUD_API_KEY` (from Vault)
- `CONFLUENT_CLOUD_API_SECRET` (from Vault)
- `CONFLUENT_CLOUD_ENDPOINT` (usually `https://api.confluent.cloud`)

### 7. Semaphore Integration

Once your test is tagged properly, it will automatically be included in the appropriate Semaphore pipeline. The pipeline uses the `TF_LIVE_TEST_GROUPS` parameter to control which groups to run. 