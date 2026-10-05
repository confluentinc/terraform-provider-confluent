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

import "fmt"

// getNestedStreamGovernancePackageKey returns the ResourceData attribute path for
// stream_governance's nested package field. The generated resource itself checks
// paramStreamGovernance as a whole (equivalent, since package is its only leaf), but
// the retained acceptance/live tests still address the nested path directly via
// resource.TestCheckResourceAttr.
func getNestedStreamGovernancePackageKey() string {
	return fmt.Sprintf("%s.0.%s", paramStreamGovernance, paramPackage)
}
