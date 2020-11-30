package metakube

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-plugin-sdk/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/syseleven/terraform-provider-metakube/go-metakube/client/datacenter"
	"github.com/syseleven/terraform-provider-metakube/go-metakube/client/project"
	"github.com/syseleven/terraform-provider-metakube/go-metakube/client/versions"
	"github.com/syseleven/terraform-provider-metakube/go-metakube/models"
)

const (
	healthStatusUp models.HealthStatus = 1
)

var supportedProviders = []string{"aws", "openstack", "azure"}

func resourceCluster() *schema.Resource {
	return &schema.Resource{
		Create: resourceClusterCreate,
		Read:   resourceClusterRead,
		Update: resourceClusterUpdate,
		Delete: resourceClusterDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

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
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Cluster name",
			},
			"labels": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "Labels added to cluster",
			},
			"sshkeys": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "SSH keys attached to nodes",
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.NoZeroValues,
				},
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
			// TODO: uncomment once `no consumer: "application/yaml"` error in metakube client is fixed.
			// "kube_config": kubernetesConfigSchema(),
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
			validateVersionExists(),
			validateOnlyOneCloudProviderSpecified(),
		),
	}
}

func validateVersionExists() schema.CustomizeDiffFunc {
	return func(d *schema.ResourceDiff, meta interface{}) error {
		k := meta.(*metakubeProviderMeta)
		version := d.Get("spec.0.version").(string)
		p := versions.NewGetMasterVersionsParams()
		r, err := k.client.Versions.GetMasterVersions(p, k.auth)
		if err != nil {
			if e, ok := err.(*versions.GetMasterVersionsDefault); ok && errorMessage(e.Payload) != "" {
				return fmt.Errorf("get cluster upgrades: %s", errorMessage(e.Payload))
			}
			return err
		}

		for _, v := range r.Payload {
			if s, ok := v.Version.(string); ok && s == version {
				return nil
			}
		}

		return fmt.Errorf("unknown version %s", version)
	}
}

func validateOnlyOneCloudProviderSpecified() schema.CustomizeDiffFunc {
	return func(d *schema.ResourceDiff, meta interface{}) error {
		var existingProviders []string
		counter := 0
		for _, provider := range supportedProviders {
			if _, ok := d.GetOk(fmt.Sprintf("spec.0.cloud.0.%s.0", provider)); ok {
				existingProviders = append(existingProviders, provider)
				counter++
			}
		}
		if counter > 1 {
			return fmt.Errorf("only one cloud provider must be specified: %v", existingProviders)
		}
		return nil
	}
}

func resourceClusterCreate(d *schema.ResourceData, m interface{}) error {
	k := m.(*metakubeProviderMeta)
	pID := d.Get("project_id").(string)
	dcName := d.Get("dc_name").(string)
	p := project.NewCreateClusterParams()

	dc, err := getDatacenterByName(k, dcName)
	if err != nil {
		return err
	}

	p.SetProjectID(pID)
	p.SetDC(dc.Spec.Seed)
	p.SetBody(&models.CreateClusterSpec{
		Cluster: &models.Cluster{
			Name:       d.Get("name").(string),
			Spec:       expandClusterSpec(d.Get("spec").([]interface{}), d.Get("dc_name").(string)),
			Type:       d.Get("type").(string),
			Labels:     getLabels(d),
			Credential: d.Get("credential").(string),
		},
	})

	r, err := k.client.Project.CreateCluster(p, k.auth)
	if err != nil {
		return fmt.Errorf("unable to create cluster for project '%s': %s", pID, getErrorResponse(err))
	}
	d.SetId(metakubeClusterMakeID(d.Get("project_id").(string), dc.Spec.Seed, r.Payload.ID))

	raw := d.Get("sshkeys").(*schema.Set).List()
	var sshkeys []string
	for _, v := range raw {
		sshkeys = append(sshkeys, v.(string))
	}
	if err := assignSSHKeysToCluster(pID, dc.Spec.Seed, r.Payload.ID, sshkeys, k); err != nil {
		return err
	}

	projectID, seedDC, clusterID, err := metakubeClusterParseID(d.Id())
	if err != nil {
		return err
	}
	if err := waitClusterReady(k, d, projectID, seedDC, clusterID); err != nil {
		return fmt.Errorf("cluster '%s' is not ready: %v", r.Payload.ID, err)
	}

	return resourceClusterRead(d, m)
}

func metakubeClusterMakeID(project, seedDC, id string) string {
	return fmt.Sprintf("%s:%s:%s", project, seedDC, id)
}

func metakubeClusterParseID(id string) (string, string, string, error) {
	parts := strings.SplitN(id, ":", 3)

	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return "", "", "", fmt.Errorf("unexpected format of ID (%s), expected project_id:seed_dc:id", id)
	}

	return parts[0], parts[1], parts[2], nil
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

func getDatacenterByName(k *metakubeProviderMeta, name string) (*models.Datacenter, error) {
	p := datacenter.NewListDatacentersParams()
	r, err := k.client.Datacenter.ListDatacenters(p, k.auth)
	if err != nil {
		if e, ok := err.(*datacenter.ListDatacentersDefault); ok && errorMessage(e.Payload) != "" {
			return nil, fmt.Errorf("list datacenters: %s", errorMessage(e.Payload))
		}
		return nil, fmt.Errorf("list datacenters: %v", err)
	}

	for _, v := range r.Payload {
		if v.Spec.Seed != "" && v.Metadata.Name == name {
			return v, nil
		}
	}

	return nil, fmt.Errorf("Datacenter '%s' not found", name)
}

func resourceClusterRead(d *schema.ResourceData, m interface{}) error {
	k := m.(*metakubeProviderMeta)
	p := project.NewGetClusterParams()
	projectID, seedDC, clusterID, err := metakubeClusterParseID(d.Id())
	if err != nil {
		return err
	}
	p.SetProjectID(projectID)
	p.SetDC(seedDC)
	p.SetClusterID(clusterID)

	r, err := k.client.Project.GetCluster(p, k.auth)
	if getClusterErrResourceIsDeleted(err) {
		k.log.Infof("removing cluster '%s' from terraform state file, could not find the resource", d.Id())
		d.SetId("")
		return nil
	}
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

		return fmt.Errorf("unable to get cluster '%s': %s", d.Id(), getErrorResponse(err))
	}

	d.Set("project_id", projectID)
	d.Set("dc_name", r.Payload.Spec.Cloud.DatacenterName)

	labels, err := excludeProjectLabels(k, projectID, r.Payload.Labels)
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

	keys, err := metakubeClusterGetAssignedSSHKeys(d, k)
	if err != nil {
		return err
	}
	if err := d.Set("sshkeys", keys); err != nil {
		return err
	}

	return nil
}

func getClusterErrResourceIsDeleted(err error) bool {
	if err == nil {
		return false
	}

	e, ok := err.(*project.GetClusterDefault)
	if !ok {
		return false
	}

	// All api replies and errors, that nevertheless indicate cluster was deleted.
	// TODO: adjust when https://github.com/metakube/metakube/issues/5462 fixed
	return e.Code() == http.StatusNotFound || errorMessage(e.Payload) == "no userInfo in request"
}

// excludeProjectLabels excludes labels defined in project.
// Project labels propogated to clusters. For better predictability of
// cluster's labels changes, project's labels are excluded from cluster state.
func excludeProjectLabels(k *metakubeProviderMeta, projectID string, allLabels map[string]string) (map[string]string, error) {
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

func metakubeClusterGetAssignedSSHKeys(d *schema.ResourceData, k *metakubeProviderMeta) ([]string, error) {
	p := project.NewListSSHKeysAssignedToClusterParams()
	projectID, seedDC, clusterID, err := metakubeClusterParseID(d.Id())
	if err != nil {
		return nil, err
	}
	p.SetProjectID(projectID)
	p.SetDC(seedDC)
	p.SetClusterID(clusterID)
	ret, err := k.client.Project.ListSSHKeysAssignedToCluster(p, k.auth)
	if err != nil {
		return nil, err
	}

	var ids []string
	for _, v := range ret.Payload {
		ids = append(ids, v.ID)
	}
	return ids, nil
}

// clusterPreserveValues helps avoid misleading diffs during read phase.
// API result does not have some important fields valeus, like sensitive
// access key or password fields. When API result is flattened and written to
// terraform state it creating state diff that might force undesired updates and
// even force replacement of a cluster. Solution is to set values for preserved
// values in flattened object before comitting it to state.
type clusterPreserveValues struct {
	openstack *clusterOpenstackPreservedValues
	// API returns empty spec for Azure and AWS clusters, so we just preserve values used for creation
	azure *models.AzureCloudSpec
	aws   *models.AWSCloudSpec
}

type clusterOpenstackPreservedValues struct {
	openstackUsername interface{}
	openstackPassword interface{}
	openstackTenant   interface{}
}

func readClusterPreserveValues(d *schema.ResourceData) clusterPreserveValues {
	key := func(s string) string {
		return fmt.Sprint("spec.0.cloud.0.", s)
	}
	var openstack *clusterOpenstackPreservedValues
	if _, ok := d.GetOk(key("openstack.0")); ok {
		openstack = &clusterOpenstackPreservedValues{
			openstackUsername: d.Get(key("openstack.0.username")),
			openstackPassword: d.Get(key("openstack.0.password")),
			openstackTenant:   d.Get(key("openstack.0.tenant")),
		}
	}

	var azure *models.AzureCloudSpec
	if _, ok := d.GetOk(key("azure.0")); ok {
		azure = &models.AzureCloudSpec{
			AvailabilitySet: d.Get(key("azure.0.availability_set")).(string),
			ClientID:        d.Get(key("azure.0.client_id")).(string),
			ClientSecret:    d.Get(key("azure.0.client_secret")).(string),
			SubscriptionID:  d.Get(key("azure.0.subscription_id")).(string),
			TenantID:        d.Get(key("azure.0.tenant_id")).(string),
			ResourceGroup:   d.Get(key("azure.0.resource_group")).(string),
			RouteTableName:  d.Get(key("azure.0.route_table")).(string),
			SecurityGroup:   d.Get(key("azure.0.security_group")).(string),
			SubnetName:      d.Get(key("azure.0.subnet")).(string),
			VNetName:        d.Get(key("azure.0.vnet")).(string),
		}
	}

	var aws *models.AWSCloudSpec
	if _, ok := d.GetOk(key("aws.0")); ok {
		aws = &models.AWSCloudSpec{
			AccessKeyID:         d.Get(key("aws.0.access_key_id")).(string),
			SecretAccessKey:     d.Get(key("aws.0.secret_access_key")).(string),
			VPCID:               d.Get(key("aws.0.vpc_id")).(string),
			SecurityGroupID:     d.Get(key("aws.0.security_group_id")).(string),
			RouteTableID:        d.Get(key("aws.0.route_table_id")).(string),
			InstanceProfileName: d.Get(key("aws.0.instance_profile_name")).(string),
			ControlPlaneRoleARN: d.Get(key("aws.0.role_arn")).(string),
		}
	}

	return clusterPreserveValues{
		openstack,
		azure,
		aws,
	}
}

func resourceClusterUpdate(d *schema.ResourceData, m interface{}) error {
	d.Partial(true)
	defer d.Partial(false)

	k := m.(*metakubeProviderMeta)

	if d.HasChanges("name", "labels", "spec") {
		if err := patchClusterFields(d, k); err != nil {
			return err
		}
		d.SetPartial("name")
		d.SetPartial("labels")
		d.SetPartial("spec")
	}
	if d.HasChange("sshkeys") {
		if err := updateClusterSSHKeys(d, k); err != nil {
			return err
		}
		d.SetPartial("sshkeys")
	}

	projectID, seedDC, clusterID, err := metakubeClusterParseID(d.Id())
	if err != nil {
		return err
	}
	if err := waitClusterReady(k, d, projectID, seedDC, clusterID); err != nil {
		return fmt.Errorf("cluster '%s' is not ready: %v", d.Id(), err)
	}

	return resourceClusterRead(d, m)
}

func patchClusterFields(d *schema.ResourceData, k *metakubeProviderMeta) error {
	p := project.NewPatchClusterParams()
	projectID, seedDC, clusterID, err := metakubeClusterParseID(d.Id())
	if err != nil {
		return err
	}
	p.SetProjectID(projectID)
	p.SetDC(seedDC)
	p.SetClusterID(clusterID)
	name := d.Get("name").(string)
	version := d.Get("spec.0.version").(string)
	auditLogging := d.Get("spec.0.audit_logging").(bool)
	labels := d.Get("labels")
	p.SetPatch(newClusterPatch(name, version, auditLogging, labels))

	err = resource.Retry(d.Timeout(schema.TimeoutUpdate), func() *resource.RetryError {
		_, err := k.client.Project.PatchCluster(p, k.auth)
		if err != nil {
			if e, ok := err.(*project.PatchClusterDefault); ok && e.Code() == http.StatusConflict {
				return resource.RetryableError(fmt.Errorf("cluster patch conflict: %v", err))
			} else if ok && errorMessage(e.Payload) != "" {
				return resource.NonRetryableError(fmt.Errorf("patch cluster '%s': %s", d.Id(), errorMessage(e.Payload)))
			}
			return resource.NonRetryableError(fmt.Errorf("patch cluster '%s': %v", d.Id(), err))
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func updateClusterSSHKeys(d *schema.ResourceData, k *metakubeProviderMeta) error {
	var unassign, assign []string
	old, new := d.GetChange("sshkeys")

	for _, id := range old.(*schema.Set).List() {
		if !new.(*schema.Set).Contains(id) {
			unassign = append(unassign, id.(string))
		}
	}

	for _, id := range new.(*schema.Set).List() {
		if !old.(*schema.Set).Contains(id) {
			assign = append(assign, id.(string))
		}
	}

	projectID, seedDC, clusterID, err := metakubeClusterParseID(d.Id())
	if err != nil {
		return err
	}

	for _, id := range unassign {
		p := project.NewDetachSSHKeyFromClusterParams()
		p.SetProjectID(projectID)
		p.SetDC(seedDC)
		p.SetClusterID(clusterID)
		p.SetKeyID(id)
		_, err := k.client.Project.DetachSSHKeyFromCluster(p, k.auth)
		if err != nil {
			if e, ok := err.(*project.DetachSSHKeyFromClusterDefault); ok && e.Code() == http.StatusNotFound {
				continue
			}
			return err
		}
	}

	if err := assignSSHKeysToCluster(projectID, seedDC, clusterID, assign, k); err != nil {
		return err
	}

	return nil
}

func assignSSHKeysToCluster(projectID, seedDC, clusterID string, sshkeyIDs []string, k *metakubeProviderMeta) error {
	for _, id := range sshkeyIDs {
		p := project.NewAssignSSHKeyToClusterParams()
		p.SetProjectID(projectID)
		p.SetDC(seedDC)
		p.SetClusterID(clusterID)
		p.SetKeyID(id)
		_, err := k.client.Project.AssignSSHKeyToCluster(p, k.auth)
		if err != nil {
			return fmt.Errorf("unable to assign sshkeys to cluster '%s': %v", clusterID, err)
		}
	}

	return nil
}

func waitClusterReady(k *metakubeProviderMeta, d *schema.ResourceData, projectID, seedDC, clusterID string) error {
	return resource.Retry(d.Timeout(schema.TimeoutCreate), func() *resource.RetryError {

		p := project.NewGetClusterHealthParams()
		p.SetProjectID(projectID)
		p.SetDC(seedDC)
		p.SetClusterID(clusterID)

		r, err := k.client.Project.GetClusterHealth(p, k.auth)
		if err != nil {
			return resource.RetryableError(fmt.Errorf("unable to get cluster '%s' health: %s", d.Id(), getErrorResponse(err)))
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
	k := m.(*metakubeProviderMeta)
	projectID, seedDC, clusterID, err := metakubeClusterParseID(d.Id())
	if err != nil {
		return err
	}
	p := project.NewDeleteClusterParams()

	p.SetProjectID(projectID)
	p.SetDC(seedDC)
	p.SetClusterID(clusterID)

	deleteSent := false
	return resource.Retry(d.Timeout(schema.TimeoutDelete), func() *resource.RetryError {
		if !deleteSent {
			_, err := k.client.Project.DeleteCluster(p, k.auth)
			if err != nil {
				if e, ok := err.(*project.DeleteClusterDefault); ok {
					if e.Code() == http.StatusConflict {
						return resource.RetryableError(err)
					}
					if e.Code() == http.StatusNotFound {
						return nil
					}
				}
				if _, ok := err.(*project.DeleteClusterForbidden); ok {
					return nil
				}
				return resource.NonRetryableError(fmt.Errorf("unable to delete cluster '%s': %s", d.Id(), getErrorResponse(err)))
			}
			deleteSent = true
		}
		p := project.NewGetClusterParams()

		p.SetProjectID(projectID)
		p.SetDC(seedDC)
		p.SetClusterID(clusterID)

		r, err := k.client.Project.GetCluster(p, k.auth)
		if err != nil {
			if e, ok := err.(*project.GetClusterDefault); ok && e.Code() == http.StatusNotFound {
				k.log.Debugf("cluster '%s' has been destroyed, returned http code: %d", d.Id(), e.Code())
				return nil
			}
			if _, ok := err.(*project.GetClusterForbidden); ok {
				return nil
			}
			return resource.NonRetryableError(fmt.Errorf("unable to get cluster '%s': %s", d.Id(), getErrorResponse(err)))
		}

		k.log.Debugf("cluster '%s' deletion in progress, deletionTimestamp: %s",
			d.Id(), r.Payload.DeletionTimestamp.String())
		return resource.RetryableError(fmt.Errorf("cluster '%s' deletion in progress", d.Id()))
	})
}
