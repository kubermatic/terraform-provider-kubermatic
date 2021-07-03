package kubermatic

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func dataSourceNodeDeployment() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceNodeDeploymentRead,
		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Reference project identifier",
			},
			"cluster_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Reference cluster identifier",
			},
			"dc_name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Data center name",
			},
			"id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "ID/Name of the node deployment",
			},
			// Computed
			"name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Name of the node deployment",
			},
			"creation_timestamp": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Creation timestamp of the node deployment",
			},
			"deletion_timestamp": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Deletion timestamp of the node deployment",
			},
			"spec": {
				Type:        schema.TypeList,
				Computed:    true,
				MaxItems:    1,
				Description: "Node deployment specification",
				Elem: &schema.Resource{
					Schema: nodeDeploymentSpecFields(),
				},
			},
		},
	}
}

func dataSourceNodeDeploymentRead(d *schema.ResourceData, m interface{}) error {
	nodeDeplID := d.Get("id").(string)
	d.SetId(nodeDeplID)

	return resourceNodeDeploymentRead(d, m)
}
