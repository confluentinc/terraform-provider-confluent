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
	"context"
	"encoding/json"
	"fmt"
	mds "github.com/confluentinc/ccloud-sdk-go-v2/mds/v2"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"net/http"
	"regexp"
	"time"
)

const (
	paramRoleName   = "role_name"
	paramCrnPattern = "crn_pattern"

	rbacWaitAfterCreateToSync = 90 * time.Second
)

func roleBindingResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: roleBindingCreate,
		ReadContext:   roleBindingRead,
		DeleteContext: roleBindingDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			paramPrincipal: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "The principal User to bind the role to.",
				ValidateFunc: validation.StringMatch(regexp.MustCompile("^User:"), "the Principal must be of the form 'User:'"),
			},
			paramRoleName: {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The name of the role to bind to the principal.",
			},
			paramCrnPattern: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "A CRN that specifies the scope and resource patterns necessary for the role to bind.",
				ValidateFunc: validation.StringMatch(regexp.MustCompile("^crn://"), "the CRN must be of the form 'crn://'"),
			},
			paramDisableWaitForReady: {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
		},
	}
}

func roleBindingCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client)

	principal := d.Get(paramPrincipal).(string)
	roleName := d.Get(paramRoleName).(string)
	crnPattern := d.Get(paramCrnPattern).(string)
	skipSync := d.Get(paramDisableWaitForReady).(bool)

	createRoleBindingRequest := mds.NewIamV2RoleBinding()
	createRoleBindingRequest.SetPrincipal(principal)
	createRoleBindingRequest.SetRoleName(roleName)
	createRoleBindingRequest.SetCrnPattern(crnPattern)
	createRoleBindingRequestJson, err := json.Marshal(createRoleBindingRequest)
	if err != nil {
		return diag.Errorf("error creating Role Binding: error marshaling %#v to json: %s", createRoleBindingRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Creating new Role Binding: %s", createRoleBindingRequestJson))

	createdRoleBinding, _, err := executeRoleBindingCreate(c.mdsApiContext(ctx), c, createRoleBindingRequest)
	if err != nil {
		return diag.Errorf("error creating Role Binding: %s", createDescriptiveError(err))
	}
	d.SetId(createdRoleBinding.GetId())

	createdRoleBindingJson, err := json.Marshal(createdRoleBinding)
	if err != nil {
		return diag.Errorf("error creating Role Binding: %q: error marshaling %#v to json: %s", createdRoleBinding.GetId(), createdRoleBinding, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished creating Role Binding %q: %s", d.Id(), createdRoleBindingJson), map[string]interface{}{roleBindingLoggingKey: d.Id()})
	if !skipSync {
		SleepIfNotTestMode(rbacWaitAfterCreateToSync, meta.(*Client).isAcceptanceTestMode)
	}
	return roleBindingRead(ctx, d, meta)
}

func executeRoleBindingCreate(ctx context.Context, c *Client, roleBinding *mds.IamV2RoleBinding) (mds.IamV2RoleBinding, *http.Response, error) {
	req := c.mdsClient.RoleBindingsIamV2Api.CreateIamV2RoleBinding(c.mdsApiContext(ctx)).IamV2RoleBinding(*roleBinding)
	return req.Execute()
}

func roleBindingDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Deleting Role Binding %q", d.Id()), map[string]interface{}{roleBindingLoggingKey: d.Id()})
	c := meta.(*Client)

	req := c.mdsClient.RoleBindingsIamV2Api.DeleteIamV2RoleBinding(c.mdsApiContext(ctx), d.Id())
	_, err := req.Execute()

	if err != nil {
		return diag.Errorf("error deleting Role Binding %q: %s", d.Id(), createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting Role Binding %q", d.Id()), map[string]interface{}{roleBindingLoggingKey: d.Id()})

	return nil
}

func roleBindingRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Role Binding %q", d.Id()), map[string]interface{}{roleBindingLoggingKey: d.Id()})
	c := meta.(*Client)
	roleBinding, resp, err := executeRoleBindingRead(c.mdsApiContext(ctx), c, d.Id())
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Role Binding %q: %s", d.Id(), createDescriptiveError(err)), map[string]interface{}{roleBindingLoggingKey: d.Id()})

		isResourceNotFound := isNonKafkaRestApiResourceNotFound(resp)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing Role Binding %q in TF state because Role Binding could not be found on the server", d.Id()), map[string]interface{}{roleBindingLoggingKey: d.Id()})
			d.SetId("")
			return nil
		}

		return diag.FromErr(createDescriptiveError(err))
	}
	roleBindingJson, err := json.Marshal(roleBinding)
	if err != nil {
		return diag.Errorf("error reading Role Binding %q: error marshaling %#v to json: %s", d.Id(), roleBinding, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Role Binding %q: %s", d.Id(), roleBindingJson), map[string]interface{}{roleBindingLoggingKey: d.Id()})

	if _, err := setRoleBindingAttributes(d, roleBinding); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Role Binding %q", d.Id()), map[string]interface{}{roleBindingLoggingKey: d.Id()})

	return nil
}
func executeRoleBindingRead(ctx context.Context, c *Client, roleBindingId string) (mds.IamV2RoleBinding, *http.Response, error) {
	req := c.mdsClient.RoleBindingsIamV2Api.GetIamV2RoleBinding(c.mdsApiContext(ctx), roleBindingId)
	return req.Execute()
}

func setRoleBindingAttributes(d *schema.ResourceData, roleBinding mds.IamV2RoleBinding) (*schema.ResourceData, error) {
	if err := d.Set(paramPrincipal, roleBinding.GetPrincipal()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramRoleName, roleBinding.GetRoleName()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramCrnPattern, roleBinding.GetCrnPattern()); err != nil {
		return nil, createDescriptiveError(err)
	}
	// Explicitly set paramDisableWaitForReady to the default value if unset
	if _, ok := d.GetOk(paramDisableWaitForReady); !ok {
		if err := d.Set(paramDisableWaitForReady, d.Get(paramDisableWaitForReady)); err != nil {
			return nil, createDescriptiveError(err)
		}
	}

	d.SetId(roleBinding.GetId())
	return d, nil
}
