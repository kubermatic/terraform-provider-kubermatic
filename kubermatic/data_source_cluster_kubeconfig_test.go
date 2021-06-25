package kubermatic

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"testing"
)

func TestAccKubermaticClusterKubeconfigDataSource(t *testing.T) {
	name := "data.kubermatic_cluster_kubeconfig.acctest_cluster"
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccKubermaticClusterKubeconfigDataSourceConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "project_id", "xxxxxxxx"),
					resource.TestCheckResourceAttr(name, "cluster_id", "yyyyyyyy"),
				),
			},
		},
	})
}

const testAccKubermaticClusterKubeconfigDataSourceConfig = `
data "kubermatic_cluster_kubeconfig" "acctest_cluster" {
  project_id = "xxxxxxxx"
  cluster_id = "yyyyyyyy"
}
`
