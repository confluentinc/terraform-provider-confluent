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
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// generatedAllowlistFile is the committed output of cmd/telemetry-attrs-gen.
const generatedAllowlistFile = "telemetry_attrs_allowlist.json"

// acknowledgedMapAttributes is the explicit registry of every top-level
// attribute that is (or contains) a schema.TypeMap. Each carries
// user-controlled keys and values that must never reach telemetry, so adding a
// new one is a deliberate decision: TestMapBearingAttributesAcknowledged fails
// until the new attribute is listed here, forcing review that the telemetry
// path still emits only the attribute's name.
var acknowledgedMapAttributes = []string{
	"confluent_business_metadata.attribute_definition",
	"confluent_business_metadata_binding.attributes",
	"confluent_catalog_entity_attributes.attributes",
	"confluent_cluster_link.config",
	"confluent_connector.config_nonsensitive",
	"confluent_connector.config_sensitive",
	"confluent_connector.offsets",
	"confluent_dns_forwarder.forward_via_gcp_dns_zones",
	"confluent_flink_materialized_table.session_options",
	"confluent_flink_materialized_table.table_options",
	"confluent_flink_statement.latest_offsets",
	"confluent_flink_statement.properties",
	"confluent_flink_statement.properties_sensitive",
	"confluent_kafka_cluster_config.config",
	"confluent_kafka_topic.config",
	"confluent_network.azure",
	"confluent_network.gcp",
	"confluent_network.zonal_subdomains",
	"confluent_schema.metadata",
	"confluent_schema.ruleset",
	"confluent_schema_exporter.config",
	"confluent_schema_registry_kek.properties",
}

// TestResourceAttributeAllowlist_CoversAllManagedResourcesAndNamesOnly asserts
// the generated allowlist covers every managed resource — including ones whose
// constructor does not follow the xResource() naming convention — and contains
// only statically declared, top-level attribute names.
func TestResourceAttributeAllowlist_CoversAllManagedResourcesAndNamesOnly(t *testing.T) {
	p := New(testVersion, "")()
	allowlist := ResourceAttributeAllowlist()

	if len(allowlist) != len(p.ResourcesMap) {
		t.Fatalf("allowlist has %d resources, ResourcesMap has %d", len(allowlist), len(p.ResourcesMap))
	}

	// v1 scope pin — bump when resources are intentionally added/removed.
	const wantManagedResources = 64
	if len(allowlist) != wantManagedResources {
		t.Errorf("allowlist covers %d resources, want %d (update the pin if resources changed)", len(allowlist), wantManagedResources)
	}

	// A resource whose constructor is not named xResource() must still be
	// covered, because the generator walks the runtime map, not source text.
	if _, ok := allowlist["confluent_rtce_topic"]; !ok {
		t.Errorf("confluent_rtce_topic missing — the generator must walk the runtime ResourcesMap, not a naming convention")
	}

	for resourceType, names := range allowlist {
		r, ok := p.ResourcesMap[resourceType]
		if !ok {
			t.Errorf("allowlist has %q which is not in ResourcesMap", resourceType)
			continue
		}
		// Exactly the sorted top-level schema keys — nothing more, nothing less.
		want := make([]string, 0, len(r.Schema))
		for name := range r.Schema {
			want = append(want, name)
		}
		sort.Strings(want)
		if !reflect.DeepEqual(names, want) {
			t.Errorf("resource %q: allowlist names %v != schema keys %v", resourceType, names, want)
		}
		for _, name := range names {
			if _, isDeclared := r.Schema[name]; !isDeclared {
				t.Errorf("resource %q: allowlist name %q is not a declared attribute", resourceType, name)
			}
			// A leaked map key or nested address would contain a '.'; a declared
			// top-level attribute name never does.
			if strings.Contains(name, ".") {
				t.Errorf("resource %q: allowlist name %q looks like a nested/map key — only top-level names are allowed", resourceType, name)
			}
		}
	}
}

// TestResourceAttributeAllowlist_TypeMapContributesNameOnly asserts a
// map-typed attribute contributes only its own name (never its dynamic keys).
func TestResourceAttributeAllowlist_TypeMapContributesNameOnly(t *testing.T) {
	p := New(testVersion, "")()
	allowlist := ResourceAttributeAllowlist()

	// confluent_kafka_topic.config is the canonical user-controlled map.
	topicNames := allowlist["confluent_kafka_topic"]
	if !contains(topicNames, "config") {
		t.Errorf("confluent_kafka_topic allowlist %v should contain the map attribute name \"config\"", topicNames)
	}

	// For every map-bearing attribute, the resource's allowlist must contain its
	// declared name AND nothing derived from the map's contents: no nested address
	// like "config.cleanup.policy". The second check is the load-bearing one — it
	// is the "names only, never keys/values" guarantee at the map boundary, and it
	// would fail if the generator ever descended into a map's contents.
	for _, id := range mapBearingAttributes(p) {
		resourceType, attr, _ := strings.Cut(id, ".")
		names := allowlist[resourceType]
		if !contains(names, attr) {
			t.Errorf("map attribute %q not present by name in allowlist %v", id, names)
		}
		for _, name := range names {
			if strings.HasPrefix(name, attr+".") {
				t.Errorf("resource %q leaked a nested key %q under map attribute %q — map contents must never reach the allowlist", resourceType, name, attr)
			}
		}
	}
}

// TestResourcesAndDataSourcesDisjointSchemas asserts no *schema.Resource pointer
// is shared between ResourcesMap and DataSourcesMap, so the v1 data-source
// exclusion cannot be silently broken by a future resource that reuses a schema
// object for both.
func TestResourcesAndDataSourcesDisjointSchemas(t *testing.T) {
	p := New(testVersion, "")()
	resourcePtrs := make(map[interface{}]string, len(p.ResourcesMap))
	for name, r := range p.ResourcesMap {
		resourcePtrs[r] = name
	}
	for dsName, ds := range p.DataSourcesMap {
		if rName, shared := resourcePtrs[ds]; shared {
			t.Errorf("data source %q and resource %q share the same *schema.Resource pointer; telemetry's data-source exclusion would be broken", dsName, rName)
		}
	}
}

// TestMapBearingAttributesAcknowledged keeps acknowledgedMapAttributes in exact
// sync with the schemas. A new map-typed attribute fails until acknowledged; a
// stale acknowledgment fails until removed. Either way a human reviews that the
// telemetry path still emits only the attribute name.
func TestMapBearingAttributesAcknowledged(t *testing.T) {
	p := New(testVersion, "")()
	got := mapBearingAttributes(p)

	want := append([]string(nil), acknowledgedMapAttributes...)
	sort.Strings(want)

	if !reflect.DeepEqual(got, want) {
		t.Errorf("map-bearing attributes drifted from the acknowledged registry.\n got: %v\nwant: %v\nAdd new entries to acknowledgedMapAttributes after confirming telemetry emits only the attribute name; remove stale ones.", got, want)
	}
}

// TestGeneratedAllowlistFileInSync fails when the committed allowlist JSON no
// longer matches the live schema — the drift guard. Regenerate with
// `make telemetry-attrs`.
func TestGeneratedAllowlistFileInSync(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed: cannot locate the generated file")
	}
	path := filepath.Join(filepath.Dir(thisFile), generatedAllowlistFile)

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v (run `make telemetry-attrs`)", generatedAllowlistFile, err)
	}
	var committed map[string][]string
	if err := json.Unmarshal(raw, &committed); err != nil {
		t.Fatalf("parsing %s: %v", generatedAllowlistFile, err)
	}

	live := ResourceAttributeAllowlist()
	if !reflect.DeepEqual(committed, live) {
		t.Errorf("%s is stale relative to the provider schema; run `make telemetry-attrs` and commit the result", generatedAllowlistFile)
	}
}

// TestAttributeContainsMap covers the recursion directly, independent of which
// shapes happen to exist in the provider today, so a regression in the
// TypeList/TypeSet -> Elem descent can't slip through.
func TestAttributeContainsMap(t *testing.T) {
	cases := []struct {
		name string
		s    *schema.Schema
		want bool
	}{
		{"nil", nil, false},
		{"scalar", &schema.Schema{Type: schema.TypeString}, false},
		{"direct map", &schema.Schema{Type: schema.TypeMap, Elem: &schema.Schema{Type: schema.TypeString}}, true},
		{"list of scalars", &schema.Schema{Type: schema.TypeList, Elem: &schema.Schema{Type: schema.TypeString}}, false},
		{"list of map (scalar Elem)", &schema.Schema{Type: schema.TypeList, Elem: &schema.Schema{Type: schema.TypeMap}}, true},
		{"set of block containing a map", &schema.Schema{
			Type: schema.TypeSet,
			Elem: &schema.Resource{Schema: map[string]*schema.Schema{
				"scalar": {Type: schema.TypeString},
				"nested": {Type: schema.TypeMap, Elem: &schema.Schema{Type: schema.TypeString}},
			}},
		}, true},
		{"set of block with only scalars", &schema.Schema{
			Type: schema.TypeSet,
			Elem: &schema.Resource{Schema: map[string]*schema.Schema{
				"scalar": {Type: schema.TypeString},
			}},
		}, false},
		{"list of block nesting a list of map", &schema.Schema{
			Type: schema.TypeList,
			Elem: &schema.Resource{Schema: map[string]*schema.Schema{
				"deeper": {Type: schema.TypeList, Elem: &schema.Schema{Type: schema.TypeMap}},
			}},
		}, true},
	}
	for _, tc := range cases {
		if got := attributeContainsMap(tc.s); got != tc.want {
			t.Errorf("%s: attributeContainsMap = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}
