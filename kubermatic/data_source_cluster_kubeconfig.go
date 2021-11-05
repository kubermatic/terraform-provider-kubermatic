package kubermatic

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/kubermatic/go-kubermatic/client/project"
)

func dataSourceClusterKubeconfigV2() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceClusterKubeconfigV2Read,
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
			"kubeconfig": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Kubeconfig of the given cluster.",
			},
		},
	}
}

func dataSourceClusterKubeconfigV2Read(d *schema.ResourceData, meta interface{}) error {
	k := meta.(*kubermaticProviderMeta)
	p := project.NewGetClusterKubeconfigV2Params()

	clusterID := d.Get("cluster_id").(string)
	p.SetProjectID(d.Get("project_id").(string))
	p.SetClusterID(clusterID)

	r, err := k.client.Project.GetClusterKubeconfigV2(p, k.auth)
	if err != nil {
		return fmt.Errorf("unable to get cluster kubeconfig '%s': %s", clusterID, err)
	}

	// Set data variables
	d.Set("kubeconfig", string(r.Payload))

	d.SetId(clusterID)
	return nil
}
