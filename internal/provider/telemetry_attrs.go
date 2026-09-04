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
	"sort"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// This file is the source of truth for the client-analytics attribute allowlist.
// The build-time generator (cmd/telemetry-attrs-gen) serializes
// ResourceAttributeAllowlist to telemetry_attrs_allowlist.json, and a drift test
// fails CI when the committed file no longer matches the live schema.
//
// The allowlist exists so downstream telemetry only ever names statically
// declared attributes — never a map's user-controlled keys or any attribute
// value. That guarantee is structural here: the only strings emitted are the
// keys of each resource's own schema.

// ResourceAttributeAllowlist returns, for every managed resource, the sorted
// list of its top-level schema-declared attribute names.
//
// It walks the runtime ResourcesMap rather than scanning source text for an
// xResource() naming convention, so a resource whose constructor doesn't follow
// that convention is still covered. Every constructor happens to follow it
// today — "confluent_rtce_topic": rtceTopic() was the last exception, renamed
// to rtceTopicResource() when that resource's codegen marker was stamped — but
// nothing enforces it, so the map stays the source of truth. Data sources are
// excluded — v1 telemetry scope is the managed resources only.
//
// Every string in the result is a key of some resource's schema map, so the
// output can never contain a map attribute's dynamic keys or any attribute
// value. A schema.TypeMap attribute contributes only its own name.
func ResourceAttributeAllowlist() map[string][]string {
	p := New("0.0.0-dev", "")()
	return resourceAttributeAllowlistFrom(p)
}

func resourceAttributeAllowlistFrom(p *schema.Provider) map[string][]string {
	out := make(map[string][]string, len(p.ResourcesMap))
	for resourceType, r := range p.ResourcesMap {
		names := make([]string, 0, len(r.Schema))
		for name := range r.Schema {
			names = append(names, name)
		}
		sort.Strings(names)
		out[resourceType] = names
	}
	return out
}

// attributeContainsMap reports whether a schema attribute is a schema.TypeMap,
// or a TypeList/TypeSet that (recursively) contains one. These attributes carry
// user-controlled keys and values, so telemetry must never emit anything but
// their declared name; the static test uses this to force explicit
// acknowledgment of every such attribute as new ones are added.
func attributeContainsMap(s *schema.Schema) bool {
	if s == nil {
		return false
	}
	if s.Type == schema.TypeMap {
		return true
	}
	if s.Type == schema.TypeList || s.Type == schema.TypeSet {
		switch elem := s.Elem.(type) {
		case *schema.Schema:
			return attributeContainsMap(elem)
		case *schema.Resource:
			for _, child := range elem.Schema {
				if attributeContainsMap(child) {
					return true
				}
			}
		}
	}
	return false
}

// mapBearingAttributes returns the "<resource_type>.<attribute>" identifiers of
// every top-level attribute that is or contains a map, across all managed
// resources. It is the input to the static acknowledgment test.
func mapBearingAttributes(p *schema.Provider) []string {
	var ids []string
	for resourceType, r := range p.ResourcesMap {
		for name, s := range r.Schema {
			if attributeContainsMap(s) {
				ids = append(ids, resourceType+"."+name)
			}
		}
	}
	sort.Strings(ids)
	return ids
}
