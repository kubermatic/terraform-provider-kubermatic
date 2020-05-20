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
	healthStatusUp models.HealthStatus = 1
)

func resourceCluster() *schema.Resource {
	return &schema.Resource{
		Create: resourceClusterCreate,
		Read:   resourceClusterRead,
		Update: resourceClusterUpdate,
		Delete: resourceClusterDelete,

		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Reference project identifier",
			},
			"dc": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Data center name",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Cluster name",
			},
			"spec": {
				Type:        schema.TypeList,
				Required:    true,
				MaxItems:    1,
				Description: "Cluster specification",
				Elem: &schema.Resource{
					Schema: clusterSpecFields(),
				},
			},
			"credential": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Cluster access credential",
			},
			"type": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "kubernetes",
				Description: "Cluster type Kubernetes or OpenShift",
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

func resourceClusterCreate(d *schema.ResourceData, m interface{}) error {
	k := m.(*kubermaticProviderMeta)
	pID := d.Get("project_id").(string)
	dc := d.Get("dc").(string)
	p := project.NewCreateClusterParams()

	p.SetProjectID(pID)
	p.SetDC(dc)
	p.SetBody(&models.CreateClusterSpec{
		Cluster: &models.Cluster{
			Name:       d.Get("name").(string),
			Spec:       expandClusterSpec(d.Get("spec").([]interface{})),
			Type:       d.Get("type").(string),
			Credential: d.Get("credential").(string),
		},
	})

	r, err := k.client.Project.CreateCluster(p, k.auth)
	if err != nil {
		return fmt.Errorf("unable to create cluster for project '%s': %s", pID, err)
	}
	d.SetId(r.Payload.ID)
	cID := r.Payload.ID

	err = resource.Retry(d.Timeout(schema.TimeoutCreate), func() *resource.RetryError {
		hp := project.NewGetClusterHealthParams()
		hp.SetClusterID(cID)
		hp.SetProjectID(pID)
		hp.SetDC(dc)

		r, err := k.client.Project.GetClusterHealth(hp, k.auth)
		if err != nil {
			return resource.NonRetryableError(fmt.Errorf("unable to get cluster '%s' health: %v", cID, err))
		}

		if r.Payload.Apiserver == healthStatusUp &&
			r.Payload.CloudProviderInfrastructure == healthStatusUp &&
			r.Payload.Controller == healthStatusUp &&
			r.Payload.Etcd == healthStatusUp &&
			r.Payload.MachineController == healthStatusUp &&
			r.Payload.Scheduler == healthStatusUp &&
			r.Payload.UserClusterControllerManager == healthStatusUp {
			return nil
		}

		k.log.Debugf("waiting for cluster '%s' to be ready, %+v", cID, r.Payload)
		return resource.RetryableError(fmt.Errorf("waiting for cluster '%s' to be ready", cID))
	})
	if err != nil {
		return fmt.Errorf("cluster '%s' is not ready: %v", cID, err)
	}

	return resourceClusterRead(d, m)
}

func resourceClusterRead(d *schema.ResourceData, m interface{}) error {
	k := m.(*kubermaticProviderMeta)
	p := project.NewGetClusterParams()
	p.SetDC(d.Get("dc").(string))
	p.SetProjectID(d.Get("project_id").(string))
	p.SetClusterID(d.Id())

	r, err := k.client.Project.GetCluster(p, k.auth)
	if err != nil {
		if e, ok := err.(*project.GetClusterDefault); ok && e.Code() == http.StatusNotFound {
			k.log.Infof("removing cluster '%s' from terraform state file, could not find the resource", d.Id())
			d.SetId("")
			return nil
		}

		// TODO: check the cluster API code
		// when cluster does not exist but it is in terraform state file
		// the GET request returns 500 http code instead of 404, probably it's a bug
		// because of that manual action to clean terraform state file is required

		return fmt.Errorf("unable to get cluster '%s': %v", d.Id(), err)
	}

	d.Set("name", r.Payload.Name)

	// TODO: check why API returns an empty credential field even if it is set
	//err = d.Set("credential", r.Payload.Credential)
	//if err != nil {
	//	return err
	//}

	d.Set("type", r.Payload.Type)

	// TODO: Do not rewrite sensetive fields
	if err = d.Set("spec", flattenClusterSpec(r.Payload.Spec)); err != nil {
		return err
	}

	d.Set("creation_timestamp", r.Payload.CreationTimestamp.String())

	d.Set("deletion_timestamp", r.Payload.DeletionTimestamp.String())

	return nil
}

func resourceClusterUpdate(d *schema.ResourceData, m interface{}) error {
	// TODO: implement after kubermatic client fix

	return resourceClusterRead(d, m)
}

func resourceClusterDelete(d *schema.ResourceData, m interface{}) error {
	k := m.(*kubermaticProviderMeta)
	cId := d.Id()
	pID := d.Get("project_id").(string)
	dc := d.Get("dc").(string)
	p := project.NewDeleteClusterParams()

	p.SetDC(dc)
	p.SetProjectID(pID)
	p.SetClusterID(cId)

	_, err := k.client.Project.DeleteCluster(p, k.auth)
	if err != nil {
		return fmt.Errorf("unable to delete cluster '%s': %v", cId, err)
	}

	return resource.Retry(d.Timeout(schema.TimeoutDelete), func() *resource.RetryError {
		p := project.NewGetClusterParams()

		p.SetClusterID(cId)
		p.SetProjectID(pID)
		p.SetDC(dc)

		r, err := k.client.Project.GetCluster(p, k.auth)
		if err != nil {
			if e, ok := err.(*project.GetClusterDefault); ok && e.Code() == http.StatusNotFound {
				k.log.Debugf("cluster '%s' has been destroyed, returned http code: %d", cId, e.Code())
				d.SetId("")
				return nil
			}
			return resource.NonRetryableError(fmt.Errorf("unable to get cluster '%s': %v", cId, err))
		}

		k.log.Debugf("cluster '%s' deletion in progress, deletionTimestamp: %s",
			cId, r.Payload.DeletionTimestamp.String())
		return resource.RetryableError(fmt.Errorf("cluster '%s' deletion in progress", cId))
	})
}
