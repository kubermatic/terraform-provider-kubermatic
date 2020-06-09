package kubermatic

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/kubermatic/go-kubermatic/client/versions"
)

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
