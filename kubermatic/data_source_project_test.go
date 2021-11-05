package kubermatic

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"testing"
)

func TestAccKubermaticProjectDataSource(t *testing.T) {
	name := "data.kubermatic_project.foo"
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccKubermaticProjectDataSourceConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "name", "xxxxxxxx"),
				),
			},
		},
	})
}

const testAccKubermaticProjectDataSourceConfig = `
data "kubermatic_project" "foo" {
  name = "xxxxxxxx"
}
`
