// Copyright 2026 Confluent Inc. All Rights Reserved.
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
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// operationTimeouts lists the timeouts each resource declares (0 = not declared). Users can only
// set declared keys in a timeouts block, so removing one breaks their configs.
type operationTimeouts struct {
	create, read, update, delete time.Duration
}

var expectedResourceTimeouts = map[string]operationTimeouts{
	"confluent_access_point":                       {create: 2 * time.Hour, delete: 5 * time.Hour},
	"confluent_cluster_link":                       {delete: time.Hour},
	"confluent_connect_artifact":                   {create: time.Hour},
	"confluent_connector":                          {create: 24 * time.Hour, update: time.Hour},
	"confluent_dns_forwarder":                      {create: 2 * time.Hour},
	"confluent_dns_record":                         {create: 2 * time.Hour, delete: 5 * time.Hour},
	"confluent_flink_compute_pool":                 {create: time.Hour, delete: time.Hour},
	"confluent_flink_materialized_table":           {create: 6 * time.Hour, update: 20 * time.Minute, delete: 20 * time.Minute},
	"confluent_flink_statement":                    {create: 6 * time.Hour, update: 20 * time.Minute, delete: 20 * time.Minute},
	"confluent_gateway":                            {create: 2 * time.Hour},
	"confluent_kafka_cluster":                      {create: 72 * time.Hour, update: 72 * time.Hour, delete: 30 * time.Minute},
	"confluent_kafka_mirror_topic":                 {update: 5 * time.Minute, delete: time.Hour},
	"confluent_kafka_topic":                        {delete: time.Hour},
	"confluent_ksql_cluster":                       {create: 12 * time.Hour, delete: time.Hour},
	"confluent_network":                            {create: 2 * time.Hour, delete: 5 * time.Hour},
	"confluent_network_link_endpoint":              {create: 2 * time.Hour, delete: 5 * time.Hour},
	"confluent_network_link_service":               {create: 2 * time.Hour, delete: 5 * time.Hour},
	"confluent_peering":                            {create: 2 * time.Hour, delete: 5 * time.Hour},
	"confluent_private_link_access":                {create: 2 * time.Hour, delete: 5 * time.Hour},
	"confluent_private_link_attachment":            {create: 2 * time.Hour, delete: 5 * time.Hour},
	"confluent_private_link_attachment_connection": {create: 2 * time.Hour, delete: 5 * time.Hour},
	"confluent_rtce_topic":                         {create: time.Hour, delete: time.Hour},
	"confluent_schema_exporter":                    {create: 12 * time.Hour, update: 20 * time.Minute},
	"confluent_switchover_pair":                    {create: 30 * time.Minute},
	"confluent_switchover_endpoint":                {delete: 15 * time.Minute},
	"confluent_tf_importer":                        {create: 8 * time.Hour},
	"confluent_transit_gateway_attachment":         {create: 2 * time.Hour, delete: 5 * time.Hour},
}

func durationOrZero(d *time.Duration) time.Duration {
	if d == nil {
		return 0
	}
	return *d
}

func TestResourceTimeoutDefaults(t *testing.T) {
	p := New(testVersion, "")()
	for name, r := range p.ResourcesMap {
		want, ok := expectedResourceTimeouts[name]
		if r.Timeouts == nil {
			if ok {
				t.Errorf("%s: declares no timeouts, want %+v", name, want)
			}
			continue
		}
		if !ok {
			t.Errorf("%s: declares timeouts but isn't listed in expectedResourceTimeouts", name)
			continue
		}
		got := operationTimeouts{
			create: durationOrZero(r.Timeouts.Create),
			read:   durationOrZero(r.Timeouts.Read),
			update: durationOrZero(r.Timeouts.Update),
			delete: durationOrZero(r.Timeouts.Delete),
		}
		if got != want {
			t.Errorf("%s: timeouts = %+v, want %+v", name, got, want)
		}
		if r.Timeouts.Default != nil {
			t.Errorf("%s: unexpected default timeout %s", name, *r.Timeouts.Default)
		}
	}
	for name := range expectedResourceTimeouts {
		if _, ok := p.ResourcesMap[name]; !ok {
			t.Errorf("%s: not registered in ResourcesMap", name)
		}
	}
}

// Every declared key must be accepted in a user's timeouts block and land in its own field.
func TestResourceTimeoutsConfigDecode(t *testing.T) {
	userValues := map[string]time.Duration{
		schema.TimeoutCreate: 101 * time.Minute,
		schema.TimeoutRead:   102 * time.Minute,
		schema.TimeoutUpdate: 103 * time.Minute,
		schema.TimeoutDelete: 104 * time.Minute,
	}
	p := New(testVersion, "")()
	for name, want := range expectedResourceTimeouts {
		declared := map[string]time.Duration{
			schema.TimeoutCreate: want.create,
			schema.TimeoutRead:   want.read,
			schema.TimeoutUpdate: want.update,
			schema.TimeoutDelete: want.delete,
		}
		raw := map[string]interface{}{}
		for key, d := range declared {
			if d != 0 {
				raw[key] = userValues[key].String()
			}
		}
		cfg := terraform.NewResourceConfigRaw(map[string]interface{}{schema.TimeoutsConfigKey: []interface{}{raw}})
		var rt schema.ResourceTimeout
		if err := rt.ConfigDecode(p.ResourcesMap[name], cfg); err != nil {
			t.Errorf("%s: timeouts block %v rejected: %s", name, raw, err)
			continue
		}
		got := map[string]time.Duration{
			schema.TimeoutCreate: durationOrZero(rt.Create),
			schema.TimeoutRead:   durationOrZero(rt.Read),
			schema.TimeoutUpdate: durationOrZero(rt.Update),
			schema.TimeoutDelete: durationOrZero(rt.Delete),
		}
		for key, d := range declared {
			if d != 0 && got[key] != userValues[key] {
				t.Errorf("%s: %s = %s after decoding %v, want %s", name, key, got[key], raw, userValues[key])
			}
		}
	}
}

// CRUD functions must only read their own operation's timeout, e.g. a Delete passing
// d.Timeout(schema.TimeoutCreate) to its wait would silently use the wrong limit.
func TestCrudFunctionsUseOwnOperationTimeout(t *testing.T) {
	ops := map[string]string{"Create": "TimeoutCreate", "Read": "TimeoutRead", "Update": "TimeoutUpdate", "Delete": "TimeoutDelete"}
	fset := token.NewFileSet()
	files, err := filepath.Glob("resource_*.go")
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range f.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			want := ""
			for suffix, op := range ops {
				if strings.HasSuffix(fd.Name.Name, suffix) {
					want = op
				}
			}
			if want == "" {
				continue
			}
			ast.Inspect(fd.Body, func(n ast.Node) bool {
				sel, ok := n.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				if pkg, ok := sel.X.(*ast.Ident); ok && pkg.Name == "schema" {
					for _, op := range ops {
						if sel.Sel.Name == op {
							checked++
							if op != want {
								t.Errorf("%s: %s uses schema.%s, want schema.%s", fset.Position(sel.Pos()), fd.Name.Name, op, want)
							}
						}
					}
				}
				return true
			})
		}
	}
	if checked == 0 {
		t.Fatal("found no schema.Timeout* references; the check is broken")
	}
}

func TestKafkaClusterCreateWaitTimeout(t *testing.T) {
	withCreateTimeout := func(timeout *time.Duration) *schema.ResourceData {
		r := kafkaResource()
		if timeout != nil {
			r.Timeouts.Create = timeout
		}
		return r.Data(nil)
	}
	tests := []struct {
		name        string
		clusterType string
		userTimeout *time.Duration
		expected    time.Duration
	}{
		{"dedicated default", kafkaClusterTypeDedicated, nil, 72 * time.Hour},
		{"basic default", kafkaClusterTypeBasic, nil, time.Hour},
		{"standard default", kafkaClusterTypeStandard, nil, time.Hour},
		{"enterprise default", kafkaClusterTypeEnterprise, nil, time.Hour},
		{"freight default", kafkaClusterTypeFreight, nil, time.Hour},
		{"dedicated user shortened", kafkaClusterTypeDedicated, schema.DefaultTimeout(2 * time.Hour), 2 * time.Hour},
		{"dedicated user extended", kafkaClusterTypeDedicated, schema.DefaultTimeout(100 * time.Hour), 100 * time.Hour},
		{"basic user shortened", kafkaClusterTypeBasic, schema.DefaultTimeout(30 * time.Minute), 30 * time.Minute},
		{"standard user extended", kafkaClusterTypeStandard, schema.DefaultTimeout(3 * time.Hour), 3 * time.Hour},
		// An explicit 72h can't be told apart from the default (documented on the resource page).
		{"basic user sets 72h", kafkaClusterTypeBasic, schema.DefaultTimeout(72 * time.Hour), time.Hour},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := kafkaClusterCreateWaitTimeout(withCreateTimeout(tt.userTimeout), tt.clusterType); got != tt.expected {
				t.Fatalf("kafkaClusterCreateWaitTimeout(%q) = %s, want %s", tt.clusterType, got, tt.expected)
			}
		})
	}
}
