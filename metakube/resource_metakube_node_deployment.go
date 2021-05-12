package metakube

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/syseleven/go-metakube/client/project"
	"github.com/syseleven/go-metakube/models"
)

func metakubeResourceNodeDeployment() *schema.Resource {
	return &schema.Resource{
		CreateContext: metakubeResourceNodeDeploymentCreate,
		ReadContext:   metakubeResourceNodeDeploymentRead,
		UpdateContext: metakubeResourceNodeDeploymentUpdate,
		DeleteContext: metakubeResourceNodeDeploymentDelete,
		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
				parts := strings.Split(d.Id(), ":")
				if len(parts) != 3 {
					return nil, fmt.Errorf("Please provide node deployment identifier in format 'project_id:cluster_id:node_deployment_name'")
				}
				d.Set("project_id", parts[0])
				d.Set("cluster_id", parts[1])
				d.SetId(parts[2])
				return []*schema.ResourceData{d}, nil
			},
		},
		CustomizeDiff: customdiff.All(
			validateNodeSpecMatchesCluster(),
			validateAutoscalerFields(),
		),

		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Project the cluster belongs to",
			},

			"cluster_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				// TODO: update descriptions
				Description: "Cluster that node deployment belongs to",
			},

			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "Node deployment name",
			},

			"spec": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Required:    true,
				Description: "Node deployment specification",
				Elem: &schema.Resource{
					Schema: matakubeResourceNodeDeploymentSpecFields(),
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

func readMachineDeploymentPreservedValues(d *schema.ResourceData) *nodeSpecPreservedValues {
	if v, ok := d.GetOk("spec.0.template.0.cloud.0.azure"); ok {
		if vv, ok := v.([]interface{}); ok && len(vv) == 1 {
			return &nodeSpecPreservedValues{
				azure: metakubeNodeDeploymentExpandAzureSpec(vv),
			}
		}
	}
	return &nodeSpecPreservedValues{}
}

func metakubeResourceNodeDeploymentCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	k := m.(*metakubeProviderMeta)
	clusterID := d.Get("cluster_id").(string)
	projectID := d.Get("project_id").(string)
	if projectID == "" {
		var err error
		projectID, err = metakubeResourceClusterFindProjectID(ctx, clusterID, k)
		if err != nil {
			return diag.FromErr(err)
		}
		if projectID == "" {
			k.log.Info("owner project for cluster '%s' is not found", clusterID)
			return diag.Errorf("")
		}
	}
	p := project.NewCreateMachineDeploymentParams().
		WithContext(ctx).
		WithProjectID(projectID).
		WithClusterID(clusterID).
		WithBody(&models.NodeDeployment{
			Name: d.Get("name").(string),
			Spec: metakubeNodeDeploymentExpandSpec(d.Get("spec").([]interface{})),
		})

	if err := metakubeResourceClusterWaitForReady(ctx, k, d, projectID, clusterID); err != nil {
		return diag.Errorf("cluster is not ready: %v", err)
	}

	// Some cloud providers, like AWS, take some time to finish initializing.
	err := resource.RetryContext(ctx, d.Timeout(schema.TimeoutCreate), func() *resource.RetryError {
		p := project.NewListMachineDeploymentsParams().
			WithContext(ctx).
			WithProjectID(projectID).
			WithClusterID(clusterID)

		_, err := k.client.Project.ListMachineDeployments(p, k.auth)
		if err != nil {
			if e, ok := err.(*project.ListMachineDeploymentsDefault); ok && e.Code() != http.StatusOK {
				return resource.RetryableError(fmt.Errorf("unable to list node deployments %v", err))
			}
			return resource.NonRetryableError(err)
		}

		return nil
	})
	if err != nil {
		return diag.Errorf("nodedeployments API is not ready: %v", err)
	}

	r, err := k.client.Project.CreateMachineDeployment(p, k.auth)
	if err != nil {
		return diag.Errorf("unable to create a node deployment: %v", stringifyResponseError(err))
	}
	d.SetId(r.Payload.ID)
	d.Set("project_id", projectID)

	if err := metakubeResourceNodeDeploymentWaitForReady(ctx, k, d.Timeout(schema.TimeoutCreate), projectID, clusterID, r.Payload.ID, 0); err != nil {
		return diag.FromErr(err)
	}

	return metakubeResourceNodeDeploymentRead(ctx, d, m)

}

func metakubeResourceNodeDeploymentRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	k := m.(*metakubeProviderMeta)
	projectID := d.Get("project_id").(string)
	clusterID := d.Get("cluster_id").(string)
	p := project.NewGetMachineDeploymentParams().
		WithContext(ctx).
		WithProjectID(projectID).
		WithClusterID(clusterID).
		WithMachineDeploymentID(d.Id())

	r, err := k.client.Project.GetMachineDeployment(p, k.auth)
	if err != nil {
		if e, ok := err.(*project.GetMachineDeploymentDefault); ok && e.Code() == http.StatusNotFound {
			k.log.Infof("removing node deployment '%s' from terraform state file, could not find the resource", d.Id())
			d.SetId("")
			return nil
		}
		if _, ok := err.(*project.GetMachineDeploymentForbidden); ok {
			k.log.Infof("removing node deployment '%s' from terraform state file, access forbidden", d.Id())
			d.SetId("")
			return nil
		}
		return diag.Errorf("unable to get node deployment '%s/%s/%s': %s", projectID, clusterID, d.Id(), stringifyResponseError(err))
	}

	_ = d.Set("name", r.Payload.Name)

	_ = d.Set("spec", metakubeNodeDeploymentFlattenSpec(readMachineDeploymentPreservedValues(d), r.Payload.Spec))

	_ = d.Set("creation_timestamp", r.Payload.CreationTimestamp.String())

	_ = d.Set("deletion_timestamp", r.Payload.DeletionTimestamp.String())

	return nil
}

func metakubeResourceNodeDeploymentUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	k := m.(*metakubeProviderMeta)
	projectID := d.Get("project_id").(string)
	clusterID := d.Get("cluster_id").(string)
	p := project.NewPatchMachineDeploymentParams()
	p.SetContext(ctx)
	p.SetProjectID(projectID)
	p.SetClusterID(clusterID)
	p.SetMachineDeploymentID(d.Id())
	p.SetPatch(&models.NodeDeployment{
		Spec: metakubeNodeDeploymentExpandSpec(d.Get("spec").([]interface{})),
	})

	res, err := k.client.Project.PatchMachineDeployment(p, k.auth)
	if err != nil {
		return diag.Errorf("unable to update a node deployment: %v", stringifyResponseError(err))
	}

	if err := metakubeResourceNodeDeploymentWaitForReady(ctx, k, d.Timeout(schema.TimeoutCreate), projectID, clusterID, d.Id(), res.Payload.Status.ObservedGeneration); err != nil {
		return diag.FromErr(err)
	}

	return metakubeResourceNodeDeploymentRead(ctx, d, m)
}

func metakubeResourceNodeDeploymentWaitForReady(ctx context.Context, k *metakubeProviderMeta, timeout time.Duration, projectID, clusterID, id string, generation int64) error {
	return resource.RetryContext(ctx, timeout, func() *resource.RetryError {
		p := project.NewGetMachineDeploymentParams().
			WithContext(ctx).
			WithProjectID(projectID).
			WithClusterID(clusterID).
			WithMachineDeploymentID(id)

		r, err := k.client.Project.GetMachineDeployment(p, k.auth)
		if err != nil {
			return resource.RetryableError(fmt.Errorf("unable to get node deployment %v", err))
		}

		if r.Payload.Status.ReadyReplicas < *r.Payload.Spec.Replicas || r.Payload.Status.ObservedGeneration <= generation || r.Payload.Status.UnavailableReplicas != 0 {
			k.log.Debugf("waiting for node deployment '%s' to be ready, %+v", id, r.Payload.Status)
			return resource.RetryableError(fmt.Errorf("waiting for node deployment '%s' to be ready", id))
		}
		return nil
	})
}

func metakubeResourceNodeDeploymentDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	k := m.(*metakubeProviderMeta)
	projectID := d.Get("project_id").(string)
	clusterID := d.Get("cluster_id").(string)
	p := project.NewDeleteMachineDeploymentParams().
		WithProjectID(projectID).
		WithClusterID(clusterID).
		WithMachineDeploymentID(d.Id())

	_, err := k.client.Project.DeleteMachineDeployment(p, k.auth)
	if err != nil {
		if e, ok := err.(*project.DeleteMachineDeploymentDefault); ok && e.Code() == http.StatusNotFound {
			k.log.Infof("removing node deployment '%s' from terraform state file, could not find the resource", d.Id())
			d.SetId("")
			return nil
		}
		return diag.Errorf("unable to delete node deployment '%s': %s", d.Id(), stringifyResponseError(err))
	}

	err = resource.RetryContext(ctx, d.Timeout(schema.TimeoutDelete), func() *resource.RetryError {
		p := project.NewGetMachineDeploymentParams().
			WithContext(ctx).
			WithProjectID(projectID).
			WithClusterID(clusterID).
			WithMachineDeploymentID(d.Id())

		r, err := k.client.Project.GetMachineDeployment(p, k.auth)
		if err != nil {
			if e, ok := err.(*project.GetMachineDeploymentDefault); ok && e.Code() == http.StatusNotFound {
				k.log.Debugf("node deployment '%s' has been destroyed, returned http code: %d", d.Id(), e.Code())
				d.SetId("")
				return nil
			}
			return resource.NonRetryableError(fmt.Errorf("unable to get node deployment '%s': %s", d.Id(), stringifyResponseError(err)))
		}

		k.log.Debugf("node deployment '%s' deletion in progress, deletionTimestamp: %s",
			d.Id(), r.Payload.DeletionTimestamp.String())
		return resource.RetryableError(fmt.Errorf("node deployment '%s' deletion in progress", d.Id()))
	})
	if err != nil {
		return diag.FromErr(err)
	}
	return nil
}
