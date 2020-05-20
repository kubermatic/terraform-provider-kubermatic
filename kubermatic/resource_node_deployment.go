package kubermatic

import (
	"fmt"
	"net/http"

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

		Schema: map[string]*schema.Schema{
			"dc": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Data center name",
			},
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
			"name": {
				Type:        schema.TypeString,
				Required:    true,
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

func resourceNodeDeploymentCreate(d *schema.ResourceData, m interface{}) error {
	k := m.(*kubermaticProviderMeta)
	dc := d.Get("dc").(string)
	pID := d.Get("project_id").(string)
	cID := d.Get("cluster_id").(string)
	p := project.NewCreateNodeDeploymentParams()

	p.SetProjectID(pID)
	p.SetClusterID(cID)
	p.SetDC(dc)
	p.SetBody(&models.NodeDeployment{
		Name: d.Get("name").(string),
		Spec: expandNodeDeploymentSpec(d.Get("spec").([]interface{})),
	})

	r, err := k.client.Project.CreateNodeDeployment(p, k.auth)
	if err != nil {
		return fmt.Errorf("unable to create a node deployment: %v", err)
	}
	nID := r.Payload.ID

	err = resource.Retry(d.Timeout(schema.TimeoutCreate), func() *resource.RetryError {
		p := project.NewGetNodeDeploymentParams()
		p.SetProjectID(pID)
		p.SetClusterID(cID)
		p.SetDC(dc)
		p.SetNodeDeploymentID(nID)

		r, err := k.client.Project.GetNodeDeployment(p, k.auth)
		if err != nil {
			return resource.NonRetryableError(fmt.Errorf("unable to get node deployment '%s' status: %v", nID, err))
		}

		if r.Payload.Status.ReadyReplicas < *r.Payload.Spec.Replicas {
			k.log.Debugf("waiting for node deployment '%s' to be ready, %+v", nID, r.Payload.Status)
			return resource.RetryableError(fmt.Errorf("waiting for node deployment '%s' to be ready", nID))
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("node deployment '%s' is not ready: %v", nID, err)
	}

	d.SetId(nID)
	return resourceNodeDeploymentRead(d, m)
}

func resourceNodeDeploymentRead(d *schema.ResourceData, m interface{}) error {
	k := m.(*kubermaticProviderMeta)
	p := project.NewGetNodeDeploymentParams()

	p.SetDC(d.Get("dc").(string))
	p.SetProjectID(d.Get("project_id").(string))
	p.SetClusterID(d.Get("cluster_id").(string))
	p.SetNodeDeploymentID(d.Id())

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
		return fmt.Errorf("unable to get node deployment '%s': %v", d.Id(), err)
	}

	err = d.Set("name", r.Payload.Name)
	if err != nil {
		return err
	}

	err = d.Set("spec", flattenNodeDeploymentSpec(r.Payload.Spec))
	if err != nil {
		return err
	}

	err = d.Set("creation_timestamp", r.Payload.CreationTimestamp.String())
	if err != nil {
		return err
	}

	err = d.Set("deletion_timestamp", r.Payload.DeletionTimestamp.String())
	if err != nil {
		return err
	}

	return nil
}

func resourceNodeDeploymentUpdate(d *schema.ResourceData, m interface{}) error {
	// TODO: implement after kubermatic client fix

	return resourceNodeDeploymentRead(d, m)
}

func resourceNodeDeploymentDelete(d *schema.ResourceData, m interface{}) error {
	k := m.(*kubermaticProviderMeta)
	dc := d.Get("dc").(string)
	pID := d.Get("project_id").(string)
	cID := d.Get("cluster_id").(string)
	nID := d.Id()
	p := project.NewDeleteNodeDeploymentParams()

	p.SetDC(dc)
	p.SetProjectID(pID)
	p.SetClusterID(cID)
	p.SetNodeDeploymentID(nID)

	_, err := k.client.Project.DeleteNodeDeployment(p, k.auth)
	if err != nil {
		// TODO: check if not found
		return fmt.Errorf("unable to delete node deployment '%s': %v", nID, err)
	}

	return resource.Retry(d.Timeout(schema.TimeoutDelete), func() *resource.RetryError {
		p := project.NewGetNodeDeploymentParams()
		p.SetDC(dc)
		p.SetProjectID(pID)
		p.SetClusterID(cID)
		p.SetNodeDeploymentID(nID)

		r, err := k.client.Project.GetNodeDeployment(p, k.auth)
		if err != nil {
			if e, ok := err.(*project.GetNodeDeploymentDefault); ok && e.Code() == http.StatusNotFound {
				k.log.Debugf("node deployment '%s' has been destroyed, returned http code: %d", nID, e.Code())
				d.SetId("")
				return nil
			}
			return resource.NonRetryableError(fmt.Errorf("unable to get node deployment '%s': %v", nID, err))
		}

		k.log.Debugf("node deployment '%s' deletion in progress, deletionTimestamp: %s",
			nID, r.Payload.DeletionTimestamp.String())
		return resource.RetryableError(fmt.Errorf("node deployment '%s' deletion in progress", nID))
	})
}
