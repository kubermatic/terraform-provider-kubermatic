package kubermatic

import (
	"fmt"

	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/kubermatic/go-kubermatic/client/project"
	"github.com/kubermatic/go-kubermatic/client/versions"
)

func getClusterVersion(id string, k *kubermaticProviderMeta) (*version.Version, error) {
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
		return nil, fmt.Errorf("unable to get cluster %s in project %s in seed dc %s", clusterID, projectID, seedDC)
	}

	v, err := version.NewVersion(r.Payload.Spec.Version.(string))
	if err != nil {
		return nil, err
	}
	return v, nil
}

func validateVersionAgainstCluster() schema.CustomizeDiffFunc {
	return func(d *schema.ResourceDiff, meta interface{}) error {
		dataVersion, ok := d.Get("spec.0.template.0.versions.0.kubelet").(string)
		if dataVersion == "" || !ok {
			return nil
		}

		v, err := version.NewVersion(dataVersion)

		if err != nil {
			return fmt.Errorf("unable to parse node deployment version")
		}

		clusterVersion, err := getClusterVersion(d.Get("cluster_id").(string), meta.(*kubermaticProviderMeta))
		if err != nil {
			return err
		}

		if clusterVersion.LessThan(v) {
			return fmt.Errorf("node deployment version (%s) cannot be greater than cluster version (%s)", v, clusterVersion)
		}
		return nil
	}
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
