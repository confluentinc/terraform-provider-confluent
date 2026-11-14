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

import "time"

// telemetryExitTimeout caps how long the provider process waits at exit for
// unsent analytics events, counted from the transport's last activity (see
// Drain). It stays well under the 2 s Terraform allows a provider to exit before
// killing it.
const telemetryExitTimeout = 500 * time.Millisecond

// FlushTelemetry gives this process's unsent analytics events a bounded wait
// (see telemetryExitTimeout). main calls it after the plugin server stops; it
// returns at once when reporting is disabled or nothing is unsent.
func FlushTelemetry() {
	rt := publishedTelemetry.Load()
	if rt == nil {
		return
	}
	if d, ok := rt.reporter.(interface{ Drain(time.Duration) bool }); ok {
		d.Drain(telemetryExitTimeout)
	}
}
