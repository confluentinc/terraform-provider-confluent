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
	v2 "github.com/confluentinc/ccloud-sdk-go-v2/iam/v2"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"net/http"
)

const (
	// The maximum allowable page size when listing service accounts using IAM V2 API
	// https://docs.confluent.io/cloud/current/api.html#operation/listIamV2Users
	listUsersPageSize = 100
	paramEmail        = "email"
	paramFullName     = "full_name"
)

func userDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: userDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramId: {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
				// Exactly one from "id", "full_name", or "email" attributes must be specified.
				ExactlyOneOf: []string{paramId, paramFullName, paramEmail},
				Description:  "The ID of the User (e.g., `u-abc123`).",
			},
			paramApiVersion: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "API Version defines the schema version of this representation of a User.",
			},
			paramKind: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Kind defines the object User represents.",
			},
			paramEmail: {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
				// Exactly one from "id", "full_name", or "email" attributes must be specified.
				ExactlyOneOf: []string{paramId, paramFullName, paramEmail},
				Description:  "The email address of the User.",
			},
			paramFullName: {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
				// Exactly one from "id", "full_name", or "email" attributes must be specified.
				ExactlyOneOf: []string{paramId, paramFullName, paramEmail},
				Description:  "The full name of the User.",
			},
		},
	}
}

func userDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// ExactlyOneOf specified in the schema ensures one of paramId, paramFullName, or paramEmail is specified.
	// The next step is to figure out which one exactly is set.

	userId := d.Get(paramId).(string)
	fullName := d.Get(paramFullName).(string)
	email := d.Get(paramEmail).(string)

	if userId != "" {
		return userDataSourceReadUsingId(ctx, d, meta, userId)
	} else if fullName != "" {
		return userDataSourceReadUsingFullName(ctx, d, meta, fullName)
	} else if email != "" {
		return userDataSourceReadUsingEmail(ctx, d, meta, email)
	} else {
		return diag.Errorf("error reading User: exactly one of %q, %q or %q must be specified but they're all empty", paramId, paramFullName, paramEmail)
	}
}

func userDataSourceReadUsingFullName(ctx context.Context, d *schema.ResourceData, meta interface{}, fullName string) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading User %q=%q", paramFullName, fullName))

	client := meta.(*Client)
	users, err := loadUsers(ctx, client)
	if err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}
	if orgHasMultipleUsersWithTargetFullname(users, fullName) {
		return diag.Errorf("error reading User: there are multiple Users with %q=%q", paramFullName, fullName)
	}
	for _, user := range users {
		if user.GetFullName() == fullName {
			return setUserAttributes(d, user)
		}
	}

	return diag.Errorf("error reading User: User with %q=%q was not found", paramFullName, fullName)
}

func userDataSourceReadUsingEmail(ctx context.Context, d *schema.ResourceData, meta interface{}, email string) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading User %q=%q", paramEmail, email))

	client := meta.(*Client)
	users, err := loadUsers(ctx, client)
	if err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}
	if orgHasMultipleUsersWithTargetEmail(users, email) {
		return diag.Errorf("error reading User: there are multiple Users with %q=%q", paramEmail, email)
	}
	for _, user := range users {
		if user.GetEmail() == email {
			return setUserAttributes(d, user)
		}
	}

	return diag.Errorf("error reading User: User with %q=%q was not found", paramEmail, email)
}

func loadUsers(ctx context.Context, c *Client) ([]v2.IamV2User, error) {
	users := make([]v2.IamV2User, 0)

	collectedAllUsers := false
	pageToken := ""
	for !collectedAllUsers {
		userList, resp, err := executeListUsers(ctx, c, pageToken)
		if err != nil {
			return nil, fmt.Errorf("error reading Users: %s", createDescriptiveError(err, resp))
		}
		users = append(users, userList.GetData()...)

		// nextPageUrlStringNullable is nil for the last page
		nextPageUrlStringNullable := userList.GetMetadata().Next

		if nextPageUrlStringNullable.IsSet() {
			nextPageUrlString := *nextPageUrlStringNullable.Get()
			if nextPageUrlString == "" {
				collectedAllUsers = true
			} else {
				pageToken, err = extractPageToken(nextPageUrlString)
				if err != nil {
					return nil, fmt.Errorf("error reading Users: %s", createDescriptiveError(err, resp))
				}
			}
		} else {
			collectedAllUsers = true
		}
	}
	return users, nil
}

func executeListUsers(ctx context.Context, c *Client, pageToken string) (v2.IamV2UserList, *http.Response, error) {
	if pageToken != "" {
		return c.iamClient.UsersIamV2Api.ListIamV2Users(c.iamApiContext(ctx)).PageSize(listUsersPageSize).PageToken(pageToken).Execute()
	} else {
		return c.iamClient.UsersIamV2Api.ListIamV2Users(c.iamApiContext(ctx)).PageSize(listUsersPageSize).Execute()
	}
}

func userDataSourceReadUsingId(ctx context.Context, d *schema.ResourceData, meta interface{}, userId string) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading User %q=%q", paramId, userId), map[string]interface{}{userLoggingKey: userId})

	c := meta.(*Client)
	user, resp, err := executeUserRead(c.iamApiContext(ctx), c, userId)
	if err != nil {
		return diag.Errorf("error reading User %q: %s", userId, createDescriptiveError(err, resp))
	}
	userJson, err := json.Marshal(user)
	if err != nil {
		return diag.Errorf("error reading User %q: error marshaling %#v to json: %s", userId, user, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched User %q: %#v", userId, userJson), map[string]interface{}{userLoggingKey: userId})
	return setUserAttributes(d, user)
}

func setUserAttributes(d *schema.ResourceData, user v2.IamV2User) diag.Diagnostics {
	if err := d.Set(paramApiVersion, user.GetApiVersion()); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}
	if err := d.Set(paramKind, user.GetKind()); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}
	if err := d.Set(paramEmail, user.GetEmail()); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}
	if err := d.Set(paramFullName, user.GetFullName()); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}
	d.SetId(user.GetId())
	return nil
}

func orgHasMultipleUsersWithTargetFullname(users []v2.IamV2User, fullName string) bool {
	var numberOfUsersWithTargetFullName = 0
	for _, user := range users {
		if user.GetFullName() == fullName {
			numberOfUsersWithTargetFullName += 1
		}
	}
	return numberOfUsersWithTargetFullName > 1
}

func orgHasMultipleUsersWithTargetEmail(users []v2.IamV2User, email string) bool {
	var numberOfUsersWithTargetEmail = 0
	for _, user := range users {
		if user.GetEmail() == email {
			numberOfUsersWithTargetEmail += 1
		}
	}
	return numberOfUsersWithTargetEmail > 1
}

func executeUserRead(ctx context.Context, c *Client, userId string) (v2.IamV2User, *http.Response, error) {
	req := c.iamClient.UsersIamV2Api.GetIamV2User(c.iamApiContext(ctx), userId)
	return req.Execute()
}
