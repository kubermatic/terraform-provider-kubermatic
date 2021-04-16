package metakube

import (
	"context"
	"fmt"

	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/syseleven/go-metakube/client/project"
	"github.com/syseleven/go-metakube/client/versions"
	"github.com/syseleven/go-metakube/models"
)

func validateNodeSpecMatchesCluster() schema.CustomizeDiffFunc {
	return func(ctx context.Context, d *schema.ResourceDiff, meta interface{}) error {
		k := meta.(*metakubeProviderMeta)
		clusterID := d.Get("cluster_id").(string)
		if clusterID == "" {
			return nil
		}
		projectID := d.Get("project_id").(string)
		if projectID == "" {
			return nil
		}
		cluster, err := metakubeGetCluster(projectID, clusterID, k)
		if err != nil {
			return err
		}
		clusterVersion := cluster.Spec.Version.(string)

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
	case c.Spec.Cloud.Azure != nil:
		return "azure", nil
	default:
		return "", fmt.Errorf("could not find cloud provider for cluster")

	}
}

func validateProviderMatchesCluster(d *schema.ResourceDiff, clusterProvider string) error {
	var availableProviders = []string{"bringyourown", "aws", "openstack", "azure"}
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

func validateKubeletVersionIsAvailable(d *schema.ResourceDiff, k *metakubeProviderMeta, clusterVersion string) error {
	version := d.Get("spec.0.template.0.versions.0.kubelet").(string)
	if version == "" {
		return nil
	}

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

func metakubeGetCluster(proj, cls string, k *metakubeProviderMeta) (*models.Cluster, error) {
	p := project.NewGetClusterV2Params().
		WithProjectID(proj).
		WithClusterID(cls)
	r, err := k.client.Project.GetClusterV2(p, k.auth)
	if err != nil {
		return nil, fmt.Errorf("unable to get cluster %s in project %s - error: %v", cls, proj, err)
	}

	return r.Payload, nil
}

func validateAutoscalerFields() schema.CustomizeDiffFunc {
	return func(ctx context.Context, d *schema.ResourceDiff, _ interface{}) error {
		minReplicas, ok1 := d.GetOk("spec.0.min_replicas")
		maxReplicas, ok2 := d.GetOk("spec.0.max_replicas")
		if ok1 && ok1 != ok2 {
			return fmt.Errorf("to configure autoscaler both min_replicas and max_replicas must be set")
		}
		if ok1 == false {
			return nil
		}

		if minReplicas.(int) > maxReplicas.(int) {
			return fmt.Errorf("min_replicas must be smaller than max_replicas")
		}

		replicas := 1
		if v, ok := d.GetOk("spec.0.replicas"); ok {
			replicas = v.(int)
		}
		if replicas > maxReplicas.(int) {
			return fmt.Errorf("max_replicas can't be smaller than replicas")
		}
		if replicas < minReplicas.(int) {
			return fmt.Errorf("min_replicas can't be bigger than replicas")
		}
		return nil
	}
}
