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
	"net/http"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	v1 "github.com/confluentinc/ccloud-sdk-go-v2-internal/vim/v1"
)

const (
	paramManufacturer    = "manufacturer"
	paramModel           = "model"
	paramWebhookEndpoint = "webhook_endpoint"
)

const (
	carLoggingKey = "car_id"
)

func resourceVimCar() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceVimCarCreate,
		ReadContext:   resourceVimCarRead,
		UpdateContext: resourceVimCarUpdate,
		DeleteContext: resourceVimCarDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceVimCarImport,
		},
		Schema: map[string]*schema.Schema{
			paramManufacturer: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The manufacturer of the car.",
			},
			paramModel: {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "The model name of the car (e.g., 'Camry', 'F-150').",
			},
			paramWebhookEndpoint: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "A publicly accessible URL where car status updates (e.g., 'maintenance_required', 'low_fuel') will be sent via POST requests.",
			},
		},
	}
}

func resourceVimCarCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client)

	manufacturer := d.Get(paramManufacturer).(string)

	createCarRequest := v1.NewVimV1Car()
	createCarRequest.SetManufacturer(manufacturer)

	if model, ok := d.GetOk(paramModel); ok {
		createCarRequest.SetModel(model.(string))
	}

	createCarRequestJson, err := json.Marshal(createCarRequest)
	if err != nil {
		return diag.Errorf("error creating Car: error marshaling %#v to json: %s", createCarRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Creating new Car: %s", createCarRequestJson))

	createdCar, resp, err := executeVimCarCreate(c.vimApiContext(ctx), c, createCarRequest)
	if err != nil {
		return diag.Errorf("error creating Car: %s", createDescriptiveError(err, resp))
	}

	d.SetId(createdCar.GetId())

	createdCarJson, err := json.Marshal(createdCar)
	if err != nil {
		return diag.Errorf("error creating Car %q: error marshaling %#v to json: %s", d.Id(), createdCar, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished creating Car %q: %s", d.Id(), createdCarJson), map[string]interface{}{carLoggingKey: d.Id()})

	return resourceVimCarRead(ctx, d, meta)
}

func executeVimCarCreate(ctx context.Context, c *Client, car *v1.VimV1Car) (v1.VimV1Car, *http.Response, error) {
	req := c.vimClient.CarsVimV1Api.CreateVimV1Car(c.vimApiContext(ctx)).VimV1Car(*car)
	return req.Execute()
}

func resourceVimCarRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Car %q", d.Id()), map[string]interface{}{carLoggingKey: d.Id()})

	c := meta.(*Client)
	car, resp, err := executeVimCarRead(c.vimApiContext(ctx), c, d.Id())
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Car %q: %s", d.Id(), createDescriptiveError(err, resp)), map[string]interface{}{carLoggingKey: d.Id()})

		isResourceNotFound := isNonKafkaRestApiResourceNotFound(resp)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing Car %q in TF state because Car could not be found on the server", d.Id()), map[string]interface{}{carLoggingKey: d.Id()})
			d.SetId("")
			return nil
		}

		return diag.FromErr(createDescriptiveError(err, resp))
	}

	carJson, err := json.Marshal(car)
	if err != nil {
		return diag.Errorf("error reading Car %q: error marshaling %#v to json: %s", d.Id(), car, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Car %q: %s", d.Id(), carJson), map[string]interface{}{carLoggingKey: d.Id()})

	if _, err := setCarAttributes(d, car); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Car %q", d.Id()), map[string]interface{}{carLoggingKey: d.Id()})

	return nil
}

func executeVimCarRead(ctx context.Context, c *Client, carId string) (v1.VimV1Car, *http.Response, error) {
	req := c.vimClient.CarsVimV1Api.GetVimV1Car(c.vimApiContext(ctx), carId)
	return req.Execute()
}

func setCarAttributes(d *schema.ResourceData, car v1.VimV1Car) (*schema.ResourceData, error) {
	if err := d.Set(paramManufacturer, car.GetManufacturer()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramModel, car.GetModel()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramWebhookEndpoint, car.GetWebhookEndpoint()); err != nil {
		return nil, createDescriptiveError(err)
	}
	d.SetId(car.GetId())
	return d, nil
}

func resourceVimCarUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChangesExcept(paramManufacturer) {
		return diag.Errorf("error updating Car %q: only %q attribute can be updated for Car", d.Id(), paramManufacturer)
	}

	updateCarRequest := v1.NewVimV1CarUpdate()

	if d.HasChange(paramManufacturer) {
		manufacturer := d.Get(paramManufacturer).(string)
		updateCarRequest.SetManufacturer(manufacturer)
	}

	updateCarRequestJson, err := json.Marshal(updateCarRequest)
	if err != nil {
		return diag.Errorf("error updating Car %q: error marshaling %#v to json: %s", d.Id(), updateCarRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Updating Car %q: %s", d.Id(), updateCarRequestJson), map[string]interface{}{carLoggingKey: d.Id()})

	c := meta.(*Client)
	updatedCar, resp, err := c.vimClient.CarsVimV1Api.UpdateVimV1Car(c.vimApiContext(ctx), d.Id()).VimV1CarUpdate(*updateCarRequest).Execute()
	if err != nil {
		return diag.Errorf("error updating Car %q: %s", d.Id(), createDescriptiveError(err, resp))
	}

	updatedCarJson, err := json.Marshal(updatedCar)
	if err != nil {
		return diag.Errorf("error updating Car %q: error marshaling %#v to json: %s", d.Id(), updatedCar, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished updating Car %q: %s", d.Id(), updatedCarJson), map[string]interface{}{carLoggingKey: d.Id()})

	return resourceVimCarRead(ctx, d, meta)
}

func resourceVimCarDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Deleting Car %q", d.Id()), map[string]interface{}{carLoggingKey: d.Id()})

	c := meta.(*Client)
	req := c.vimClient.CarsVimV1Api.DeleteVimV1Car(c.vimApiContext(ctx), d.Id())
	resp, err := req.Execute()

	if err != nil {
		return diag.Errorf("error deleting Car %q: %s", d.Id(), createDescriptiveError(err, resp))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting Car %q", d.Id()), map[string]interface{}{carLoggingKey: d.Id()})

	return nil
}

func resourceVimCarImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing Car %q", d.Id()), map[string]interface{}{carLoggingKey: d.Id()})

	d.MarkNewResource()
	if diagnostics := resourceVimCarRead(ctx, d, meta); diagnostics != nil {
		return nil, fmt.Errorf("error importing Car %q: %s", d.Id(), diagnostics[0].Summary)
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished importing Car %q", d.Id()), map[string]interface{}{carLoggingKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}
