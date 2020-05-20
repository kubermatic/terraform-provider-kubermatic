package kubermatic

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

const testNamePrefix = "tf-acc-test-"

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

func testAccPreCheck(t *testing.T) {
	if v := os.Getenv("KUBERMATIC_HOST"); v == "" {
		t.Fatal("KUBERMATIC_HOST must be set for acceptance tests")
	}
	if v := os.Getenv("KUBERMATIC_TOKEN"); v == "" {
		t.Fatal("KUBERMATIC_TOKEN must be set for acceptance tests")
	}
}

func randomTestName() string {
	return randomName(testNamePrefix, 10)
}

func randomName(prefix string, length int) string {
	return fmt.Sprintf("%s%s", prefix, acctest.RandString(length))
}
