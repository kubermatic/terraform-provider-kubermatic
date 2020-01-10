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
				Type:     schema.TypeString,
				Required: true,
			},
			"project_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"cluster_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"spec": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Required: true,
				Elem: &schema.Resource{
					Schema: nodeDeploymentSpecFields(),
				},
			},
			"creation_timestamp": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"deletion_timestamp": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceNodeDeploymentCreate(d *schema.ResourceData, m interface{}) error {
	k := m.(*kubermaticProvider)
	dc := d.Get("dc").(string)
	pId := d.Get("project_id").(string)
	cId := d.Get("cluster_id").(string)
	p := project.NewCreateNodeDeploymentParams()

	p.SetProjectID(pId)
	p.SetClusterID(cId)
	p.SetDc(dc)
	p.SetBody(&models.NodeDeployment{
		Name: d.Get("name").(string),
		Spec: expandNodeDeploymentSpec(d.Get("spec").([]interface{})),
	})

	r, err := k.client.Project.CreateNodeDeployment(p, k.auth)
	if err != nil {
		return fmt.Errorf("unable to create a node deployment: %v", err)
	}
	nId := r.Payload.ID

	err = resource.Retry(d.Timeout(schema.TimeoutCreate), func() *resource.RetryError {
		p := project.NewGetNodeDeploymentParams()
		p.SetProjectID(pId)
		p.SetClusterID(cId)
		p.SetDc(dc)
		p.SetNodedeploymentID(nId)

		r, err := k.client.Project.GetNodeDeployment(p, k.auth)
		if err != nil {
			return resource.NonRetryableError(fmt.Errorf("unable to get node deployment '%s' status: %v", nId, err))
		}

		if r.Payload.Status.ReadyReplicas < *r.Payload.Spec.Replicas {
			k.log.Debugf("waiting for node deployment '%s' to be ready, %+v", nId, r.Payload.Status)
			return resource.RetryableError(fmt.Errorf("waiting for node deployment '%s' to be ready", nId))
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("node deployment '%s' is not ready: %v", nId, err)
	}

	d.SetId(nId)
	return resourceNodeDeploymentRead(d, m)
}

func resourceNodeDeploymentRead(d *schema.ResourceData, m interface{}) error {
	k := m.(*kubermaticProvider)
	p := project.NewGetNodeDeploymentParams()

	p.SetDc(d.Get("dc").(string))
	p.SetProjectID(d.Get("project_id").(string))
	p.SetClusterID(d.Get("cluster_id").(string))
	p.SetNodedeploymentID(d.Id())

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
	k := m.(*kubermaticProvider)
	dc := d.Get("dc").(string)
	pId := d.Get("project_id").(string)
	cId := d.Get("cluster_id").(string)
	nId := d.Id()
	p := project.NewDeleteNodeDeploymentParams()

	p.SetDc(dc)
	p.SetProjectID(pId)
	p.SetClusterID(cId)
	p.SetNodedeploymentID(nId)

	_, err := k.client.Project.DeleteNodeDeployment(p, k.auth)
	if err != nil {
		// TODO: check if not found
		return fmt.Errorf("unable to delete node deployment '%s': %v", nId, err)
	}

	return resource.Retry(d.Timeout(schema.TimeoutDelete), func() *resource.RetryError {
		p := project.NewGetNodeDeploymentParams()
		p.SetDc(dc)
		p.SetProjectID(pId)
		p.SetClusterID(cId)
		p.SetNodedeploymentID(nId)

		r, err := k.client.Project.GetNodeDeployment(p, k.auth)
		if err != nil {
			if e, ok := err.(*project.GetNodeDeploymentDefault); ok && e.Code() == http.StatusNotFound {
				k.log.Debugf("node deployment '%s' has been destroyed, returned http code: %d", nId, e.Code())
				d.SetId("")
				return nil
			}
			return resource.NonRetryableError(fmt.Errorf("unable to get node deployment '%s': %v", nId, err))
		}

		k.log.Debugf("node deployment '%s' deletion in progress, deletionTimestamp: %s",
			nId, r.Payload.DeletionTimestamp.String())
		return resource.RetryableError(fmt.Errorf("node deployment '%s' deletion in progress", nId))
	})
}
