package kubermatic

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

const (
	testNamePrefix           = "tf-acc-test-"
	testEnvOpenstackUsername = "KUBERMATIC_OPENSTACK_USERNAME"
	testEnvOpenstackPassword = "KUBERMATIC_OPENSTACK_PASSWORD"
	testEnvOpenstackTenant   = "KUBERMATIC_OPENSTACK_TENANT"
	testEnvOpenstackSeedDC   = "KUBERMATIC_OPENSTACK_SEED_DC"
	testEnvOpenstackNodeDC   = "KUBERMATIC_OPENSTACK_NODE_DC"
)

var (
	testAccProviders map[string]terraform.ResourceProvider
	testAccProvider  *schema.Provider
)

func init() {
	testAccProvider = Provider().(*schema.Provider)
	testAccProviders = map[string]terraform.ResourceProvider{
		"kubermatic": testAccProvider,
	}
}

func testAccPreCheckForOpenstack(t *testing.T) {
	t.Helper()
	testAccPreCheck(t)
	checkEnv(t, testEnvOpenstackUsername)
	checkEnv(t, testEnvOpenstackPassword)
	checkEnv(t, testEnvOpenstackTenant)
	checkEnv(t, testEnvOpenstackSeedDC)
	checkEnv(t, testEnvOpenstackNodeDC)
}

func testAccPreCheck(t *testing.T) {
	t.Helper()
	checkEnv(t, "KUBERMATIC_HOST")
	checkEnv(t, "KUBERMATIC_TOKEN")
}

func checkEnv(t *testing.T, n string) {
	t.Helper()
	if v := os.Getenv(n); v == "" {
		t.Fatalf("%s must be set for acceptance tests", n)
	}
}

func randomTestName() string {
	return randomName(testNamePrefix, 10)
}

func randomName(prefix string, length int) string {
	return fmt.Sprintf("%s%s", prefix, acctest.RandString(length))
}

func testResourceInstanceState(name string, check func(*terraform.InstanceState) error) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		m := s.RootModule()
		if rs, ok := m.Resources[name]; ok {
			is := rs.Primary
			if is == nil {
				return fmt.Errorf("No primary instance: %s", name)
			}

			return check(is)
		}
		return fmt.Errorf("Not found: %s", name)

	}
}
