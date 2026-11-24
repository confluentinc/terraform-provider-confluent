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

// Package xpprovider exposes the provider to Go programs outside this module,
// for example Crossplane providers generated with Upjet that compile the
// provider in-process instead of running the Terraform CLI. It does not change
// the provider's behavior and is not used by the Terraform plugin binary.
package xpprovider

import (
	"github.com/confluentinc/terraform-provider-confluent/internal/provider"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// GetProvider returns the provider, like provider.New does for main.go.
// version and userAgent are reported in the User-Agent header of API calls.
func GetProvider(version, userAgent string) *schema.Provider {
	return provider.New(version, userAgent)()
}
