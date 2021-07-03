package kubermatic

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"testing"
)

func TestAccKubermaticClusterDataSource(t *testing.T) {
	name := "data.kubermatic_cluster.foo"
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccKubermaticClusterDataSourceConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "project_id", "xxxxxxxx"),
					resource.TestCheckResourceAttr(name, "cluster_id", "yyyyyyyy"),
				),
			},
		},
	})
}

const testAccKubermaticClusterDataSourceConfig = `
data "kubermatic_cluster" "acctest_cluster" {
  project_id = "xxxxxxxx"
  cluster_id = "yyyyyyyy"
}
`
