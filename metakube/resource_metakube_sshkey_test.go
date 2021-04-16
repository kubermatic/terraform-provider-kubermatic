package metakube

import (
	"fmt"
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/syseleven/go-metakube/client/project"
	"github.com/syseleven/go-metakube/models"
)

func TestAccMetakubeSSHKey_Basic(t *testing.T) {
	var sshkey models.SSHKey
	testName := makeRandomString()
	resourceName := "metakube_sshkey.acctest_sshkey"
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckMetaKubeSSHKeyDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccCheckMetaKubeSSHKeyConfigBasic, testName, testName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckMetaKubeSSHKeyExists(resourceName, "metakube_project.acctest_project", &sshkey),
					testAccCheckMetaKubeSSHKeyAttributes(&sshkey, testName),
					resource.TestCheckResourceAttr(resourceName, "name", testName),
					resource.TestCheckResourceAttr(resourceName, "public_key", testSSHPubKey),
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

const (
	testSSHPubKey                         = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCzoO6BIidD4Us9a9Kh0GzaUUxosl61GNUZzqcIdmf4EYZDdRtLa+nu88dHPHPQ2dj52BeVV9XVN9EufqdAZCaKpPLj5XxEwMpGcmdrOAl38kk2KKbiswjXkrdhYSBw3w0KkoCPKG/+yNpAUI9z+RJZ9lukeYBvxdDe8nuvUWX7mGRaPaumCpQaBHwYKNn6jMVns2RrumgE9w+Z6jlaKHk1V7T5rCBDcjXwcy6waOX6hKdPPBk84FpUfcfN/SdpwSVGFrcykazrpmzD2nYr71EcOm9T6/yuhBOiIa3H/TOji4G9wr02qtSWuGUpULkqWMFD+BQcYQQA71GSAa+rTZuf user@machine.local"
	testAccCheckMetaKubeSSHKeyConfigBasic = `
resource "metakube_project" "acctest_project" {
	name = "%s"
	labels = {}
}

resource "metakube_sshkey" "acctest_sshkey" {
	project_id = metakube_project.acctest_project.id

	name = "%s"
	public_key = "` + testSSHPubKey + `"
}
`
)

func testAccCheckMetaKubeSSHKeyDestroy(s *terraform.State) error {
	k := testAccProvider.Meta().(*metakubeProviderMeta)

	// Check all ssh keys from all projects.
	for _, rsPrj := range s.RootModule().Resources {
		if rsPrj.Type != "metakube_project" {
			continue
		}

		p := project.NewListSSHKeysParams()
		p.SetProjectID(rsPrj.Primary.ID)
		sshkeys, err := k.client.Project.ListSSHKeys(p, k.auth)
		if err != nil {
			// API returns 403 if project doesn't exist.
			if _, ok := err.(*project.ListSSHKeysForbidden); ok {
				continue
			}
			if e, ok := err.(*project.ListSSHKeysDefault); ok && e.Code() == http.StatusNotFound {
				continue
			}
			return fmt.Errorf("check destroy: %v", err)
		}

		for _, rs := range s.RootModule().Resources {
			if rs.Type != "metakube_sshkey" {
				continue
			}

			// Try to find sshkey
			for _, r := range sshkeys.Payload {
				if r.ID == rs.Primary.ID {
					return fmt.Errorf("SSHKey still exists")
				}
			}
		}
	}

	return nil
}

func testAccCheckMetaKubeSSHKeyExists(sshkeyN, projectN string, rec *models.SSHKey) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[sshkeyN]

		if !ok {
			return fmt.Errorf("Not found: %s", sshkeyN)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		rsPrj, ok := s.RootModule().Resources[projectN]

		if !ok {
			return fmt.Errorf("Not found: %s", sshkeyN)
		}

		if rsPrj.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		k := testAccProvider.Meta().(*metakubeProviderMeta)
		p := project.NewListSSHKeysParams()
		p.SetProjectID(rsPrj.Primary.ID)

		ret, err := k.client.Project.ListSSHKeys(p, k.auth)
		if err != nil {
			return fmt.Errorf("Cannot verify record exist, list sshkeys error: %v", err)
		}

		for _, r := range ret.Payload {
			if r.ID == rs.Primary.ID {
				*rec = *r
				return nil
			}
		}

		return fmt.Errorf("Record not found")
	}
}

func testAccCheckMetaKubeSSHKeyAttributes(rec *models.SSHKey, name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if rec.Name != name {
			return fmt.Errorf("want SSHKey.Name=%s, got %s", name, rec.Name)
		}

		if rec.Spec.PublicKey != testSSHPubKey {
			return fmt.Errorf("want SSHKey.PublicKey=%s, got %s", testSSHPubKey, rec.Spec.PublicKey)
		}

		return nil
	}
}
