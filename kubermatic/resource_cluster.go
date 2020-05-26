package kubermatic

import (
	"fmt"
	"net/http"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-plugin-sdk/helper/customdiff"
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
				ForceNew:    true,
				Description: "Reference project identifier",
			},
			"dc": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Data center name",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Cluster name",
			},
			"labels": {
				Type:     schema.TypeMap,
				Optional: true,
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
				ForceNew:    true,
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

		CustomizeDiff: customdiff.All(
			customdiff.ForceNewIfChange("spec.0.version", func(old, new, meta interface{}) bool {
				// "version" can only be upgraded to newer versions, so we must create a new resource
				// if it is decreased.
				newVer, err := version.NewVersion(new.(string))
				if err != nil {
					return false
				}

				oldVer, err := version.NewVersion(old.(string))
				if err != nil {
					return false
				}

				if newVer.LessThan(oldVer) {
					return true
				}
				return false
			}),
		),
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
			Labels:     getLabels(d),
			Credential: d.Get("credential").(string),
		},
	})

	r, err := k.client.Project.CreateCluster(p, k.auth)
	if err != nil {
		return fmt.Errorf("unable to create cluster for project '%s': %s", pID, err)
	}
	d.SetId(r.Payload.ID)

	if err := waitClusterReady(k, d); err != nil {
		return fmt.Errorf("cluster '%s' is not ready: %v", r.Payload.ID, err)
	}

	return resourceClusterRead(d, m)
}

func getLabels(d *schema.ResourceData) map[string]string {
	var labels map[string]string
	if v := d.Get("labels"); v != nil {
		labels = make(map[string]string)
		m := d.Get("labels").(map[string]interface{})
		for k, v := range m {
			labels[k] = v.(string)
		}
	}
	return labels
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

	labels, err := excludeProjectLabels(k, d.Get("project_id").(string), r.Payload.Labels)
	if err != nil {
		return err
	}
	if err := d.Set("labels", labels); err != nil {
		return err
	}

	d.Set("name", r.Payload.Name)

	// TODO: check why API returns an empty credential field even if it is set
	//err = d.Set("credential", r.Payload.Credential)
	//if err != nil {
	//	return err
	//}

	d.Set("type", r.Payload.Type)

	values := readClusterPreserveValues(d)
	specFlattenned := flattenClusterSpec(values, r.Payload.Spec)
	if err = d.Set("spec", specFlattenned); err != nil {
		return err
	}

	d.Set("creation_timestamp", r.Payload.CreationTimestamp.String())

	d.Set("deletion_timestamp", r.Payload.DeletionTimestamp.String())

	return nil
}

// excludeProjectLabels excludes labels defined in project.
// Project labels propogated to clusters. For better predictability of
// cluster's labels changes, project's labels are excluded from cluster state.
func excludeProjectLabels(k *kubermaticProviderMeta, projectID string, allLabels map[string]string) (map[string]string, error) {
	p := project.NewGetProjectParams()
	p.SetProjectID(projectID)

	r, err := k.client.Project.GetProject(p, k.auth)
	if err != nil {
		return nil, err
	}

	for k := range r.Payload.Labels {
		delete(allLabels, k)
	}

	return allLabels, nil
}

// clusterPreserveValues helps avoid misleading diffs during read phase.
// API result does not have some important fields valeus, like sensitive
// access key or password fields. When API result is flattened and written to
// terraform state it creating state diff that might force undesired updates and
// even force replacement of a cluster. Solution is to set values for preserved
// values in flattened object before comitting it to state.
type clusterPreserveValues struct {
	openstackUsername interface{}
	openstackPassword interface{}
	openstackTenant   interface{}
}

func readClusterPreserveValues(d *schema.ResourceData) clusterPreserveValues {
	return clusterPreserveValues{
		openstackUsername: d.Get("spec.0.cloud.0.openstack.0.username"),
		openstackPassword: d.Get("spec.0.cloud.0.openstack.0.password"),
		openstackTenant:   d.Get("spec.0.cloud.0.openstack.0.tenant"),
	}
}

func resourceClusterUpdate(d *schema.ResourceData, m interface{}) error {
	k := m.(*kubermaticProviderMeta)
	p := project.NewPatchClusterParams()
	p.SetProjectID(d.Get("project_id").(string))
	p.SetDC(d.Get("dc").(string))
	p.SetClusterID(d.Id())
	p.SetPatch(newClusterPatch(d.Get("name").(string), d.Get("spec.0.version").(string), d.Get("spec.0.audit_logging").(bool), d.Get("labels")))

	err := resource.Retry(d.Timeout(schema.TimeoutUpdate), func() *resource.RetryError {
		_, err := k.client.Project.PatchCluster(p, k.auth)
		if err != nil {
			if e, ok := err.(*project.PatchClusterDefault); ok && e.Code() == http.StatusConflict {
				return resource.RetryableError(fmt.Errorf("cluster patch conflict: %w", err))
			}
			return resource.NonRetryableError(fmt.Errorf("patch cluster '%s': %v", d.Id(), err))
		}
		return nil
	})
	if err != nil {
		return err
	}

	if err := waitClusterReady(k, d); err != nil {
		return fmt.Errorf("cluster '%s' is not ready: %v", d.Id(), err)
	}

	return resourceClusterRead(d, m)
}

func waitClusterReady(k *kubermaticProviderMeta, d *schema.ResourceData) error {
	return resource.Retry(d.Timeout(schema.TimeoutCreate), func() *resource.RetryError {
		hp := project.NewGetClusterHealthParams()
		hp.SetClusterID(d.Id())
		hp.SetProjectID(d.Get("project_id").(string))
		hp.SetDC(d.Get("dc").(string))

		r, err := k.client.Project.GetClusterHealth(hp, k.auth)
		if err != nil {
			return resource.NonRetryableError(fmt.Errorf("unable to get cluster '%s' health: %v", d.Id(), err))
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

		k.log.Debugf("waiting for cluster '%s' to be ready, %+v", d.Id(), r.Payload)
		return resource.RetryableError(fmt.Errorf("waiting for cluster '%s' to be ready", d.Id()))
	})
}

func newClusterPatch(name, version string, auditLogging bool, labels interface{}) interface{} {
	// TODO(furkhat): change to dedicated struct when API has it.
	return map[string]interface{}{
		"name":   name,
		"labels": labels,
		"spec": map[string]interface{}{
			"auditLogging": map[string]bool{
				"enabled": auditLogging,
			},
			"version": version,
		},
	}
}

func resourceClusterDelete(d *schema.ResourceData, m interface{}) error {
	k := m.(*kubermaticProviderMeta)
	cID := d.Id()
	pID := d.Get("project_id").(string)
	dc := d.Get("dc").(string)
	p := project.NewDeleteClusterParams()

	p.SetDC(dc)
	p.SetProjectID(pID)
	p.SetClusterID(cID)

	_, err := k.client.Project.DeleteCluster(p, k.auth)
	if err != nil {
		return fmt.Errorf("unable to delete cluster '%s': %v", cID, err)
	}

	return resource.Retry(d.Timeout(schema.TimeoutDelete), func() *resource.RetryError {
		p := project.NewGetClusterParams()

		p.SetClusterID(cID)
		p.SetProjectID(pID)
		p.SetDC(dc)

		r, err := k.client.Project.GetCluster(p, k.auth)
		if err != nil {
			if e, ok := err.(*project.GetClusterDefault); ok && e.Code() == http.StatusNotFound {
				k.log.Debugf("cluster '%s' has been destroyed, returned http code: %d", cID, e.Code())
				d.SetId("")
				return nil
			}
			return resource.NonRetryableError(fmt.Errorf("unable to get cluster '%s': %v", cID, err))
		}

		k.log.Debugf("cluster '%s' deletion in progress, deletionTimestamp: %s",
			cID, r.Payload.DeletionTimestamp.String())
		return resource.RetryableError(fmt.Errorf("cluster '%s' deletion in progress", cID))
	})
}
