package kubermatic

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/kubermatic/go-kubermatic/client/project"
)

func dataSourceSSHKey() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceSSHKeyRead,
		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.NoZeroValues,
			},
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.NoZeroValues,
			},
			"public_key": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceSSHKeyRead(d *schema.ResourceData, m interface{}) error {
	ssh_name := d.Get("name").(string)
	project_id := d.Get("project_id").(string)

	k := m.(*kubermaticProviderMeta)
	p := project.NewListSSHKeysParams()
	p.SetProjectID(project_id)

	r, err := k.client.Project.ListSSHKeys(p, k.auth)
	if err != nil {
		return fmt.Errorf("error retrieving SSH key list for the given project. Error: %s", getErrorResponse(err))
	}
	for _, key := range r.Payload {
		if key.Name == ssh_name {
			d.Set("public_key", key.Spec.PublicKey)
			d.SetId(key.ID)
			break
		}
	}
	if d.Get("public_key").(string) == "" {
		return fmt.Errorf("error: no SSH key with the given name found.")
	}
	return nil
}
