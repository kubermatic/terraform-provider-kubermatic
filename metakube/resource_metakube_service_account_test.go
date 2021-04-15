package metakube

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/syseleven/terraform-provider-metakube/go-metakube/client/serviceaccounts"
	"github.com/syseleven/terraform-provider-metakube/go-metakube/models"
)

func TestAccMetakubeServiceAccount_Basic(t *testing.T) {
	var serviceAccount models.ServiceAccount
	testName := makeRandomString()
	resourceName := "metakube_service_account.acctest_sa"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckMetaKubeServiceAccountDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccMetaKubeServiceAccountBasic1, testName, testName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccMetaKubeServiceAccountExists(&serviceAccount),
					testResourceInstanceState(resourceName, func(is *terraform.InstanceState) error {
						if serviceAccount.Name != testName {
							return fmt.Errorf("want name=%s, got=%s", testName, serviceAccount.Name)
						}
						if !strings.Contains(serviceAccount.Group, "viewers") {
							return fmt.Errorf("want group=viewers*, got=%s", serviceAccount.Group)
						}

						if is.Attributes["name"] != testName {
							return fmt.Errorf("attribute 'name' expected value '%s', got: '%s'", testName, is.Attributes["name"])
						}
						if !strings.Contains(is.Attributes["group"], "viewers") {
							return fmt.Errorf("attribute 'group' expected starts with 'viewers', got '%s'", is.Attributes["group"])
						}
						return nil
					}),
				),
			},
			{
				Config: fmt.Sprintf(testAccMetaKubeServiceAccountBasic2, testName, testName+"edit"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testResourceInstanceState(resourceName, func(is *terraform.InstanceState) error {
						if is.ID != serviceAccount.ID {
							return fmt.Errorf("service account not updated: wrong ID")
						}
						return nil
					}),
					testAccMetaKubeServiceAccountExists(&serviceAccount),
					testResourceInstanceState(resourceName, func(is *terraform.InstanceState) error {
						if serviceAccount.Name != testName+"edit" {
							return fmt.Errorf("want name=%s, got=%s", testName+"edit", serviceAccount.Name)
						}
						if !strings.Contains(serviceAccount.Group, "editors") {
							return fmt.Errorf("want group=editors*, got=%s", serviceAccount.Group)
						}

						if is.Attributes["name"] != testName+"edit" {
							return fmt.Errorf("attribute 'name' expected value '%s', got: '%s'", testName+"edit", is.Attributes["name"])
						}
						if !strings.Contains(is.Attributes["group"], "editors") {
							return fmt.Errorf("attribute 'group' expected starts with 'editors', got '%s'", is.Attributes["group"])
						}
						return nil
					}),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Test importing non-existent resource provides expected error.
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: false,
				ImportStateId:     "123abc",
				ExpectError:       regexp.MustCompile(`(Please verify the ID is correct|Cannot import non-existent remote object)`),
			},
		},
	})
}

func testAccCheckMetaKubeServiceAccountDestroy(s *terraform.State) error {
	k := testAccProvider.Meta().(*metakubeProviderMeta)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "metakube_serviceaccount" {
			continue
		}
		p := serviceaccounts.NewListServiceAccountsParams()
		p.SetProjectID(rs.Primary.Attributes["project_id"])
		ret, err := k.client.Serviceaccounts.ListServiceAccounts(p, k.auth)
		if err != nil {
			return fmt.Errorf("list service account: %v", err)
		}
		for _, item := range ret.Payload {
			if item.ID == rs.Primary.ID {
				return fmt.Errorf("service accounts are not destroyed")
			}
		}
	}

	return nil
}

func testAccMetaKubeServiceAccountExists(serviceAccount *models.ServiceAccount) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		n := "metakube_service_account.acctest_sa"
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("record id is not set")
		}

		p := serviceaccounts.NewListServiceAccountsParams()
		p.SetProjectID(rs.Primary.Attributes["project_id"])
		k := testAccProvider.Meta().(*metakubeProviderMeta)
		r, err := k.client.Serviceaccounts.ListServiceAccounts(p, k.auth)
		if err != nil {
			return fmt.Errorf("ListServiceAccounts: %v", err)
		}

		for _, item := range r.Payload {
			if item.ID == rs.Primary.ID {
				*serviceAccount = *item
				return nil
			}
		}

		return fmt.Errorf("record not found")
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
