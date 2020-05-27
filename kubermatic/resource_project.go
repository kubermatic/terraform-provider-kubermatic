package kubermatic

import (
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/kubermatic/go-kubermatic/client/project"
	"github.com/kubermatic/go-kubermatic/models"
)

const (
	projectActive   = "Active"
	projectInactive = "Inactive"
)

func resourceProject() *schema.Resource {
	return &schema.Resource{
		Create: resourceProjectCreate,
		Read:   resourceProjectRead,
		Update: resourceProjectUpdate,
		Delete: resourceProjectDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Project name",
			},
			"labels": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "Project labels",
				Elem:        schema.TypeString,
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

func resourceProjectCreate(d *schema.ResourceData, m interface{}) error {
	k := m.(*kubermaticProviderMeta)
	p := project.NewCreateProjectParams()

	p.Body.Name = d.Get("name").(string)
	if l, ok := d.GetOk("labels"); ok {
		p.Body.Labels = make(map[string]string)
		att := l.(map[string]interface{})
		for key, val := range att {
			p.Body.Labels[key] = val.(string)
		}
	}

	r, err := k.client.Project.CreateProject(p, k.auth)
	if err != nil {
		return fmt.Errorf("error when creating a project: %s", err)
	}
	d.SetId(r.Payload.ID)

	id := r.Payload.ID
	createStateConf := &resource.StateChangeConf{
		Pending: []string{
			projectInactive,
		},
		Target: []string{
			projectActive,
		},
		Refresh: func() (interface{}, string, error) {
			p := project.NewGetProjectParams()
			r, err := k.client.Project.GetProject(p.WithProjectID(id), k.auth)
			if err != nil {
				if e, ok := err.(*project.GetProjectDefault); ok && (e.Code() == http.StatusForbidden || e.Code() == http.StatusNotFound) {
					return r, projectInactive, nil
				}
				return nil, "", err
			}
			k.log.Debugf("creating project '%s', currently in '%s' state", id, r.Payload.Status)
			return r, r.Payload.Status, nil
		},
		Timeout:    d.Timeout(schema.TimeoutCreate),
		MinTimeout: retryTimeout,
		Delay:      requestDelay,
	}

	if _, err := createStateConf.WaitForState(); err != nil {
		k.log.Debugf("error while waiting for project '%s' to be created: %s", id, err)
		return fmt.Errorf("error while waiting for project '%s' to be created: %s", id, err)
	}
	return resourceProjectRead(d, m)
}

func resourceProjectRead(d *schema.ResourceData, m interface{}) error {
	k := m.(*kubermaticProviderMeta)
	p := project.NewGetProjectParams()

	r, err := k.client.Project.GetProject(p.WithProjectID(d.Id()), k.auth)
	if err != nil {
		if e, ok := err.(*project.GetProjectDefault); ok && (e.Code() == http.StatusForbidden || e.Code() == http.StatusNotFound) {
			// remove a project from terraform state file that a user does not have access or does not exist
			k.log.Infof("removing project '%s' from terraform state file, code '%d' has been returned", d.Id(), e.Code())
			d.SetId("")
			return nil

		}

		return fmt.Errorf("unable to get project '%s': %v", d.Id(), err)
	}

	if err := d.Set("labels", r.Payload.Labels); err != nil {
		return err
	}
	d.Set("name", r.Payload.Name)
	d.Set("status", r.Payload.Status)
	d.Set("creation_timestamp", r.Payload.CreationTimestamp.String())
	d.Set("deletion_timestamp", r.Payload.DeletionTimestamp.String())
	return nil
}

func resourceProjectUpdate(d *schema.ResourceData, m interface{}) error {
	k := m.(*kubermaticProviderMeta)
	p := project.NewUpdateProjectParams()
	p.Body = &models.Project{
		// name is always required for update requests, otherwise bad request returns
		Name: d.Get("name").(string),
	}

	if d.HasChange("name") {
		p.Body.Name = d.Get("name").(string)
	}

	if d.HasChange("labels") {
		p.Body.Labels = make(map[string]string)
		for key, val := range d.Get("labels").(map[string]interface{}) {
			p.Body.Labels[key] = val.(string)
		}
	}

	_, err := k.client.Project.UpdateProject(p.WithProjectID(d.Id()), k.auth)
	if err != nil {
		return fmt.Errorf("unable to update project '%s': %v", d.Id(), err)
	}

	return resourceProjectRead(d, m)
}

func resourceProjectDelete(d *schema.ResourceData, m interface{}) error {
	k := m.(*kubermaticProviderMeta)
	p := project.NewDeleteProjectParams()
	_, err := k.client.Project.DeleteProject(p.WithProjectID(d.Id()), k.auth)
	if err != nil {
		return fmt.Errorf("unable to delete project '%s': %s", d.Id(), err)
	}

	return resource.Retry(d.Timeout(schema.TimeoutDelete), func() *resource.RetryError {
		p := project.NewGetProjectParams()
		r, err := k.client.Project.GetProject(p.WithProjectID(d.Id()), k.auth)
		if err != nil {
			e, ok := err.(*project.GetProjectDefault)
			if ok && (e.Code() == http.StatusForbidden || e.Code() == http.StatusNotFound) {
				k.log.Debugf("project '%s' has been destroyed, returned http code: %d", d.Id(), e.Code())
				return nil
			}
			return resource.NonRetryableError(err)
		}
		k.log.Debugf("project '%s' deletion in progress, deletionTimestamp: %s, status: %s",
			d.Id(), r.Payload.DeletionTimestamp.String(), r.Payload.Status)
		return resource.RetryableError(
			fmt.Errorf("project '%s' still exists, currently in '%s' state", d.Id(), r.Payload.Status),
		)
	})
}
