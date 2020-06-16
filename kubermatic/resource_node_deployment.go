package kubermatic

import (
	"fmt"
	"net/http"
	"strings"
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

		Schema: map[string]*schema.Schema{
			"cluster_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Reference full cluster identifier of format <project id>:<seed dc>:<cluster id>",
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
	projectID, seedDC, clusterID, err := kubermaticClusterParseID(d.Get("cluster_id").(string))
	if err != nil {
		return err
	}

	k := m.(*kubermaticProviderMeta)

	p := project.NewCreateNodeDeploymentParams()
	p.SetProjectID(projectID)
	p.SetDC(seedDC)
	p.SetClusterID(clusterID)
	p.SetBody(&models.NodeDeployment{
		Name: d.Get("name").(string),
		Spec: expandNodeDeploymentSpec(d.Get("spec").([]interface{})),
	})

	r, err := k.client.Project.CreateNodeDeployment(p, k.auth)
	if err != nil {
		if e, ok := err.(*project.CreateNodeDeploymentDefault); ok && errorMessage(e.Payload) != "" {
			return fmt.Errorf("%s: %v", errorMessage(e.Payload), err)
		}

		return fmt.Errorf("unable to create a node deployment: %v", getErrorResponse(err))
	}
	d.SetId(kubermaticNodeDeploymentMakeID(projectID, seedDC, clusterID, r.Payload.ID))

	if err := waitForNodeDeploymentRead(k, d.Timeout(schema.TimeoutCreate), projectID, seedDC, clusterID, r.Payload.ID); err != nil {
		return err
	}

	return resourceNodeDeploymentRead(d, m)
}

func kubermaticNodeDeploymentMakeID(projectID, seedDC, clusterID, id string) string {
	return fmt.Sprintf("%s:%s:%s:%s", projectID, seedDC, clusterID, id)
}

func kubermaticNodeDeploymentParseID(id string) (string, string, string, string, error) {
	parts := strings.SplitN(id, ":", 4)

	if len(parts) != 4 || parts[0] == "" || parts[1] == "" || parts[2] == "" || parts[3] == "" {
		return "", "", "", "", fmt.Errorf("unexpected format of ID (%s), expected project_id:seed_dc:cluster_id:id", id)
	}

	return parts[0], parts[1], parts[2], parts[3], nil
}

func resourceNodeDeploymentRead(d *schema.ResourceData, m interface{}) error {
	k := m.(*kubermaticProviderMeta)
	projectID, seedDC, clusterID, nodeDeplID, err := kubermaticNodeDeploymentParseID(d.Id())
	if err != nil {
		return err
	}
	p := project.NewGetNodeDeploymentParams()
	p.SetProjectID(projectID)
	p.SetDC(seedDC)
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

	d.Set("cluster_id", kubermaticClusterMakeID(projectID, seedDC, clusterID))

	d.Set("name", r.Payload.Name)

	d.Set("spec", flattenNodeDeploymentSpec(readNodeDeploymentPreservedValues(d), r.Payload.Spec))

	d.Set("creation_timestamp", r.Payload.CreationTimestamp.String())

	d.Set("deletion_timestamp", r.Payload.DeletionTimestamp.String())

	return nil
}

func resourceNodeDeploymentUpdate(d *schema.ResourceData, m interface{}) error {
	// TODO(furkhat): uncomment and adjust when client is fixed.
	k := m.(*kubermaticProviderMeta)
	projectID, seedDC, clusterID, nodeDeplID, err := kubermaticNodeDeploymentParseID(d.Id())
	if err != nil {
		return err
	}
	p := project.NewPatchNodeDeploymentParams()
	p.SetProjectID(projectID)
	p.SetDC(seedDC)
	p.SetClusterID(clusterID)
	p.SetNodeDeploymentID(nodeDeplID)
	p.SetPatch(models.NodeDeployment{
		Spec: expandNodeDeploymentSpec(d.Get("spec").([]interface{})),
	})

	r, err := k.client.Project.PatchNodeDeployment(p, k.auth)
	if err != nil {
		if e, ok := err.(*project.PatchNodeDeploymentDefault); ok && errorMessage(e.Payload) != "" {
			return fmt.Errorf(errorMessage(e.Payload))
		}
		return fmt.Errorf("unable to update a node deployment: %v", err)
	}

	if err := waitForNodeDeploymentRead(k, d.Timeout(schema.TimeoutCreate), projectID, seedDC, clusterID, r.Payload.ID); err != nil {
		return err
	}

	return resourceNodeDeploymentRead(d, m)
}

func waitForNodeDeploymentRead(k *kubermaticProviderMeta, timeout time.Duration, projectID, seedDC, clusterID, id string) error {
	err := resource.Retry(timeout, func() *resource.RetryError {
		p := project.NewGetNodeDeploymentParams()
		p.SetProjectID(projectID)
		p.SetClusterID(clusterID)
		p.SetDC(seedDC)
		p.SetNodeDeploymentID(id)

		r, err := k.client.Project.GetNodeDeployment(p, k.auth)
		if err != nil {
			if e, ok := err.(*project.GetNodeDeploymentDefault); ok && errorMessage(e.Payload) != "" {
				// Sometimes api returns 500 which often means try later
				return resource.RetryableError(fmt.Errorf("unable to get node deployment '%s' status: %s: %v", id, errorMessage(e.Payload), err))
			}
			return resource.NonRetryableError(fmt.Errorf("unable to get node deployment '%s' status: %v", id, getErrorResponse(err)))
		}

		if r.Payload.Status.ReadyReplicas < *r.Payload.Spec.Replicas {
			k.log.Debugf("waiting for node deployment '%s' to be ready, %+v", id, r.Payload.Status)
			return resource.RetryableError(fmt.Errorf("waiting for node deployment '%s' to be ready", id))
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("node deployment '%s' is not ready: %v", id, err)
	}
	return nil
}

func resourceNodeDeploymentDelete(d *schema.ResourceData, m interface{}) error {
	k := m.(*kubermaticProviderMeta)
	projectID, seedDC, clusterID, nodeDeplID, err := kubermaticNodeDeploymentParseID(d.Id())
	if err != nil {
		return err
	}
	p := project.NewDeleteNodeDeploymentParams()
	p.SetProjectID(projectID)
	p.SetDC(seedDC)
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
		p.SetDC(seedDC)
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
