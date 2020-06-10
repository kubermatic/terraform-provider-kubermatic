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
		k := meta.(*kubermaticProviderMeta)
		cluster, err := getClusterForNodeDeployment(d.Get("cluster_id").(string), k)
		if err != nil {
			return err
		}
		clusterVersion := cluster.Spec.Version.(string)
		if err != nil {
			return err
		}
		err = validateVersionAgainstCluster(d, clusterVersion)
		if err != nil {
			return err
		}
		clusterProvider, err := getClusterCloudProvider(cluster)
		if err != nil {
			return err
		}
		err = validateProviderMatchesCluster(d, clusterProvider)
		if err != nil {
			return err
		}
		err = validateKubeletVersionIsAvailable(d, k, clusterVersion)
		if err != nil {
			return err
		}
		return nil
	}
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

func validateProviderMatchesCluster(d *schema.ResourceDiff, clusterProvider string) error {
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
	if provider != clusterProvider {
		return fmt.Errorf("provider for node deployment must (%s) match cluster provider (%s)", provider, clusterProvider)
	}
	return nil

}

func validateVersionAgainstCluster(d *schema.ResourceDiff, clusterVersion string) error {
	nodeVersion, ok := d.Get("spec.0.template.0.versions.0.kubelet").(string)
	if nodeVersion == "" || !ok {
		return nil
	}

	clusterSemverVersion, err := version.NewVersion(clusterVersion)
	if err != nil {
		return err
	}

	v, err := version.NewVersion(nodeVersion)

	if err != nil {
		return fmt.Errorf("unable to parse node deployment version")
	}

	if clusterSemverVersion.LessThan(v) {
		return fmt.Errorf("node deployment version (%s) cannot be greater than cluster version (%s)", v, clusterVersion)
	}
	return nil
}

func validateKubeletVersionIsAvailable(d *schema.ResourceDiff, k *kubermaticProviderMeta, clusterVersion string) error {
	version := d.Get("spec.0.template.0.versions.0.kubelet").(string)
	versionType := "kubernetes"

	p := versions.NewGetNodeUpgradesParams()
	p.SetType(&versionType)
	p.SetControlPlaneVersion(&clusterVersion)
	r, err := k.client.Versions.GetNodeUpgrades(p, k.auth)

	if err != nil {
		if e, ok := err.(*versions.GetNodeUpgradesDefault); ok && e.Payload != nil && e.Payload.Error != nil && e.Payload.Error.Message != nil {
			return fmt.Errorf("get node_deployment upgrades: %s", *e.Payload.Error.Message)
		}
		return err
	}

	var availableVersions []string
	for _, v := range r.Payload {
		s, ok := v.Version.(string)
		if ok && s == version && !v.RestrictedByKubeletVersion {
			return nil
		}
		availableVersions = append(availableVersions, s)
	}

	return fmt.Errorf("unknown version for node deployment %s, available versions %v", version, availableVersions)
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
