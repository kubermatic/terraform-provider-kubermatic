package kubermatic

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/kubermatic/go-kubermatic/client/project"
)

func dataSourceProject() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceProjectRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Project name",
			},
			"id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Project id",
			},
			"labels": {
				Type:        schema.TypeMap,
				Computed:    true,
				Description: "Project labels",
				Elem:        schema.TypeString,
			},
			"user": {
				Type:        schema.TypeSet,
				Computed:    true,
				Description: "Project user",
				Elem:        projectUsersSchema(),
			},
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Status represents the current state of the project",
			},
			"creation_timestamp": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Creation timestamp",
			},
			"deletion_timestamp": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Deletion timestamp",
			},
		},
	}
}

func dataSourceProjectRead(d *schema.ResourceData, m interface{}) error {
	p_name := d.Get("name").(string)
	p_id := d.Get("id").(string)

	k := m.(*kubermaticProviderMeta)
	p := project.NewListProjectsParams()

	if p_id == "" {
		if p_name == "" {
			return fmt.Errorf("You must specify either an ID or a name.")
		} else {
			// get the project ID from the API
			r, err := k.client.Project.ListProjects(p, k.auth)
			if err != nil {
				return fmt.Errorf("error when getting projects list: %s", getErrorResponse(err))
			}
			for _, project := range r.Payload {
				if project.Name == p_name {
					p_id = project.ID
					break
				}
			}
			if p_id == "" {
				return fmt.Errorf("error: project not found in project list.")
			}

		}
	}
	d.SetId(p_id)

	return resourceProjectRead(d, m)
}
