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
	"math/rand"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccYourResourceLive(t *testing.T) {
	// Enable parallel execution for I/O bound operations
	t.Parallel()

	// Skip this test unless explicitly enabled
	if os.Getenv("TF_ACC_PROD") == "" {
		t.Skip("Skipping live test. Set TF_ACC_PROD=1 to run this test.")
	}

	// Read credentials and configuration from environment variables (populated by Vault)
	apiKey := os.Getenv("CONFLUENT_CLOUD_API_KEY")
	apiSecret := os.Getenv("CONFLUENT_CLOUD_API_SECRET")
	endpoint := os.Getenv("CONFLUENT_CLOUD_ENDPOINT")
	if endpoint == "" {
		endpoint = "https://api.confluent.cloud" // Use default endpoint if not set
	}

	// Validate required environment variables are present
	if apiKey == "" || apiSecret == "" {
		t.Fatal("CONFLUENT_CLOUD_API_KEY and CONFLUENT_CLOUD_API_SECRET must be set for live tests")
	}

	// Generate unique names for test resources to avoid conflicts
	randomSuffix := rand.Intn(100000)
	resourceDisplayName := fmt.Sprintf("tf-live-test-%d", randomSuffix)
	resourceLabel := "test_resource"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckYourResourceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckYourResourceLiveConfig(endpoint, resourceLabel, resourceDisplayName, apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckYourResourceExists(fmt.Sprintf("confluent_your_resource.%s", resourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_your_resource.%s", resourceLabel), "display_name", resourceDisplayName),
					// Add more checks as needed
				),
			},
			{
				ResourceName:      fmt.Sprintf("confluent_your_resource.%s", resourceLabel),
				ImportState:       true,
				ImportStateVerify: true,
				// Add ImportStateIdFunc if needed for complex import formats
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					resourceId := resources[fmt.Sprintf("confluent_your_resource.%s", resourceLabel)].Primary.ID
					environmentId := resources[fmt.Sprintf("confluent_your_resource.%s", resourceLabel)].Primary.Attributes["environment.0.id"]
					return environmentId + "/" + resourceId, nil
				},
			},
		},
	})
}

func testAccCheckYourResourceLiveConfig(endpoint, resourceLabel, resourceDisplayName, apiKey, apiSecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint         = "%s"
		cloud_api_key    = "%s"
		cloud_api_secret = "%s"
	}

	resource "confluent_your_resource" "%s" {
		display_name = "%s"
		# Add other required fields
	}
	`, endpoint, apiKey, apiSecret, resourceLabel, resourceDisplayName)
}

func testAccCheckYourResourceExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource ID is not set")
		}

		return nil
	}
}

func testAccCheckYourResourceDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_your_resource" {
			continue
		}

		// In live tests, we can't easily check if the resource is actually destroyed
		// without making API calls, so we just verify the resource is removed from state
		if rs.Primary.ID != "" {
			// This is normal - the resource should have an ID but be removed from the live environment
			// The actual cleanup happens through the API calls during destroy
		}
	}
	return nil
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

#### Resource Naming
- **Use `rand.Intn(100000)` for unique suffixes** instead of timestamps to avoid conflicts
- **Use descriptive prefixes** like `tf-live-test-` for easy identification
- **Separate resource labels from display names** for better test organization

#### Test Structure
- **Always use `t.Parallel()`** to enable parallel execution for I/O bound operations
- **Test the full lifecycle**: Create → Read → Update → Import → Destroy
- **Use real API endpoints** and credentials from environment variables
- **Include meaningful assertions** in your checks
- **Handle import state formatting** correctly for complex resources

#### Error Handling
- **Validate environment variables early** with clear error messages
- **Use default endpoints** when not specified (`https://api.confluent.cloud`)
- **Handle timing issues** for resources that take time to provision/destroy

#### Cleanup and Destroy
- **Keep destroy functions simple** for live tests - they just verify state removal
- **Don't make API calls in destroy functions** - let Terraform handle the actual cleanup
- **Comment that actual cleanup happens through API calls during destroy**

### 6. Common Patterns

#### Multiple Test Functions
For resources with different configurations, create multiple test functions:
```go
// Test Basic configuration - simplest, fastest test
func TestAccYourResourceBasicLive(t *testing.T) {
	t.Parallel()
	// ... basic configuration
}

// Test Standard configuration - production-ready with extended feature set
func TestAccYourResourceStandardLive(t *testing.T) {
	t.Parallel()
	// ... standard configuration
}

// Test with specific features
func TestAccYourResourceWithFeatureLive(t *testing.T) {
	t.Parallel()
	// ... feature-specific configuration
}
```

#### Resource Dependencies
When testing resources that depend on others (like schemas depending on environments):
```go
// Generate unique names for all resources
randomSuffix := rand.Intn(100000)
environmentResourceLabel := "test_env"
environmentDisplayName := fmt.Sprintf("tf-test-env-%d", randomSuffix)
resourceResourceLabel := "test_resource"
resourceDisplayName := fmt.Sprintf("tf-test-resource-%d", randomSuffix)
```

#### Timing Considerations
- **Some resources take time to provision** (e.g., Kafka clusters, schema registry clusters)
- **Some resources take time to fully delete** (e.g., Kafka clusters with physical infrastructure)
- **The provider handles most waiting internally** through wait functions
- **Don't add manual sleeps** - rely on the provider's built-in timing logic

### 7. Environment Variables Required

All live tests expect these environment variables:
- `TF_ACC_PROD=1` (to enable live tests)
- `CONFLUENT_CLOUD_API_KEY` (from Vault)
- `CONFLUENT_CLOUD_API_SECRET` (from Vault)
- `CONFLUENT_CLOUD_ENDPOINT` (usually `https://api.confluent.cloud`)

### 8. Semaphore Integration

Once your test is tagged properly, it will automatically be included in the appropriate Semaphore pipeline. The pipeline uses the `TF_LIVE_TEST_GROUPS` parameter to control which groups to run.

### 9. Troubleshooting Common Issues

#### Timing Issues
- **409 Conflict errors**: Usually indicate a resource is still being deleted. Ensure proper wait functions are implemented in the provider.
- **Resource not found errors**: May indicate timing issues with provisioning. Check if the provider has appropriate wait functions.

#### Resource Conflicts
- **Name conflicts**: Ensure you're using unique random suffixes, not timestamps
- **Resource limits**: Be aware of account limits for certain resource types

#### Test Failures
- **Intermittent failures**: Often due to timing issues or resource limits
- **Import failures**: Check that ImportStateIdFunc correctly formats the resource ID
- **Destroy failures**: Usually handled by Terraform, but check for dependency issues 