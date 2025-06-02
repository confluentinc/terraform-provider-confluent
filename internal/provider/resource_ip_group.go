package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// ipGroupResource returns the schema.Resource for confluent_ip_group.
func ipGroupResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceIPGroupCreate,
		ReadContext:   resourceIPGroupRead,
		UpdateContext: resourceIPGroupUpdate,
		DeleteContext: resourceIPGroupDelete,
		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the IP group.",
			},
			"group_name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "A human readable name for an IP Group.",
			},
			"cidr_blocks": {
				Type:        schema.TypeSet,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Required:    true,
				Description: "A set of CIDR blocks to include in the IP group.",
			},
		},
	}
}

func resourceIPGroupCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// TODO: Implement create logic.
	return nil
}

func resourceIPGroupRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// TODO: Implement read logic.
	return nil
}

func resourceIPGroupUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// TODO: Implement update logic.
	return nil
}

func resourceIPGroupDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// TODO: Implement delete logic.
	return nil
}
