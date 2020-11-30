package metakube

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/syseleven/terraform-provider-metakube/go-metakube/client/serviceaccounts"
	"github.com/syseleven/terraform-provider-metakube/go-metakube/models"
)

func TestAccMetaKubeServiceAccount_Basic(t *testing.T) {
	var serviceAccount models.ServiceAccount
	testName := randomTestName()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckMetaKubeServiceAccountDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccMetaKubeServiceAccountBasic1, testName, testName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccMetaKubeServiceAccountExists(&serviceAccount),
					testResourceInstanceState("metakube_service_account.acctest_sa", func(is *terraform.InstanceState) error {
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
			// 	Config: fmt.Sprintf(testAccMetaKubeServiceAccountBasic2, testName, testName+"edit"),
			// 	Check: resource.ComposeAggregateTestCheckFunc(
			// 		testResourceInstanceState("metakube_service_account.acctest_sa", func(is *terraform.InstanceState) error {
			// 			_, id, err := metakubeServiceAccountParseID(is.ID)
			// 			if err != nil {
			// 				return err
			// 			}
			// 			if id != serviceAccount.ID {
			// 				return fmt.Errorf("service account not updated: wrong ID")
			// 			}
			// 			return nil
			// 		}),
			// 		testAccMetaKubeServiceAccountExists(&serviceAccount),
			// 		testResourceInstanceState("metakube_service_account.acctest_sa", func(is *terraform.InstanceState) error {
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

func testAccCheckMetaKubeServiceAccountDestroy(s *terraform.State) error {
	return nil
}

func testAccMetaKubeServiceAccountExists(serviceAccount *models.ServiceAccount) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		n := "metakube_service_account.acctest_sa"
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		projectID, serviceAccountID, err := metakubeServiceAccountParseID(rs.Primary.ID)
		if err != nil {
			return err
		}

		p := serviceaccounts.NewListServiceAccountsParams()
		p.SetProjectID(projectID)
		k := testAccProvider.Meta().(*metakubeProviderMeta)
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

const testAccMetaKubeServiceAccountBasic1 = `
resource "metakube_project" "acctest_project" {
	name = "%s"
}

resource "metakube_service_account" "acctest_sa" {
	project_id = metakube_project.acctest_project.id
	name = "%s"
	group = "viewers"
}
`

const testAccMetaKubeServiceAccountBasic2 = `
resource "metakube_project" "acctest_project" {
	name = "%s"
}

resource "metakube_service_account" "acctest_sa" {
	project_id = metakube_project.acctest_project.id
	name = "%s"
	group = "editors"
}
`
