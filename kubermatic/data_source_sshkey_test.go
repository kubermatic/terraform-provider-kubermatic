package kubermatic

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"testing"
)

func TestAccKubermaticSSHKeyDataSource(t *testing.T) {
	name := "data.kubermatic_sshkey.foo"
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccKubermaticSSHKeyDataSourceConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "project_id", "xxxxxxxx"),
					resource.TestCheckResourceAttr(name, "name", "yyyyyyyy"),
				),
			},
		},
	})
}

const testAccKubermaticSSHKeyDataSourceConfig = `
data "kubermatic_sshkey" "foo" {
  project_id = "xxxxxxxx"
  name       = "yyyyyyyy"
}
`
