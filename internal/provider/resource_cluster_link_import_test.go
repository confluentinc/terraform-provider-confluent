package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestClusterLinkImportMissingRestEndpointEnvVar(t *testing.T) {
	tests := []struct {
		name   string
		id     string
		envVar string
	}{
		{"destination", "link/DESTINATION/OUTBOUND/lkc-src/lkc-dst", importDestinationKafkaRestEndpointEnvVar},
		{"source", "link/SOURCE/OUTBOUND/lkc-src/lkc-dst", importSourceKafkaRestEndpointEnvVar},
		{"bidirectional", "link/BIDIRECTIONAL/OUTBOUND/lkc-local/lkc-remote", importLocalKafkaRestEndpointEnvVar},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for _, v := range []string{importSourceKafkaRestEndpointEnvVar, importDestinationKafkaRestEndpointEnvVar, importLocalKafkaRestEndpointEnvVar, importRemoteKafkaRestEndpointEnvVar} {
				t.Setenv(v, "")
			}
			d := schema.TestResourceDataRaw(t, clusterLinkResource().Schema, map[string]interface{}{})
			d.SetId(tc.id)
			// meta is nil on purpose: the check must fail before any client is created.
			_, err := clusterLinkImport(context.Background(), d, nil)
			if err == nil || !strings.Contains(err.Error(), tc.envVar) {
				t.Fatalf("expected error mentioning %s, got: %v", tc.envVar, err)
			}
		})
	}
}
