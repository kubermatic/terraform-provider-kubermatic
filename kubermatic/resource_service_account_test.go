package kubermatic

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/kubermatic/go-kubermatic/client/serviceaccounts"
	"github.com/kubermatic/go-kubermatic/models"
)

func TestAccKubermaticServiceAccount_Basic(t *testing.T) {
	var serviceAccount models.ServiceAccount
	testName := randomTestName()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubermaticServiceAccountDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccKubermaticServiceAccountBasic1, testName, testName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccKubermaticServiceAccountExists(&serviceAccount),
					testResourceInstanceState("kubermatic_service_account.acctest_sa", func(is *terraform.InstanceState) error {
						if serviceAccount.Name != testName {
							return fmt.Errorf("Want Name=%s, got=%s", testName, serviceAccount.Name)
						}
						if !strings.Contains(serviceAccount.Group, "viewers") {
							return fmt.Errorf("Want Group=viewers*, got=%s", serviceAccount.Group)
						}

						if is.Attributes["name"] != testName {
							return fmt.Errorf("Attribute 'name' expected value '%s', got: '%s'", testName, is.Attributes["name"])
						}
						if !strings.Contains(is.Attributes["group"], "viewers") {
							return fmt.Errorf("Attribute 'group' expected starts with 'viewers', got '%s'", is.Attributes["group"])
						}
						return nil
					}),
				),
			},
			// TODO(furkhat): uncomment when fix to `assignment to entry in nil map` released.
			// {
			// 	Config: fmt.Sprintf(testAccKubermaticServiceAccountBasic2, testName, testName+"edit"),
			// 	Check: resource.ComposeAggregateTestCheckFunc(
			// 		testResourceInstanceState("kubermatic_service_account.acctest_sa", func(is *terraform.InstanceState) error {
			// 			_, id, err := kubermaticServiceAccountParseID(is.ID)
			// 			if err != nil {
			// 				return err
			// 			}
			// 			if id != serviceAccount.ID {
			// 				return fmt.Errorf("service account not updated: wrong ID")
			// 			}
			// 			return nil
			// 		}),
			// 		testAccKubermaticServiceAccountExists(&serviceAccount),
			// 		testResourceInstanceState("kubermatic_service_account.acctest_sa", func(is *terraform.InstanceState) error {
			// 			if serviceAccount.Name != testName {
			// 				return fmt.Errorf("Want Name=%s, got=%s", testName, serviceAccount.Name)
			// 			}
			// 			if !strings.Contains(serviceAccount.Group, "editors") {
			// 				return fmt.Errorf("Want Group=editors*, got=%s", serviceAccount.Group)
			// 			}

			// 			if is.Attributes["name"] != testName+"edit" {
			// 				return fmt.Errorf("Attribute 'name' expected value '%s', got: '%s'", testName+"edit", is.Attributes["name"])
			// 			}
			// 			if !strings.Contains(is.Attributes["group"], "editors") {
			// 				return fmt.Errorf("Attribute 'group' expected starts with 'editors', got '%s'", is.Attributes["group"])
			// 			}
			// 			return nil
			// 		}),
			// 	),
			// },
		},
	})
}

func testAccCheckKubermaticServiceAccountDestroy(s *terraform.State) error {
	return nil
}

func testAccKubermaticServiceAccountExists(serviceAccount *models.ServiceAccount) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		n := "kubermatic_service_account.acctest_sa"
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		projectID, serviceAccountID, err := kubermaticServiceAccountParseID(rs.Primary.ID)
		if err != nil {
			return err
		}

		p := serviceaccounts.NewListServiceAccountsParams()
		p.SetProjectID(projectID)
		k := testAccProvider.Meta().(*kubermaticProviderMeta)
		r, err := k.client.Serviceaccounts.ListServiceAccounts(p, k.auth)
		if err != nil {
			return fmt.Errorf("ListServiceAccounts: %v", err)
		}

		for _, item := range r.Payload {
			if item.ID == serviceAccountID {
				*serviceAccount = *item
				return nil
			}
		}

		return fmt.Errorf("Record not found")
	}
}

const testAccKubermaticServiceAccountBasic1 = `
resource "kubermatic_project" "acctest_project" {
	name = "%s"
}

resource "kubermatic_service_account" "acctest_sa" {
	project_id = kubermatic_project.acctest_project.id
	name = "%s"
	group = "viewers"
}
`

const testAccKubermaticServiceAccountBasic2 = `
resource "kubermatic_project" "acctest_project" {
	name = "%s"
}

resource "kubermatic_service_account" "acctest_sa" {
	project_id = kubermatic_project.acctest_project.id
	name = "%s"
	group = "editors"
}
`
