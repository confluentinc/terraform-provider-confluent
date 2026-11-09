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
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
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
	"confluent_kafka_mirror_topic":                 {update: 20 * time.Minute, delete: time.Hour},
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
	"confluent_schema_exporter":                    {create: 12 * time.Hour, update: 12 * time.Hour},
	"confluent_switchover_pair":                    {create: 30 * time.Minute},
	"confluent_switchover_endpoint":                {delete: 20 * time.Minute},
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

// Helpers that read a timeout on behalf of one CRUD operation.
var crudHelperOps = map[string]string{
	"flinkStatementStop":            "TimeoutUpdate",
	"flinkStatementResume":          "TimeoutUpdate",
	"resumeExporter":                "TimeoutUpdate",
	"kafkaClusterCreateWaitTimeout": "TimeoutCreate",
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
	seenHelpers := map[string]bool{}
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
			if !ok || fd.Body == nil {
				continue
			}
			want := ""
			for suffix, op := range ops {
				if strings.HasSuffix(fd.Name.Name, suffix) {
					want = op
				}
			}
			if op, ok := crudHelperOps[fd.Name.Name]; ok {
				want = op
				seenHelpers[fd.Name.Name] = true
			}
			ast.Inspect(fd.Body, func(n ast.Node) bool {
				// A helper that reads one operation's timeout must only be called from that operation.
				if call, ok := n.(*ast.CallExpr); ok {
					if fn, ok := call.Fun.(*ast.Ident); ok {
						if op, ok := crudHelperOps[fn.Name]; ok && op != want {
							t.Errorf("%s: %s calls %s, which reads schema.%s; call it only from that operation", fset.Position(call.Pos()), fd.Name.Name, fn.Name, op)
						}
					}
				}
				sel, ok := n.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				if pkg, ok := sel.X.(*ast.Ident); ok && pkg.Name == "schema" {
					for _, op := range ops {
						if sel.Sel.Name == op {
							checked++
							if want == "" {
								t.Errorf("%s: %s reads schema.%s but isn't a Create/Read/Update/Delete function; add it to crudHelperOps", fset.Position(sel.Pos()), fd.Name.Name, op)
							} else if op != want {
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
	for name := range crudHelperOps {
		if !seenHelpers[name] {
			t.Errorf("crudHelperOps lists %s, which no longer exists in resource_*.go; remove it", name)
		}
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

// Waits that keep a short fixed limit inside a longer operation, by design.
var fixedLimitWaits = map[string]bool{
	"waitForAnySchemaRegistryClusterToProvision": true,
	"waitForTagToProvision":                      true,
	"waitForBusinessMetadataToProvision":         true,
	"waitForTagBindingToProvision":               true,
	"waitForBusinessMetadataBindingToProvision":  true,
}

// Every other status-polling wait must stop at its operation's timeout, so a timeouts value can
// lengthen it as well as shorten it. A wait edited or regenerated back to a fixed limit fails here,
// and so does a caller that passes anything but d.Timeout(schema.Timeout*) or a forwarded timeout.
func TestStatusWaitsUseOperationTimeout(t *testing.T) {
	fset := token.NewFileSet()
	paths, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	var files []*ast.File
	for _, path := range paths {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, f)
	}

	// Index of each function's `timeout` parameter.
	timeoutParam := map[string]int{}
	for _, f := range files {
		for _, decl := range f.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			i := 0
			for _, field := range fd.Type.Params.List {
				for _, name := range field.Names {
					if name.Name == "timeout" {
						timeoutParam[fd.Name.Name] = i
					}
					i++
				}
			}
		}
	}
	if len(timeoutParam) < 40 {
		t.Fatalf("found only %d functions with a timeout parameter; the check is broken", len(timeoutParam))
	}

	seenFixed := map[string]bool{}
	for _, f := range files {
		for _, decl := range f.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok || fd.Body == nil {
				continue
			}
			_, isWait := timeoutParam[fd.Name.Name]
			ast.Inspect(fd.Body, func(n ast.Node) bool {
				if fd.Name.Name != "kafkaClusterCreateWaitTimeout" {
					if assign, ok := n.(*ast.AssignStmt); ok {
						for _, lhs := range assign.Lhs {
							if id, ok := lhs.(*ast.Ident); ok && id.Name == "timeout" {
								t.Errorf("%s: %s assigns timeout; pass the operation's timeout through unchanged", fset.Position(assign.Pos()), fd.Name.Name)
							}
							// e.g. stateConf.Timeout = time.Hour after the StateChangeConf literal.
							if sel, ok := lhs.(*ast.SelectorExpr); ok && isWait && sel.Sel.Name == "Timeout" {
								t.Errorf("%s: %s sets %s.Timeout; set it in the StateChangeConf literal from the timeout parameter", fset.Position(assign.Pos()), fd.Name.Name, exprName(sel.X))
							}
						}
					}
					if spec, ok := n.(*ast.ValueSpec); ok {
						for _, name := range spec.Names {
							if name.Name == "timeout" {
								t.Errorf("%s: %s declares timeout; pass the operation's timeout through unchanged", fset.Position(spec.Pos()), fd.Name.Name)
							}
						}
					}
				}
				// A shorter context inside a wait would end it before its timeout.
				if call, ok := n.(*ast.CallExpr); ok && isWait {
					if sel, ok := call.Fun.(*ast.SelectorExpr); ok && exprName(sel.X) == "context" && (sel.Sel.Name == "WithTimeout" || sel.Sel.Name == "WithDeadline") {
						t.Errorf("%s: %s calls context.%s; let the wait stop at its timeout parameter", fset.Position(call.Pos()), fd.Name.Name, sel.Sel.Name)
					}
				}
				lit, ok := n.(*ast.CompositeLit)
				if !ok {
					return true
				}
				if sel, ok := lit.Type.(*ast.SelectorExpr); !ok || sel.Sel.Name != "StateChangeConf" {
					return true
				}
				if fixedLimitWaits[fd.Name.Name] {
					seenFixed[fd.Name.Name] = true
					return true
				}
				if _, ok := timeoutParam[fd.Name.Name]; !ok {
					t.Errorf("%s: %s polls with a StateChangeConf but takes no timeout parameter", fset.Position(lit.Pos()), fd.Name.Name)
					return true
				}
				usesParam := false
				for _, elt := range lit.Elts {
					kv, ok := elt.(*ast.KeyValueExpr)
					if !ok {
						continue
					}
					if key, ok := kv.Key.(*ast.Ident); ok && key.Name == "Timeout" {
						v, ok := kv.Value.(*ast.Ident)
						usesParam = ok && v.Name == "timeout"
					}
				}
				if !usesParam {
					t.Errorf("%s: %s must set StateChangeConf.Timeout to its timeout parameter", fset.Position(lit.Pos()), fd.Name.Name)
				}
				return true
			})
		}
	}
	for name := range fixedLimitWaits {
		if !seenFixed[name] {
			t.Errorf("fixedLimitWaits lists %s, which no longer polls; remove it", name)
		}
	}

	for _, f := range files {
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			fn, ok := call.Fun.(*ast.Ident)
			if !ok {
				return true
			}
			i, ok := timeoutParam[fn.Name]
			if !ok || i >= len(call.Args) {
				return true
			}
			if !isOperationTimeoutArg(call.Args[i]) {
				t.Errorf("%s: %s must get d.Timeout(schema.Timeout*) or a forwarded timeout", fset.Position(call.Pos()), fn.Name)
			}
			return true
		})
	}
}

func exprName(e ast.Expr) string {
	if id, ok := e.(*ast.Ident); ok {
		return id.Name
	}
	return ""
}

func isOperationTimeoutArg(arg ast.Expr) bool {
	if id, ok := arg.(*ast.Ident); ok {
		return id.Name == "timeout"
	}
	call, ok := arg.(*ast.CallExpr)
	if !ok {
		return false
	}
	if fn, ok := call.Fun.(*ast.Ident); ok {
		return fn.Name == "kafkaClusterCreateWaitTimeout"
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Timeout" || len(call.Args) != 1 {
		return false
	}
	op, ok := call.Args[0].(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := op.X.(*ast.Ident)
	return ok && pkg.Name == "schema" && strings.HasPrefix(op.Sel.Name, "Timeout")
}

// Calling the CRUD function directly skips the SDK's context deadline, so only the wait's own
// limit can stop it: this proves a timeouts.delete value reaches the wait.
func TestKafkaTopicDeleteWaitUsesConfiguredTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		// The topic never goes away.
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"kind":"KafkaTopic","topic_name":"` + topicName + `"}`))
	}))
	defer server.Close()

	r := kafkaTopicResource()
	configured := 2 * time.Second
	r.Timeouts.Delete = &configured
	d := r.Data(nil)
	d.SetId(createKafkaTopicId(clusterId, topicName))
	_ = d.Set(paramTopicName, topicName)
	_ = d.Set(paramRestEndpoint, server.URL)
	_ = d.Set(paramKafkaCluster, []interface{}{map[string]interface{}{paramId: clusterId}})
	_ = d.Set(paramCredentials, []interface{}{map[string]interface{}{paramKey: kafkaApiKey, paramSecret: kafkaApiSecret}})
	zero := 0
	meta := &Client{
		isAcceptanceTestMode:   true,
		kafkaRestClientFactory: &KafkaRestClientFactory{ctx: context.Background(), userAgent: "test", maxRetries: &zero},
	}

	done := make(chan diag.Diagnostics, 1)
	go func() { done <- kafkaTopicDelete(context.Background(), d, meta) }()
	select {
	case diags := <-done:
		if !diags.HasError() || !strings.Contains(diags[0].Summary, "timeout: 2s") {
			t.Fatalf("got %v, want the delete wait to stop at the configured 2s", diags)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("the delete wait ignored timeouts.delete = 2s")
	}
}
