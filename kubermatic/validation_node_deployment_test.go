package kubermatic

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

func TestAccKubermaticNodeDeployment_ValidationAgainstCluster(t *testing.T) {
	testName := randomTestName()

	username := os.Getenv(testEnvOpenstackUsername)
	password := os.Getenv(testEnvOpenstackPassword)
	tenant := os.Getenv(testEnvOpenstackTenant)
	nodeDC := os.Getenv(testEnvOpenstackNodeDC)
	image := os.Getenv(testEnvOpenstackImage)
	flavor := os.Getenv(testEnvOpenstackFlavor)

	k8sVersion17 := os.Getenv(testEnvK8sVersion17)
	kubeletVersion16 := os.Getenv(testEnvK8sVersion16)
	unavailableVersion := "1.12.1"
	bigVersion := "3.0.0"

	existingClusterID := os.Getenv(testEnvExistingClusterID)

	azure := `
		azure {
		size = "2"
	}`
	openstack := fmt.Sprintf(`
		openstack {
	  		flavor = "%s"
	  		image = "%s"
		  }`, flavor, image)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheckForOpenstack(t)
			testAccPreCheckExistingCluster(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubermaticNodeDeploymentDestroy,
		Steps: []resource.TestStep{
			{
				PlanOnly:           true,
				Config:             testAccCheckKubermaticNodeDeploymentBasic(testName, nodeDC, username, password, tenant, k8sVersion17, kubeletVersion16, image, flavor),
				ExpectNonEmptyPlan: true,
			},
			{
				PlanOnly:    true,
				Config:      testAccCheckKubermaticNodeDeploymentBasicValidation(existingClusterID, testName, kubeletVersion16, azure),
				ExpectError: regexp.MustCompile(`provider for node deployment must \(.*\) match cluster provider \(.*\)`),
			},
			{
				PlanOnly:    true,
				Config:      testAccCheckKubermaticNodeDeploymentBasicValidation(existingClusterID, testName, bigVersion, azure),
				ExpectError: regexp.MustCompile(`cannot be greater than cluster version`),
			},
			{
				PlanOnly:    true,
				Config:      testAccCheckKubermaticNodeDeploymentBasicValidation(existingClusterID, testName, unavailableVersion, openstack),
				ExpectError: regexp.MustCompile(fmt.Sprintf(`unknown version for node deployment %s, available versions`, unavailableVersion)),
			},
		},
	})
}

func testAccCheckKubermaticNodeDeploymentBasicValidation(clusterID, testName, kubeletVersion, provider string) string {
	return fmt.Sprintf(`
	resource "kubermatic_node_deployment" "acctest_nd" {
		cluster_id = "%s"
		name = "%s"
		spec {
			replicas = 1
			template {
				cloud {
					%s
				}
				operating_system {
					ubuntu {}
				}
				versions {
					kubelet = "%s"
				}
			}
		}
	}`, clusterID, testName, provider, kubeletVersion)
}
