package metakube

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/syseleven/terraform-provider-metakube/go-metakube/client/project"
	"github.com/syseleven/terraform-provider-metakube/go-metakube/models"
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
		CustomizeDiff: customdiff.All(
			validateNodeSpecMatchesCluster(),
			validateAutoscalerFields(),
		),

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
	if _, ok := d.GetOk("spec.0.template.0.cloud.0.azure.0"); !ok {
		return &nodeSpecPreservedValues{}
	}
	return &nodeSpecPreservedValues{
		azure: expandAzureNodeSpec(d.Get("spec.0.template.0.cloud.0.azure").([]interface{})),
	}
}

func resourceNodeDeploymentCreate(d *schema.ResourceData, m interface{}) error {

	projectID, seedDC, clusterID, err := metakubeClusterParseID(d.Get("cluster_id").(string))
	if err != nil {
		return err
	}
	k := m.(*metakubeProviderMeta)

	p := project.NewCreateNodeDeploymentParams()
	p.SetProjectID(projectID)
	p.SetDC(seedDC)
	p.SetClusterID(clusterID)
	p.SetBody(&models.NodeDeployment{
		Name: d.Get("name").(string),
		Spec: expandNodeDeploymentSpec(d.Get("spec").([]interface{})),
	})

	if err := waitClusterReady(k, d, projectID, seedDC, clusterID); err != nil {
		return fmt.Errorf("cluster is not ready: %v", err)
	}

	r, err := k.client.Project.CreateNodeDeployment(p, k.auth)
	if err != nil {
		if e, ok := err.(*project.CreateNodeDeploymentDefault); ok && errorMessage(e.Payload) != "" {
			return fmt.Errorf("unable to create node deployment: %s", errorMessage(e.Payload))
		}

		return fmt.Errorf("unable to create a node deployment: %v", getErrorResponse(err))
	}
	d.SetId(metakubeNodeDeploymentMakeID(projectID, seedDC, clusterID, r.Payload.ID))

	if err := waitForNodeDeploymentRead(k, d.Timeout(schema.TimeoutCreate), projectID, seedDC, clusterID, r.Payload.ID); err != nil {
		return err
	}

	return resourceNodeDeploymentRead(d, m)

}

func metakubeNodeDeploymentMakeID(projectID, seedDC, clusterID, id string) string {
	return fmt.Sprintf("%s:%s:%s:%s", projectID, seedDC, clusterID, id)
}

func metakubeNodeDeploymentParseID(id string) (string, string, string, string, error) {
	parts := strings.SplitN(id, ":", 4)

	if len(parts) != 4 || parts[0] == "" || parts[1] == "" || parts[2] == "" || parts[3] == "" {
		return "", "", "", "", fmt.Errorf("unexpected format of ID (%s), expected project_id:seed_dc:cluster_id:id", id)
	}

	return parts[0], parts[1], parts[2], parts[3], nil
}

func resourceNodeDeploymentRead(d *schema.ResourceData, m interface{}) error {
	k := m.(*metakubeProviderMeta)
	projectID, seedDC, clusterID, nodeDeploymentID, err := metakubeNodeDeploymentParseID(d.Id())
	if err != nil {
		return err
	}
	p := project.NewGetNodeDeploymentParams()
	p.SetProjectID(projectID)
	p.SetDC(seedDC)
	p.SetClusterID(clusterID)
	p.SetNodeDeploymentID(nodeDeploymentID)

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

	_ = d.Set("cluster_id", metakubeClusterMakeID(projectID, seedDC, clusterID))

	_ = d.Set("name", r.Payload.Name)

	_ = d.Set("spec", flattenNodeDeploymentSpec(readNodeDeploymentPreservedValues(d), r.Payload.Spec))

	_ = d.Set("creation_timestamp", r.Payload.CreationTimestamp.String())

	_ = d.Set("deletion_timestamp", r.Payload.DeletionTimestamp.String())

	return nil
}

func resourceNodeDeploymentUpdate(d *schema.ResourceData, m interface{}) error {
	k := m.(*metakubeProviderMeta)
	projectID, seedDC, clusterID, nodeDeploymentID, err := metakubeNodeDeploymentParseID(d.Id())
	if err != nil {
		return err
	}
	p := project.NewPatchNodeDeploymentParams()
	p.SetProjectID(projectID)
	p.SetDC(seedDC)
	p.SetClusterID(clusterID)
	p.SetNodeDeploymentID(nodeDeploymentID)
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

	if err := waitForNodeDeploymentRead(k, d.Timeout(schema.TimeoutCreate), projectID, seedDC, clusterID, r.Payload.ID); err != nil {
		return err
	}

	return resourceNodeDeploymentRead(d, m)
}

func waitForNodeDeploymentRead(k *metakubeProviderMeta, timeout time.Duration, projectID, seedDC, clusterID, id string) error {
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
	k := m.(*metakubeProviderMeta)
	projectID, seedDC, clusterID, nodeDeploymentID, err := metakubeNodeDeploymentParseID(d.Id())
	if err != nil {
		return err
	}
	p := project.NewDeleteNodeDeploymentParams()
	p.SetProjectID(projectID)
	p.SetDC(seedDC)
	p.SetClusterID(clusterID)
	p.SetNodeDeploymentID(nodeDeploymentID)

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
		p.SetNodeDeploymentID(nodeDeploymentID)

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
