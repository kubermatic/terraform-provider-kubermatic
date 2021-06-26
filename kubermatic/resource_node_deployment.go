package kubermatic

import (
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/kubermatic/go-kubermatic/client/project"
	"github.com/kubermatic/go-kubermatic/models"
)

func resourceNodeDeployment() *schema.Resource {
	return &schema.Resource{
		Create: resourceNodeDeploymentCreate,
		Read:   resourceNodeDeploymentRead,
		Update: resourceNodeDeploymentUpdate,
		Delete: resourceNodeDeploymentDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		CustomizeDiff: validateNodeSpecMatchesCluster(),

		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Reference project identifier",
			},
			"dc_name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Data center name",
			},
			"cluster_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Cluster identifier",
			},
			"name": {
				Type: schema.TypeString,
				// TODO(furkhat): make field "Computed: true" when back end error is fixed.
				Required:    true,
				ForceNew:    true,
				Description: "Node deployment name",
			},
			"spec": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Required:    true,
				Description: "Node deployment specification",
				Elem: &schema.Resource{
					Schema: nodeDeploymentSpecFields(),
				},
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

type nodeSpecPreservedValues struct {
	// API returns empty spec for Azure clusters, so we just preserve values used for creation
	azure *models.AzureNodeSpec
}

func readNodeDeploymentPreservedValues(d *schema.ResourceData) *nodeSpecPreservedValues {
	if _, ok := d.GetOkExists("spec.0.template.0.cloud.0.azure.0"); !ok {
		return &nodeSpecPreservedValues{}
	}
	return &nodeSpecPreservedValues{
		azure: expandAzureNodeSpec(d.Get("spec.0.template.0.cloud.0.azure").([]interface{})),
	}
}

func resourceNodeDeploymentCreate(d *schema.ResourceData, m interface{}) error {
	projectID := d.Get("project_id").(string)
	dc_name := d.Get("dc_name").(string)
	clusterID := d.Get("cluster_id").(string)

	k := m.(*kubermaticProviderMeta)
	dc, err := getDatacenterByName(k, dc_name)
	if err != nil {
		return err
	}

	p := project.NewCreateNodeDeploymentParams()
	p.SetProjectID(projectID)
	p.SetDC(dc.Spec.Seed)
	p.SetClusterID(clusterID)
	p.SetBody(&models.NodeDeployment{
		Name: d.Get("name").(string),
		Spec: expandNodeDeploymentSpec(d.Get("spec").([]interface{})),
	})

	if err := waitClusterReady(k, d, projectID, dc.Spec.Seed, clusterID); err != nil {
		return fmt.Errorf("cluster is not ready: %v", err)
	}

	r, err := k.client.Project.CreateNodeDeployment(p, k.auth)
	if err != nil {
		if e, ok := err.(*project.CreateNodeDeploymentDefault); ok && errorMessage(e.Payload) != "" {
			return fmt.Errorf("unable to create node deployment: %s", errorMessage(e.Payload))
		}

		return fmt.Errorf("unable to create a node deployment: %v", getErrorResponse(err))
	}
	d.SetId(r.Payload.ID)

	if err := waitForNodeDeploymentRead(k, d.Timeout(schema.TimeoutCreate), projectID, dc.Spec.Seed, clusterID, r.Payload.ID); err != nil {
		return err
	}

	return resourceNodeDeploymentRead(d, m)

}

func resourceNodeDeploymentRead(d *schema.ResourceData, m interface{}) error {
	k := m.(*kubermaticProviderMeta)
	projectID := d.Get("project_id").(string)
	dc_name := d.Get("dc_name").(string)
	dc, err := getDatacenterByName(k, dc_name)
	if err != nil {
		return err
	}
	clusterID := d.Get("cluster_id").(string)
	nodeDeplID := d.Id()

	p := project.NewGetNodeDeploymentParams()
	p.SetProjectID(projectID)
	p.SetDC(dc.Spec.Seed)
	p.SetClusterID(clusterID)
	p.SetNodeDeploymentID(nodeDeplID)

	r, err := k.client.Project.GetNodeDeployment(p, k.auth)
	if err != nil {
		if e, ok := err.(*project.GetNodeDeploymentDefault); ok && e.Code() == http.StatusNotFound {
			k.log.Infof("removing node deployment '%s' from terraform state file, could not find the resource", d.Id())
			d.SetId("")
			return nil
		}
		if _, ok := err.(*project.GetNodeDeploymentForbidden); ok {
			k.log.Infof("removing node deployment '%s' from terraform state file, access forbidden", d.Id())
			d.SetId("")
			return nil
		}
		return fmt.Errorf("unable to get node deployment '%s': %s", d.Id(), getErrorResponse(err))
	}

	d.Set("cluster_id", clusterID)

	d.Set("project_id", projectID)

	d.Set("name", r.Payload.Name)

	d.Set("spec", flattenNodeDeploymentSpec(readNodeDeploymentPreservedValues(d), r.Payload.Spec))

	d.Set("creation_timestamp", r.Payload.CreationTimestamp.String())

	d.Set("deletion_timestamp", r.Payload.DeletionTimestamp.String())

	return nil
}

func resourceNodeDeploymentUpdate(d *schema.ResourceData, m interface{}) error {
	// TODO(furkhat): uncomment and adjust when client is fixed.
	k := m.(*kubermaticProviderMeta)

	projectID := d.Get("project_id").(string)
	dc_name := d.Get("dc_name").(string)
	clusterID := d.Get("cluster_id").(string)
	nodeDeplID := d.Id()
	dc, err := getDatacenterByName(k, dc_name)
	if err != nil {
		return err
	}
	p := project.NewPatchNodeDeploymentParams()
	p.SetProjectID(projectID)
	p.SetDC(dc.Spec.Seed)
	p.SetClusterID(clusterID)
	p.SetNodeDeploymentID(nodeDeplID)
	p.SetPatch(models.NodeDeployment{
		Spec: expandNodeDeploymentSpec(d.Get("spec").([]interface{})),
	})

	r, err := k.client.Project.PatchNodeDeployment(p, k.auth)
	if err != nil {
		if e, ok := err.(*project.PatchNodeDeploymentDefault); ok && errorMessage(e.Payload) != "" {
			return fmt.Errorf("unable to update a node deployment: %v", errorMessage(e.Payload))
		}
		return fmt.Errorf("unable to update a node deployment: %v", getErrorResponse(err))
	}

	if err := waitForNodeDeploymentRead(k, d.Timeout(schema.TimeoutCreate), projectID, dc.Spec.Seed, clusterID, r.Payload.ID); err != nil {
		return err
	}

	return resourceNodeDeploymentRead(d, m)
}

func waitForNodeDeploymentRead(k *kubermaticProviderMeta, timeout time.Duration, projectID, seedDC, clusterID, id string) error {
	return resource.Retry(timeout, func() *resource.RetryError {
		p := project.NewGetNodeDeploymentParams()
		p.SetProjectID(projectID)
		p.SetClusterID(clusterID)
		p.SetDC(seedDC)
		p.SetNodeDeploymentID(id)

		r, err := k.client.Project.GetNodeDeployment(p, k.auth)
		if err != nil {
			return resource.RetryableError(fmt.Errorf("unable to get node deployment %v", err))
		}

		if r.Payload.Status.ReadyReplicas < *r.Payload.Spec.Replicas {
			k.log.Debugf("waiting for node deployment '%s' to be ready, %+v", id, r.Payload.Status)
			return resource.RetryableError(fmt.Errorf("waiting for node deployment '%s' to be ready", id))
		}
		return nil
	})
}

func resourceNodeDeploymentDelete(d *schema.ResourceData, m interface{}) error {
	k := m.(*kubermaticProviderMeta)
	projectID := d.Get("project_id").(string)
	dc_name := d.Get("dc_name").(string)
	clusterID := d.Get("cluster_id").(string)
	nodeDeplID := d.Id()
	dc, err := getDatacenterByName(k, dc_name)
	if err != nil {
		return err
	}
	p := project.NewDeleteNodeDeploymentParams()
	p.SetProjectID(projectID)
	p.SetDC(dc.Spec.Seed)
	p.SetClusterID(clusterID)
	p.SetNodeDeploymentID(nodeDeplID)

	_, err = k.client.Project.DeleteNodeDeployment(p, k.auth)
	if err != nil {
		// TODO: check if not found
		return fmt.Errorf("unable to delete node deployment '%s': %s", d.Id(), getErrorResponse(err))
	}

	return resource.Retry(d.Timeout(schema.TimeoutDelete), func() *resource.RetryError {
		p := project.NewGetNodeDeploymentParams()
		p.SetProjectID(projectID)
		p.SetDC(dc.Spec.Seed)
		p.SetClusterID(clusterID)
		p.SetNodeDeploymentID(nodeDeplID)

		r, err := k.client.Project.GetNodeDeployment(p, k.auth)
		if err != nil {
			if e, ok := err.(*project.GetNodeDeploymentDefault); ok && e.Code() == http.StatusNotFound {
				k.log.Debugf("node deployment '%s' has been destroyed, returned http code: %d", d.Id(), e.Code())
				d.SetId("")
				return nil
			}
			return resource.NonRetryableError(fmt.Errorf("unable to get node deployment '%s': %s", d.Id(), getErrorResponse(err)))
		}

		k.log.Debugf("node deployment '%s' deletion in progress, deletionTimestamp: %s",
			d.Id(), r.Payload.DeletionTimestamp.String())
		return resource.RetryableError(fmt.Errorf("node deployment '%s' deletion in progress", d.Id()))
	})
}
