// Copyright 2022 Confluent Inc. All Rights Reserved.
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
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	networkingdnsforwarderv1 "github.com/confluentinc/ccloud-sdk-go-v2/networking-dnsforwarder/v1"
)

// requiredGateway is the schema helper for the required, ForceNew gateway block shared by
// confluent_dns_forwarder and confluent_access_point. It lives here (not in the generated,
// DO NOT EDIT resource_dns_forwarder.go) so the generator's objectref_schema_overrides can
// reference it and it survives regeneration. Its `id` carries no ValidateFunc — the shared
// spec ObjectReference has no per-referent constraint — matching the published resource.
func requiredGateway() *schema.Schema {
	return &schema.Schema{
		Type: schema.TypeList,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramId: {
					Type:        schema.TypeString,
					Required:    true,
					ForceNew:    true,
					Description: "The unique identifier for the gateway.",
				},
			},
		},
		Required: true,
		MinItems: 1,
		MaxItems: 1,
		ForceNew: true,
	}
}

// buildDomainMappings converts the forward_via_gcp_dns_zones block's domain_mappings TypeMap
// (domain name -> "zone,project" string) into the SDK's map[string]{Zone,Project}. It is the
// build_func half of the generator's terraform.variant_field_serializers for this resource;
// the generated create/update calls it via
// buildDomainMappings(d, paramForwardViaGcpDnsZones, paramForwardViaGcpDnsZonesDomainMappings).
//
// Non-error by contract: the schema's Elem ValidateFunc (`^[^,]+,[^,]+$`) has already rejected
// any value that is not exactly "zone,project" before Create/Update runs, so unlike the former
// hand-written convertToStringObjectMap there is no error path — a value that somehow lacks the
// comma is skipped defensively rather than propagated.
func buildDomainMappings(d *schema.ResourceData, blockName, fieldName string) map[string]networkingdnsforwarderv1.NetworkingV1ForwardViaGcpDnsZonesDomainMappings {
	raw := d.Get(fmt.Sprintf("%s.0.%s", blockName, fieldName)).(map[string]interface{})
	// The schema's Elem ValidateFunc (`^[^,]+,[^,]+$`) has already rejected any value that is
	// not exactly "zone,project" before Create/Update runs, so convertToStringObjectMap's error
	// is unreachable here; discard it.
	result, _ := convertToStringObjectMap(convertToStringStringMap(raw))
	return result
}

// convertToStringObjectMap parses a "zone,project" map into the SDK's {Zone,Project} map. Preserved
// from the former hand-written resource (it is exercised by utils_test.go) and reused by
// buildDomainMappings so the parsing lives in one place.
func convertToStringObjectMap(data map[string]string) (map[string]networkingdnsforwarderv1.NetworkingV1ForwardViaGcpDnsZonesDomainMappings, error) {
	stringMap := make(map[string]networkingdnsforwarderv1.NetworkingV1ForwardViaGcpDnsZonesDomainMappings)
	for key, value := range data {
		if len(strings.Split(value, ",")) != 2 {
			return nil, fmt.Errorf(`the mapping format of "%s" is incorrect. The correct format should be domainName=zoneName,projectName`, value)
		}
		s := strings.SplitN(value, ",", 2)
		zone := strings.TrimSpace(s[0])
		project := strings.TrimSpace(s[1])
		stringMap[key] = networkingdnsforwarderv1.NetworkingV1ForwardViaGcpDnsZonesDomainMappings{
			Zone:    networkingdnsforwarderv1.PtrString(zone),
			Project: networkingdnsforwarderv1.PtrString(project),
		}
	}
	return stringMap, nil
}

// flattenDomainMappings is the read-back (flatten_func) half of the serializer: it joins each
// SDK {Zone,Project} back into the published "zone,project" string, matching the hand-written
// setDnsForwarderAttributes exactly (only entries with both fields present are surfaced).
func flattenDomainMappings(domainMappings map[string]networkingdnsforwarderv1.NetworkingV1ForwardViaGcpDnsZonesDomainMappings) map[string]string {
	stringMap := make(map[string]string, len(domainMappings))
	for key, value := range domainMappings {
		zone, zoneOk := value.GetZoneOk()
		project, projectOk := value.GetProjectOk()
		if zoneOk && projectOk {
			stringMap[key] = *zone + "," + *project
		}
	}
	return stringMap
}
