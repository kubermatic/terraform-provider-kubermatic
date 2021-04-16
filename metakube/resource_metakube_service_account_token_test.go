package metakube

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/syseleven/go-metakube/client/tokens"
	"github.com/syseleven/go-metakube/models"
)

func TestAccMetakubeServiceAccountToken_Basic(t *testing.T) {
	var token models.PublicServiceAccountToken
	testName := makeRandomString()
	resourceName := "metakube_service_account_token.acctest_sa_token"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckMetaKubeServiceAccountTokenDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccMetaKubeServiceAccountTokenBasic, testName, testName, testName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccMetaKubeServiceAccountTokenExists(&token),
					resource.TestCheckResourceAttrSet(resourceName, "token"),
					resource.TestCheckResourceAttr(resourceName, "name", testName),
					resource.TestCheckResourceAttrPtr(resourceName, "name", &token.Name),
					resource.TestCheckResourceAttrSet(resourceName, "creation_timestamp"),
					resource.TestCheckResourceAttrSet(resourceName, "expiry"),
				),
			},
			{
				Config: fmt.Sprintf(testAccMetaKubeServiceAccountTokenBasic, testName, testName, testName+"edit"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccMetaKubeServiceAccountTokenExists(&token),
					resource.TestCheckResourceAttrSet(resourceName, "token"),
					resource.TestCheckResourceAttr(resourceName, "name", testName+"edit"),
					resource.TestCheckResourceAttrPtr(resourceName, "name", &token.Name),
				),
			},
		},
	})
}

func testAccCheckMetaKubeServiceAccountTokenDestroy(s *terraform.State) error {
	token, err := testAccMetaKubeServiceAccountFetchToken(s)
	if err != nil {
		if e, ok := err.(*tokens.ListServiceAccountTokensDefault); ok && e.Code() == http.StatusNotFound {
			return nil
		}
		return err
	}
	if token != nil {
		return fmt.Errorf("record not deleted")
	}
	return nil
}

func testAccMetaKubeServiceAccountTokenExists(token *models.PublicServiceAccountToken) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		v, err := testAccMetaKubeServiceAccountFetchToken(s)
		if err != nil {
			return err
		}
		if v != nil {
			*token = *v
			return nil
		}
		return fmt.Errorf("no record created")
	}
}

func testAccMetaKubeServiceAccountFetchToken(s *terraform.State) (*models.PublicServiceAccountToken, error) {
	n := "metakube_service_account_token.acctest_sa_token"
	rs, ok := s.RootModule().Resources[n]
	if !ok {
		return nil, fmt.Errorf("not found: %s", n)
	}
	if rs.Primary.ID == "" {
		return nil, fmt.Errorf("record id is not set")
	}

	k := testAccProvider.Meta().(*metakubeProviderMeta)

	p := tokens.NewListServiceAccountTokensParams()
	p.SetProjectID(rs.Primary.Attributes["project_id"])
	p.SetServiceAccountID(rs.Primary.Attributes["service_account_id"])
	r, err := k.client.Tokens.ListServiceAccountTokens(p, k.auth)
	if err != nil {
		if _, ok := err.(*tokens.ListServiceAccountTokensForbidden); ok {
			return nil, nil
		}
		return nil, err
	}
	for _, v := range r.Payload {
		if v.ID == rs.Primary.ID {
			return v, nil
		}
	}

	return nil, nil
}

const testAccMetaKubeServiceAccountTokenBasic = `
resource "metakube_project" "acctest_project" {
	name = "%s"
}

resource "metakube_service_account" "acctest_sa" {
	project_id = metakube_project.acctest_project.id
	name = "%s"
	group = "viewers"
}

resource "metakube_service_account_token" "acctest_sa_token" {
	service_account_id = metakube_service_account.acctest_sa.id
	name = "%s"
}`
