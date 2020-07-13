package kubermatic

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/kubermatic/go-kubermatic/client/tokens"
	"github.com/kubermatic/go-kubermatic/models"
)

func TestAccKubermaticToken_Basic(t *testing.T) {
	var token models.PublicServiceAccountToken
	testName := randomTestName()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubermaticServiceAccountTokenDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccKubermaticServiceAccountTokenBasic, testName, testName, testName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccKubermaticServiceAccountTokenExists(&token),
					resource.TestCheckResourceAttrSet("kubermatic_service_account_token.acctest_sa_token", "token"),
					resource.TestCheckResourceAttr("kubermatic_service_account_token.acctest_sa_token", "name", testName),
					resource.TestCheckResourceAttrPtr("kubermatic_service_account_token.acctest_sa_token", "name", &token.Name),
					resource.TestCheckResourceAttrSet("kubermatic_service_account_token.acctest_sa_token", "creation_timestamp"),
					resource.TestCheckResourceAttrSet("kubermatic_service_account_token.acctest_sa_token", "expiry"),
				),
			},
			// TODO(furkhat): Fix go-kubermatic client PatchServiceAccountTokenParams structure
			// {
			// 	Config: fmt.Sprintf(testAccKubermaticServiceAccountTokenBasic, testName, testName, testName+"edit"),
			// 	Check: resource.ComposeAggregateTestCheckFunc(
			// 		testAccKubermaticServiceAccountTokenExists(&token),
			// 		resource.TestCheckResourceAttrSet("kubermatic_service_account_token.acctest_sa_token", "token"),
			// 		resource.TestCheckResourceAttr("kubermatic_service_account_token.acctest_sa_token", "name", testName+"edit"),
			// 		resource.TestCheckResourceAttrPtr("kubermatic_service_account_token.acctest_sa_token", "name", &token.Name),
			// 	),
			// },
		},
	})
}

func testAccCheckKubermaticServiceAccountTokenDestroy(s *terraform.State) error {
	token, err := testAccKubermaticServiceAccountFetchToken(s)
	if err != nil {
		if e, ok := err.(*tokens.ListServiceAccountTokensDefault); ok && e.Code() == http.StatusNotFound {
			return nil
		}
		return err
	}
	if token != nil {
		return fmt.Errorf("Record not deleted")
	}
	return nil
}

func testAccKubermaticServiceAccountTokenExists(token *models.PublicServiceAccountToken) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		v, err := testAccKubermaticServiceAccountFetchToken(s)
		if err != nil {
			return err
		}
		if v != nil {
			*token = *v
			return nil
		}
		return fmt.Errorf("No Record created")
	}
}

func testAccKubermaticServiceAccountFetchToken(s *terraform.State) (*models.PublicServiceAccountToken, error) {
	n := "kubermatic_service_account_token.acctest_sa_token"
	rs, ok := s.RootModule().Resources[n]
	if !ok {
		return nil, fmt.Errorf("Not found: %s", n)
	}
	if rs.Primary.ID == "" {
		return nil, fmt.Errorf("No Record ID is set")
	}

	projectID, serviceAccountID, tokenID, err := kubermaticServiceAccountTokenParseID(rs.Primary.ID)
	if err != nil {
		return nil, err
	}

	k := testAccProvider.Meta().(*kubermaticProviderMeta)

	p := tokens.NewListServiceAccountTokensParams()
	p.SetProjectID(projectID)
	p.SetServiceAccountID(serviceAccountID)
	r, err := k.client.Tokens.ListServiceAccountTokens(p, k.auth)
	if err != nil {
		if _, ok := err.(*tokens.ListServiceAccountTokensForbidden); ok {
			return nil, nil
		}
		return nil, err
	}
	for _, v := range r.Payload {
		if v.ID == tokenID {
			return v, nil
		}
	}

	return nil, nil
}

const testAccKubermaticServiceAccountTokenBasic = `
resource "kubermatic_project" "acctest_project" {
	name = "%s"
}

resource "kubermatic_service_account" "acctest_sa" {
	project_id = kubermatic_project.acctest_project.id
	name = "%s"
	group = "viewers"
}

resource "kubermatic_service_account_token" "acctest_sa_token" {
	service_account_id = kubermatic_service_account.acctest_sa.id
	name = "%s"
}`
