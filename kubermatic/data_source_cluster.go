package kubermatic

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/kubermatic/go-kubermatic/client/project"
	"github.com/kubermatic/go-kubermatic/models"
)

func dataSourceCluster() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceClusterRead,
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
				Computed:    true,
				Description: "Data center name",
			},
			"name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Cluster name",
			},
			"labels": {
				Type:        schema.TypeMap,
				Computed:    true,
				Description: "Labels added to cluster",
			},
			"spec": {
				Type:        schema.TypeList,
				Computed:    true,
				MaxItems:    1,
				Description: "Cluster specification",
				Elem: &schema.Resource{
					Schema: clusterSpecFields(),
				},
			},
			"credential": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Cluster access credential",
			},
			"type": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Cloud orchestrator, either Kubernetes or OpenShift",
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

func dataSourceClusterRead(d *schema.ResourceData, meta interface{}) error {
	k := meta.(*kubermaticProviderMeta)
	p := project.NewGetClusterV2Params()

	projectID := d.Get("project_id").(string)
	clusterID := d.Get("cluster_id").(string)
	p.SetProjectID(projectID)
	p.SetClusterID(clusterID)

	r, err := k.client.Project.GetClusterV2(p, k.auth)
	if err != nil {
		return fmt.Errorf("unable to get cluster '%s': %s", clusterID, err)
	}

	// Set data variables
	d.Set("project_id", projectID)
	d.Set("cluster_id", clusterID)
	d.Set("name", r.Payload.Name)
	d.Set("type", r.Payload.Type)
	d.Set("dc_name", r.Payload.Spec.Cloud.DatacenterName)

	d.Set("creation_timestamp", r.Payload.CreationTimestamp.String())
	d.Set("deletion_timestamp", r.Payload.DeletionTimestamp.String())
	k.log.Info("ProjectID: ", d.Get("project_id"), "ClusterID: ", d.Get("cluster_id"))
	err = d.Set("credential", r.Payload.Credential)
	if err != nil {
		return err
	}

	labels, err := excludeProjectLabels(k, projectID, r.Payload.Labels)
	if err != nil {
		return err
	}
	if err := d.Set("labels", labels); err != nil {
		return err
	}
	values := clusterPreserveValues{
		openstack: &clusterOpenstackPreservedValues{},
		azure:     &models.AzureCloudSpec{},
		aws:       &models.AWSCloudSpec{},
	}
	specFlattenned := flattenClusterSpec(values, r.Payload.Spec)
	if err = d.Set("spec", specFlattenned); err != nil {
		return err
	}

	d.SetId(clusterID)
	return nil
}
