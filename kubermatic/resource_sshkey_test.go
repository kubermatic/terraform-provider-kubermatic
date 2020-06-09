package kubermatic

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/kubermatic/go-kubermatic/client/project"
	"github.com/kubermatic/go-kubermatic/models"
)

func TestAccKubermaticSSHKey_Basic(t *testing.T) {
	var sshkey models.SSHKey
	testName := randomTestName()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubermaticSSHKeyDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccCheckKubermaticSSHKeyConfigBasic, testName, testName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubermaticSSHKeyExists("kubermatic_sshkey.acctest_sshkey", "kubermatic_project.acctest_project", &sshkey),
					testAccCheckKubermaticSSHKeyAttributes(&sshkey, testName),
					resource.TestCheckResourceAttr("kubermatic_sshkey.acctest_sshkey", "name", testName),
					resource.TestCheckResourceAttr("kubermatic_sshkey.acctest_sshkey", "public_key", testSSHPubKey),
				),
			},
		},
	})
}

const (
	testSSHPubKey                           = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCzoO6BIidD4Us9a9Kh0GzaUUxosl61GNUZzqcIdmf4EYZDdRtLa+nu88dHPHPQ2dj52BeVV9XVN9EufqdAZCaKpPLj5XxEwMpGcmdrOAl38kk2KKbiswjXkrdhYSBw3w0KkoCPKG/+yNpAUI9z+RJZ9lukeYBvxdDe8nuvUWX7mGRaPaumCpQaBHwYKNn6jMVns2RrumgE9w+Z6jlaKHk1V7T5rCBDcjXwcy6waOX6hKdPPBk84FpUfcfN/SdpwSVGFrcykazrpmzD2nYr71EcOm9T6/yuhBOiIa3H/TOji4G9wr02qtSWuGUpULkqWMFD+BQcYQQA71GSAa+rTZuf user@machine.local"
	testAccCheckKubermaticSSHKeyConfigBasic = `
resource "kubermatic_project" "acctest_project" {
	name = "%s"
	labels = {}
}

resource "kubermatic_sshkey" "acctest_sshkey" {
	project_id = kubermatic_project.acctest_project.id

	name = "%s"
	public_key = "` + testSSHPubKey + `"
}
`
)

func testAccCheckKubermaticSSHKeyDestroy(s *terraform.State) error {
	k := testAccProvider.Meta().(*kubermaticProviderMeta)

	// Check all ssh keys from all projects.
	for _, rsPrj := range s.RootModule().Resources {
		if rsPrj.Type != "kubermatic_project" {
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
			if rs.Type != "kubermatic_sshkey" {
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

func testAccCheckKubermaticSSHKeyExists(sshkeyN, projectN string, rec *models.SSHKey) resource.TestCheckFunc {
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

		k := testAccProvider.Meta().(*kubermaticProviderMeta)
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

func testAccCheckKubermaticSSHKeyAttributes(rec *models.SSHKey, name string) resource.TestCheckFunc {
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
