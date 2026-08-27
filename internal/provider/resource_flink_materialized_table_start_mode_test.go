package provider

import (
	"testing"
	"time"

	flinkgatewayv1 "github.com/confluentinc/ccloud-sdk-go-v2-internal/flink-gateway/v1"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func startModeResourceData(t *testing.T, startMode []interface{}) *schema.ResourceData {
	t.Helper()
	raw := map[string]interface{}{}
	if startMode != nil {
		raw[paramStartMode] = startMode
	}
	return schema.TestResourceDataRaw(t, flinkMaterializedTableResource().Schema, raw)
}

func TestExpandMaterializedTableStartMode(t *testing.T) {
	tests := []struct {
		name         string
		startMode    []interface{}
		wantNil      bool
		wantErr      bool
		wantKind     string
		wantTS       string // RFC3339, or "" when unset
		wantInterval int32  // 0 when unset
		wantUnit     string
	}{
		{
			name:      "absent block yields nil",
			startMode: nil,
			wantNil:   true,
		},
		{
			name: "kind only",
			startMode: []interface{}{
				map[string]interface{}{paramStartModeKind: "FROM_BEGINNING"},
			},
			wantKind: "FROM_BEGINNING",
		},
		{
			name: "from_now with interval",
			startMode: []interface{}{
				map[string]interface{}{
					paramStartModeKind: "FROM_NOW",
					paramStartModeTimeInterval: []interface{}{
						map[string]interface{}{paramIntervalValue: 2, paramIntervalTimeUnit: "HOURS"},
					},
				},
			},
			wantKind:     "FROM_NOW",
			wantInterval: 2,
			wantUnit:     "HOURS",
		},
		{
			name: "from_timestamp with timestamp",
			startMode: []interface{}{
				map[string]interface{}{
					paramStartModeKind:      "FROM_TIMESTAMP",
					paramStartModeTimestamp: "2026-04-01T00:00:00Z",
				},
			},
			wantKind: "FROM_TIMESTAMP",
			wantTS:   "2026-04-01T00:00:00Z",
		},
		{
			name: "from_timestamp missing timestamp errors",
			startMode: []interface{}{
				map[string]interface{}{paramStartModeKind: "FROM_TIMESTAMP"},
			},
			wantErr: true,
		},
		{
			name: "resume_or_from_timestamp missing timestamp errors",
			startMode: []interface{}{
				map[string]interface{}{paramStartModeKind: "RESUME_OR_FROM_TIMESTAMP"},
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := startModeResourceData(t, tc.startMode)
			got, err := expandMaterializedTableStartMode(d)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if tc.wantNil {
				if got != nil {
					t.Fatalf("expected nil start mode, got %#v", got)
				}
				return
			}
			if got == nil {
				t.Fatalf("expected non-nil start mode")
			}
			if got.GetKind() != tc.wantKind {
				t.Errorf("kind = %q, want %q", got.GetKind(), tc.wantKind)
			}
			if tc.wantTS == "" {
				if got.Timestamp != nil {
					t.Errorf("timestamp = %v, want unset", got.GetTimestamp())
				}
			} else {
				if got.Timestamp == nil {
					t.Errorf("timestamp unset, want %q", tc.wantTS)
				} else if got.GetTimestamp().Format(time.RFC3339) != tc.wantTS {
					t.Errorf("timestamp = %q, want %q", got.GetTimestamp().Format(time.RFC3339), tc.wantTS)
				}
			}
			if tc.wantInterval == 0 {
				if got.TimeInterval != nil {
					t.Errorf("time_interval = %#v, want unset", got.TimeInterval)
				}
			} else {
				if got.TimeInterval == nil {
					t.Fatalf("time_interval unset, want interval=%d unit=%q", tc.wantInterval, tc.wantUnit)
				}
				if got.TimeInterval.GetInterval() != tc.wantInterval {
					t.Errorf("interval = %d, want %d", got.TimeInterval.GetInterval(), tc.wantInterval)
				}
				if got.TimeInterval.GetTimeUnit() != tc.wantUnit {
					t.Errorf("time_unit = %q, want %q", got.TimeInterval.GetTimeUnit(), tc.wantUnit)
				}
			}
		})
	}
}

func TestFlattenMaterializedTableStartMode(t *testing.T) {
	t.Run("nil yields empty list", func(t *testing.T) {
		got := flattenMaterializedTableStartMode(nil)
		if len(got) != 0 {
			t.Fatalf("expected empty list, got %#v", got)
		}
	})

	t.Run("kind with interval", func(t *testing.T) {
		sm := &flinkgatewayv1.SqlV1MaterializedTableStartMode{}
		sm.SetKind("FROM_NOW")
		interval := flinkgatewayv1.SqlV1IntervalExpression{}
		interval.SetInterval(2)
		interval.SetTimeUnit("HOURS")
		sm.SetTimeInterval(interval)

		got := flattenMaterializedTableStartMode(sm)
		if len(got) != 1 {
			t.Fatalf("expected one block, got %#v", got)
		}
		block := got[0].(map[string]interface{})
		if block[paramStartModeKind] != "FROM_NOW" {
			t.Errorf("kind = %v, want FROM_NOW", block[paramStartModeKind])
		}
		if block[paramStartModeTimestamp] != "" {
			t.Errorf("timestamp should be empty when unset, got %v", block[paramStartModeTimestamp])
		}
		ti := block[paramStartModeTimeInterval].([]interface{})
		if len(ti) != 1 {
			t.Fatalf("expected one time_interval block, got %#v", ti)
		}
		tiBlock := ti[0].(map[string]interface{})
		if tiBlock[paramIntervalValue] != 2 {
			t.Errorf("interval = %v, want 2", tiBlock[paramIntervalValue])
		}
		if tiBlock[paramIntervalTimeUnit] != "HOURS" {
			t.Errorf("time_unit = %v, want HOURS", tiBlock[paramIntervalTimeUnit])
		}
	})

	// Regression guard for the kind-switch stale-sibling leak: a kind that carries
	// no timestamp/time_interval must flatten them to explicit empty values, not
	// omit the keys. Omitting a Computed nested-list key makes the SDK retain the
	// prior state value, which leaks e.g. a FROM_NOW time_interval into a later
	// FROM_BEGINNING state until the next refresh.
	t.Run("kind only clears timestamp and time_interval", func(t *testing.T) {
		sm := &flinkgatewayv1.SqlV1MaterializedTableStartMode{}
		sm.SetKind("FROM_BEGINNING")

		got := flattenMaterializedTableStartMode(sm)
		if len(got) != 1 {
			t.Fatalf("expected one block, got %#v", got)
		}
		block := got[0].(map[string]interface{})
		if block[paramStartModeKind] != "FROM_BEGINNING" {
			t.Errorf("kind = %v, want FROM_BEGINNING", block[paramStartModeKind])
		}
		if block[paramStartModeTimestamp] != "" {
			t.Errorf("timestamp should be empty when unset, got %v", block[paramStartModeTimestamp])
		}
		ti, ok := block[paramStartModeTimeInterval].([]interface{})
		if !ok {
			t.Fatalf("time_interval should be an empty list, got %#v", block[paramStartModeTimeInterval])
		}
		if len(ti) != 0 {
			t.Errorf("time_interval should be empty when unset, got %#v", ti)
		}
	})

	t.Run("kind with timestamp", func(t *testing.T) {
		sm := &flinkgatewayv1.SqlV1MaterializedTableStartMode{}
		sm.SetKind("FROM_TIMESTAMP")
		sm.SetTimestamp(time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC))

		got := flattenMaterializedTableStartMode(sm)
		block := got[0].(map[string]interface{})
		if block[paramStartModeTimestamp] != "2026-04-01T00:00:00Z" {
			t.Errorf("timestamp = %v, want 2026-04-01T00:00:00Z", block[paramStartModeTimestamp])
		}
		ti, ok := block[paramStartModeTimeInterval].([]interface{})
		if !ok {
			t.Fatalf("time_interval should be an empty list, got %#v", block[paramStartModeTimeInterval])
		}
		if len(ti) != 0 {
			t.Errorf("time_interval should be empty when unset, got %#v", ti)
		}
	})
}

func TestExpandFlattenStartModeRoundTrip(t *testing.T) {
	// Config value -> API object -> state value should be stable, which is what
	// keeps Terraform from showing a perpetual diff.
	startMode := []interface{}{
		map[string]interface{}{
			paramStartModeKind:      "RESUME_OR_FROM_TIMESTAMP",
			paramStartModeTimestamp: "2026-04-01T00:00:00Z",
		},
	}
	d := startModeResourceData(t, startMode)
	expanded, err := expandMaterializedTableStartMode(d)
	if err != nil {
		t.Fatalf("expand error: %s", err)
	}
	flattened := flattenMaterializedTableStartMode(expanded)
	block := flattened[0].(map[string]interface{})
	if block[paramStartModeKind] != "RESUME_OR_FROM_TIMESTAMP" {
		t.Errorf("kind round trip = %v", block[paramStartModeKind])
	}
	if block[paramStartModeTimestamp] != "2026-04-01T00:00:00Z" {
		t.Errorf("timestamp round trip = %v", block[paramStartModeTimestamp])
	}
}

func TestSuppressFlinkTimestampDiff(t *testing.T) {
	tests := []struct {
		name string
		old  string
		new  string
		want bool
	}{
		{name: "identical", old: "2026-04-01T00:00:00Z", new: "2026-04-01T00:00:00Z", want: true},
		{name: "same instant different offset", old: "2026-04-01T02:00:00+02:00", new: "2026-04-01T00:00:00Z", want: true},
		{name: "different instant", old: "2026-04-01T00:00:00Z", new: "2026-04-02T00:00:00Z", want: false},
		{name: "empty to value", old: "", new: "2026-04-01T00:00:00Z", want: false},
		{name: "unparseable", old: "not-a-time", new: "2026-04-01T00:00:00Z", want: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := suppressFlinkTimestampDiff("", tc.old, tc.new, nil); got != tc.want {
				t.Errorf("suppressFlinkTimestampDiff(%q, %q) = %v, want %v", tc.old, tc.new, got, tc.want)
			}
		})
	}
}
