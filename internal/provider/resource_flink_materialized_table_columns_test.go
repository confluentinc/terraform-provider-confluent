package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// TestExpandAllColumnsKind verifies the kind sent for each column block: an omitted or empty kind is
// sent as the block's kind, and an explicitly configured kind is sent unchanged.
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
		{name: "empty physical kind defaults to Physical", block: paramColumnPhysical, column: map[string]interface{}{paramPhysicalName: "event_ts", paramPhysicalKind: ""}, wantKind: "Physical"},
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

// materializedTableState returns the state of an existing table with the given attributes, plus the
// attributes Read always sets, so planning against it updates the table in place instead of replacing it.
func materializedTableState(attributes map[string]string) *terraform.InstanceState {
	state := map[string]string{
		"id":                "env-abc123/lkc-01/t",
		"distribution.#":    "0",
		"session_options.%": "0",
		"table_options.%":   "0",
		"stopped":           "false",
	}
	for k, v := range attributes {
		state[k] = v
	}
	return &terraform.InstanceState{ID: state["id"], Attributes: state}
}

// planColumns plans the given columns against state the way Terraform does and returns the changed
// columns attributes. A nil state plans a create.
func planColumns(t *testing.T, state *terraform.InstanceState, columns []interface{}) map[string]*terraform.ResourceAttrDiff {
	t.Helper()
	config := terraform.NewResourceConfigRaw(map[string]interface{}{paramColumns: columns})
	diff, err := flinkMaterializedTableResource().SimpleDiff(context.Background(), state, config, nil)
	if err != nil {
		t.Fatal(err)
	}
	changed := map[string]*terraform.ResourceAttrDiff{}
	if diff == nil {
		return changed
	}
	if state != nil && diff.RequiresNew() {
		t.Fatalf("expected an in-place update, got a replacement: %v", diff.Attributes)
	}
	for k, v := range diff.Attributes {
		if strings.HasPrefix(k, paramColumns+".") && (v.Old != v.New || v.NewComputed || v.NewRemoved) {
			changed[k] = v
		}
	}
	return changed
}

// TestEmptyPhysicalBlockPlansNoKindChange verifies that a resource whose state holds an empty physical
// column block (no kind) plans no change for it.
func TestEmptyPhysicalBlockPlansNoKindChange(t *testing.T) {
	state := materializedTableState(map[string]string{
		"columns.#":                    "1",
		"columns.0.columns_physical.#": "1",
		"columns.0.columns_computed.#": "0",
		"columns.0.columns_metadata.#": "0",
	})
	changed := planColumns(t, state, []interface{}{map[string]interface{}{paramColumnPhysical: []interface{}{map[string]interface{}{}}}})
	for k, v := range changed {
		t.Errorf("unexpected columns diff %s: %q -> %q", k, v.Old, v.New)
	}
}

// TestColumnKindPlan verifies when a column kind shows in the plan: an omitted kind never does, and a
// configured kind does unless the state already holds it.
func TestColumnKindPlan(t *testing.T) {
	blocks := []struct {
		block, name, kind, blockKind string
	}{
		{paramColumnPhysical, paramPhysicalName, paramPhysicalKind, physicalColumnKind},
		{paramColumnComputed, paramComputedName, paramComputedKind, computedColumnKind},
		{paramColumnMetadata, paramMetadataName, paramMetadataKind, metadataColumnKind},
	}
	const omitted = "<omitted>"
	tests := []struct {
		name       string
		stateKind  string // ignored for a create
		configKind string
		create     bool
		wantPlan   bool
	}{
		{name: "create with the kind plans it", create: true, configKind: "<block kind>", wantPlan: true},
		{name: "create without a kind plans none", create: true, configKind: omitted, wantPlan: false},
		{name: "omitting the stored kind plans no change", stateKind: "<block kind>", configKind: omitted, wantPlan: false},
		{name: "setting the kind over an empty stored kind plans it", stateKind: "", configKind: "<block kind>", wantPlan: true},
		{name: "changing the stored kind plans it", stateKind: "<block kind>", configKind: "other", wantPlan: true},
		{name: "omitting a stored kind that is not the block's plans the change", stateKind: "other", configKind: omitted, wantPlan: true},
	}
	for _, b := range blocks {
		for _, tt := range tests {
			t.Run(fmt.Sprintf("%s/%s", b.block, tt.name), func(t *testing.T) {
				value := func(v string) string {
					if v == "<block kind>" {
						return b.blockKind
					}
					return v
				}
				column := map[string]interface{}{b.name: "c"}
				if tt.configKind != omitted {
					column[b.kind] = value(tt.configKind)
				}
				var state *terraform.InstanceState
				if !tt.create {
					attributes := map[string]string{
						"columns.#":                    "1",
						"columns.0.columns_physical.#": "0",
						"columns.0.columns_computed.#": "0",
						"columns.0.columns_metadata.#": "0",
					}
					attributes[fmt.Sprintf("columns.0.%s.#", b.block)] = "1"
					attributes[fmt.Sprintf("columns.0.%s.0.%s", b.block, b.name)] = "c"
					attributes[fmt.Sprintf("columns.0.%s.0.%s", b.block, b.kind)] = value(tt.stateKind)
					state = materializedTableState(attributes)
				}
				changed := planColumns(t, state, []interface{}{map[string]interface{}{b.block: []interface{}{column}}})
				kindDiff, planned := changed[fmt.Sprintf("columns.0.%s.0.%s", b.block, b.kind)]
				if planned != tt.wantPlan {
					t.Fatalf("expected kind planned = %t, got %t (%+v)", tt.wantPlan, planned, kindDiff)
				}
			})
		}
	}
}

// TestColumnKindPlanIgnoresUnknownValues verifies that a physical column's name, type, or comment being
// unknown at plan time does not change the planned kind, so the plan stays the same once the value
// resolves to empty.
func TestColumnKindPlanIgnoresUnknownValues(t *testing.T) {
	kindKey := fmt.Sprintf("columns.0.%s.0.%s", paramColumnPhysical, paramPhysicalKind)
	for _, attribute := range []string{paramPhysicalName, paramPhysicalType, paramPhysicalComment} {
		t.Run(attribute, func(t *testing.T) {
			plan := func(value string) *terraform.ResourceAttrDiff {
				column := map[string]interface{}{attribute: value}
				return planColumns(t, nil, []interface{}{map[string]interface{}{paramColumnPhysical: []interface{}{column}}})[kindKey]
			}
			if unknown, resolved := plan(unknownConfigValue), plan(""); unknown != nil || resolved != nil {
				t.Errorf("expected no planned kind, got %+v while unknown and %+v once resolved", unknown, resolved)
			}
		})
	}
}

// TestSuppressOmittedColumnKindDiff verifies that suppressOmittedColumnKindDiff hides only a change from
// the given kind to an omitted kind, not a newly configured kind or a change to or from any other kind.
func TestSuppressOmittedColumnKindDiff(t *testing.T) {
	suppress := suppressOmittedColumnKindDiff(physicalColumnKind)
	tests := []struct {
		old, new string
		want     bool
	}{
		{old: "Physical", new: "", want: true},
		{old: "", new: "Physical", want: false},
		{old: "sql.v1.PhysicalColumn", new: "", want: false},
		{old: "", new: "sql.v1.PhysicalColumn", want: false},
		{old: "Physical", new: "sql.v1.PhysicalColumn", want: false},
		{old: "Computed", new: "", want: false},
	}
	for _, tt := range tests {
		if got := suppress("columns.0.columns_physical.0.column_physical_kind", tt.old, tt.new, nil); got != tt.want {
			t.Errorf("suppress(%q -> %q) = %t, want %t", tt.old, tt.new, got, tt.want)
		}
	}
}
