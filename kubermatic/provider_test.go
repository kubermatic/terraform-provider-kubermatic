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
	testNamePrefix = "tf-acc-test-"

	testEnvOtherUserEmail = "KUBERMATIC_ANOTHER_USER_EMAIL"

	testEnvOpenstackNodeDC   = "KUBERMATIC_OPENSTACK_NODE_DC"
	testEnvOpenstackUsername = "KUBERMATIC_OPENSTACK_USERNAME"
	testEnvOpenstackPassword = "KUBERMATIC_OPENSTACK_PASSWORD"
	testEnvOpenstackTenant   = "KUBERMATIC_OPENSTACK_TENANT"
	testEnvOpenstackImage    = "KUBERMATIC_OPENSTACK_IMAGE"
	testEnvOpenstackImage2   = "KUBERMATIC_OPENSTACK_IMAGE2"
	testEnvOpenstackFlavor   = "KUBERMATIC_OPENSTACK_FLAVOR"

	testEnvAzureNodeDC         = "KUBERMATIC_AZURE_NODE_DC"
	testEnvAzureNodeSize       = "KUBERMATIC_AZURE_NODE_SIZE"
	testEnvAzureClientID       = "KUBERMATIC_AZURE_CLIENT_ID"
	testEnvAzureClientSecret   = "KUBERMATIC_AZURE_CLIENT_SECRET"
	testEnvAzureTenantID       = "KUBERMATIC_AZURE_TENANT_ID"
	testEnvAzureSubscriptionID = "KUBERMATIC_AZURE_SUBSCRIPTION_ID"

	testEnvAWSAccessKeyID      = "KUBERMATIC_AWS_ACCESS_KEY_ID"
	testAWSSecretAccessKey     = "KUBERMATIC_AWS_ACCESS_KEY_SECRET"
	testEnvAWSVPCID            = "KUBERMATIC_AWS_VPC_ID"
	testEnvAWSNodeDC           = "KUBERMATIC_AWS_NODE_DC"
	testEnvAWSInstanceType     = "KUBERMATIC_AWS_INSTANCE_TYPE"
	testEnvAWSSubnetID         = "KUBERMATIC_AWS_SUBNET_ID"
	testEnvAWSAvailabilityZone = "KUBERMATIC_AWS_AVAILABILITY_ZONE"
	testEnvAWSDiskSize         = "KUBERMATIC_AWS_DISK_SIZE"
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
	checkEnv(t, testEnvOpenstackNodeDC)
	checkEnv(t, testEnvOpenstackImage)
	checkEnv(t, testEnvOpenstackImage2)
	checkEnv(t, testEnvOpenstackFlavor)
}

func testAccPreCheckForAzure(t *testing.T) {
	t.Helper()
	testAccPreCheck(t)
	checkEnv(t, testEnvAzureClientID)
	checkEnv(t, testEnvAzureClientSecret)
	checkEnv(t, testEnvAzureSubscriptionID)
	checkEnv(t, testEnvAzureTenantID)
	checkEnv(t, testEnvAzureNodeDC)
	checkEnv(t, testEnvAzureNodeSize)
}

func testAccPreCheckForAWS(t *testing.T) {
	t.Helper()
	testAccPreCheck(t)
	checkEnv(t, testEnvAWSAccessKeyID)
	checkEnv(t, testAWSSecretAccessKey)
	checkEnv(t, testEnvAWSVPCID)
	checkEnv(t, testEnvAWSNodeDC)
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
