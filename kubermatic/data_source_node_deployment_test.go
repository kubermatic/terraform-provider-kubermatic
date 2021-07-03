package kubermatic

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"testing"
)

func TestAccKubermaticNodeDeploymentDataSource(t *testing.T) {
	name := "data.kubermatic_node_deployment.foo"
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccKubermaticNodeDeploymentDataSourceConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "project_id", "wwwwwwww"),
					resource.TestCheckResourceAttr(name, "cluster_id", "xxxxxxxx"),
					resource.TestCheckResourceAttr(name, "dc_name", "yyyyyyyy"),
					resource.TestCheckResourceAttr(name, "id", "zzzzzzzz"),
				),
			},
		},
	})
}

const testAccKubermaticNodeDeploymentDataSourceConfig = `
data "kubermatic_cluster" "acctest_cluster" {
  project_id = "wwwwwwww"
  cluster_id = "xxxxxxxx"
  dc_name    = "yyyyyyyy"
  id         = "zzzzzzzz"
}
`
