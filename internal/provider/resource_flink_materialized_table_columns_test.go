package provider

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
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
		// A physical block that sets only a type or only a comment still gets the default kind.
		{name: "physical with only a type defaults to Physical", block: paramColumnPhysical, column: map[string]interface{}{paramPhysicalType: "STRING"}, wantKind: "Physical"},
		{name: "physical with only a comment defaults to Physical", block: paramColumnPhysical, column: map[string]interface{}{paramPhysicalComment: "note"}, wantKind: "Physical"},
		// Explicit kinds are sent unchanged; non-default values tell a kept kind apart from the default.
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

// TestExpandAllColumnsSkipsEmptyPhysicalBlock verifies that an empty columns_physical {} block is not
// sent, while a named physical column next to it still gets the default kind.
func TestExpandAllColumnsSkipsEmptyPhysicalBlock(t *testing.T) {
	d := schema.TestResourceDataRaw(t, flinkMaterializedTableResource().Schema, map[string]interface{}{
		paramColumns: []interface{}{
			map[string]interface{}{paramColumnPhysical: []interface{}{map[string]interface{}{paramPhysicalName: "a", paramPhysicalType: "STRING"}}},
			map[string]interface{}{paramColumnPhysical: []interface{}{map[string]interface{}{}}},
		},
	})
	columns := expandAllColumns(d)
	if len(columns) != 1 || columns[0].SqlV1PhysicalColumn == nil ||
		columns[0].SqlV1PhysicalColumn.Name != "a" || columns[0].SqlV1PhysicalColumn.Kind != "Physical" {
		t.Fatalf("expected only column \"a\" of kind Physical, got %d column(s): %+v", len(columns), columns)
	}
}

// TestEmptyPhysicalBlockPlansNoKindChange verifies that a resource whose state holds an empty physical
// column block (no kind) plans no change for it.
func TestEmptyPhysicalBlockPlansNoKindChange(t *testing.T) {
	state := &terraform.InstanceState{ID: "env-abc123/lkc-01/t", Attributes: map[string]string{
		"id":                           "env-abc123/lkc-01/t",
		"columns.#":                    "1",
		"columns.0.columns_physical.#": "1",
		"columns.0.columns_computed.#": "0",
		"columns.0.columns_metadata.#": "0",
	}}
	config := terraform.NewResourceConfigRaw(map[string]interface{}{
		paramColumns: []interface{}{map[string]interface{}{paramColumnPhysical: []interface{}{map[string]interface{}{}}}},
	})
	diff, err := flinkMaterializedTableResource().Diff(context.Background(), state, config, nil)
	if err != nil {
		t.Fatal(err)
	}
	if diff == nil {
		return
	}
	for k, v := range diff.Attributes {
		if strings.HasPrefix(k, paramColumns+".") && (v.Old != v.New || v.NewComputed || v.NewRemoved) {
			t.Errorf("unexpected columns diff %s: %q -> %q", k, v.Old, v.New)
		}
	}
}
