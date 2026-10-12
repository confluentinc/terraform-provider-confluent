package provider

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// TestExpandAllColumnsKind verifies the kind sent for each column block: an omitted kind defaults to
// the block's kind, and an explicitly configured kind is sent unchanged.
func TestExpandAllColumnsKind(t *testing.T) {
	tests := []struct {
		name     string
		block    string
		column   map[string]interface{}
		wantKind string
	}{
		{name: "physical defaults to Physical", block: paramColumnPhysical, column: map[string]interface{}{paramPhysicalName: "event_ts"}, wantKind: "Physical"},
		{name: "computed defaults to Computed", block: paramColumnComputed, column: map[string]interface{}{paramComputedName: "event_hour"}, wantKind: "Computed"},
		{name: "metadata defaults to Metadata", block: paramColumnMetadata, column: map[string]interface{}{paramMetadataName: "kafka_ts"}, wantKind: "Metadata"},
		// Explicit kinds are sent unchanged (aliases used here; the API itself only accepts the canonical kinds).
		{name: "explicit physical kind is kept", block: paramColumnPhysical, column: map[string]interface{}{paramPhysicalName: "event_ts", paramPhysicalKind: "sql.v1.PhysicalColumn"}, wantKind: "sql.v1.PhysicalColumn"},
		{name: "explicit computed kind is kept", block: paramColumnComputed, column: map[string]interface{}{paramComputedName: "event_hour", paramComputedKind: "sql.v1.ComputedColumn"}, wantKind: "sql.v1.ComputedColumn"},
		{name: "explicit metadata kind is kept", block: paramColumnMetadata, column: map[string]interface{}{paramMetadataName: "kafka_ts", paramMetadataKind: "sql.v1.MetadataColumn"}, wantKind: "sql.v1.MetadataColumn"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := schema.TestResourceDataRaw(t, flinkMaterializedTableResource().Schema, map[string]interface{}{
				paramColumns: []interface{}{map[string]interface{}{tt.block: []interface{}{tt.column}}},
			})
			columns := expandAllColumns(d)
			if len(columns) != 1 {
				t.Fatalf("expected 1 column, got %d", len(columns))
			}
			body, err := json.Marshal(columns[0])
			if err != nil {
				t.Fatal(err)
			}
			var sent map[string]interface{}
			if err := json.Unmarshal(body, &sent); err != nil {
				t.Fatal(err)
			}
			if sent["kind"] != tt.wantKind {
				t.Errorf("expected kind %q, got %q in %s", tt.wantKind, sent["kind"], body)
			}
		})
	}
}
