package kubermatic

import (
	"fmt"

	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/kubermatic/go-kubermatic/client/project"
	"github.com/kubermatic/go-kubermatic/client/versions"
	"github.com/kubermatic/go-kubermatic/models"
)

func validateNodeSpecMatchesCluster() schema.CustomizeDiffFunc {
	return func(d *schema.ResourceDiff, meta interface{}) error {
		cluster, err := getClusterForNodeDeployment(d.Get("cluster_id").(string), meta.(*kubermaticProviderMeta))
		if err != nil {
			return err
		}
		err = validateVersionAgainstCluster(d, cluster)
		if err != nil {
			return err
		}
		err = validateProviderMatchesCluster(d, cluster)
		if err != nil {
			return err
		}
		return nil
	}
}

func getClusterVersion(cluster *models.Cluster) (*version.Version, error) {
	v, err := version.NewVersion(cluster.Spec.Version.(string))
	if err != nil {
		return nil, err
	}
	return v, nil
}

func getClusterCloudProvider(c *models.Cluster) (string, error) {
	switch {
	case c.Spec.Cloud.Bringyourown != nil:
		return "bringyourown", nil
	case c.Spec.Cloud.Aws != nil:
		return "aws", nil
	case c.Spec.Cloud.Openstack != nil:
		return "openstack", nil
	default:
		return "", fmt.Errorf("could not find cloud provider for cluster")

	}
}

func validateProviderMatchesCluster(d *schema.ResourceDiff, c *models.Cluster) error {
	var availableProviders = []string{"bringyourown", "aws", "openstack"}
	var provider string

	for _, p := range availableProviders {
		providerField := fmt.Sprintf("spec.0.template.0.cloud.0.%s", p)
		_, ok := d.GetOk(providerField)
		if ok {
			provider = p
			break
		}
	}
	clusterProvider, err := getClusterCloudProvider(c)
	if err != nil {
		return err
	}
	if provider != clusterProvider {
		return fmt.Errorf("provider for node deployment must (%s) match cluster provider (%s)", provider, clusterProvider)
	}
	return nil

}

func validateVersionAgainstCluster(d *schema.ResourceDiff, c *models.Cluster) error {
	dataVersion, ok := d.Get("spec.0.template.0.versions.0.kubelet").(string)
	if dataVersion == "" || !ok {
		return nil
	}

	v, err := version.NewVersion(dataVersion)

	if err != nil {
		return fmt.Errorf("unable to parse node deployment version")
	}

	clusterVersion, err := getClusterVersion(c)
	if err != nil {
		return err
	}

	if clusterVersion.LessThan(v) {
		return fmt.Errorf("node deployment version (%s) cannot be greater than cluster version (%s)", v, clusterVersion)
	}
	return nil
}

func validateKubeletVersionExists() schema.CustomizeDiffFunc {
	return func(d *schema.ResourceDiff, meta interface{}) error {
		k := meta.(*kubermaticProviderMeta)
		version := d.Get("kubelet").(string)
		versionType := "kubernetes"
		p := versions.NewGetNodeUpgradesParams()
		p.SetType(&versionType)
		p.SetControlPlaneVersion(&version)
		r, err := k.client.Versions.GetNodeUpgrades(p, k.auth)
		if err != nil {
			if e, ok := err.(*versions.GetNodeUpgradesDefault); ok && e.Payload != nil && e.Payload.Error != nil && e.Payload.Error.Message != nil {
				return fmt.Errorf("get node_deployment upgrades: %s", *e.Payload.Error.Message)
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

func getClusterForNodeDeployment(id string, k *kubermaticProviderMeta) (*models.Cluster, error) {
	projectID, seedDC, clusterID, err := kubermaticClusterParseID(id)
	if err != nil {
		return nil, err
	}

	p := project.NewGetClusterParams()
	p.SetProjectID(projectID)
	p.SetDC(seedDC)
	p.SetClusterID(clusterID)
	r, err := k.client.Project.GetCluster(p, k.auth)
	if err != nil {
		return nil, fmt.Errorf("unable to get cluster %s in project %s in seed dc %s - error: %v", clusterID, projectID, seedDC, err)
	}

	return r.Payload, nil
}
