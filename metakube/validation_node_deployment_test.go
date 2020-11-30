package metakube

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

func TestAccMetaKubeNodeDeployment_ValidationAgainstCluster(t *testing.T) {
	testName := randomTestName()

	accessKeyID := os.Getenv(testEnvAWSAccessKeyID)
	accessKeySecret := os.Getenv(testAWSSecretAccessKey)
	vpcID := os.Getenv(testEnvAWSVPCID)
	nodeDC := os.Getenv(testEnvAWSNodeDC)
	k8sVersion17 := os.Getenv(testEnvK8sVersion)
	instanceType := os.Getenv(testEnvAWSInstanceType)
	subnetID := os.Getenv(testEnvAWSSubnetID)
	availabilityZone := os.Getenv(testEnvAWSAvailabilityZone)
	diskSize := os.Getenv(testEnvAWSDiskSize)

	unavailableVersion := "1.12.1"
	bigVersion := "3.0.0"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheckForAWS(t)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckMetaKubeClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckMetaKubeNodeDeploymentBasicValidation(testName, accessKeyID, accessKeySecret, vpcID, nodeDC, instanceType, subnetID, availabilityZone, diskSize, k8sVersion17, k8sVersion17),
			},
			{
				Config:      testAccCheckMetaKubeNodeDeploymentBasicValidation(testName, accessKeyID, accessKeySecret, vpcID, nodeDC, instanceType, subnetID, availabilityZone, diskSize, k8sVersion17, unavailableVersion),
				ExpectError: regexp.MustCompile(fmt.Sprintf(`unknown version for node deployment %s, available versions`, unavailableVersion)),
			},
			{
				Config:      testAccCheckMetaKubeNodeDeploymentTypeValidation(testName, accessKeyID, accessKeySecret, vpcID, nodeDC, k8sVersion17, k8sVersion17),
				ExpectError: regexp.MustCompile(`provider for node deployment must \(.*\) match cluster provider \(.*\)`),
			},
			{
				Config:      testAccCheckMetaKubeNodeDeploymentBasicValidation(testName, accessKeyID, accessKeySecret, vpcID, nodeDC, instanceType, subnetID, availabilityZone, diskSize, k8sVersion17, bigVersion),
				ExpectError: regexp.MustCompile(`cannot be greater than cluster version`),
			},
		},
	})
}

func testAccCheckMetaKubeNodeDeploymentBasicValidation(n, keyID, keySecret, vpcID, nodeDC, instanceType, subnetID, availabilityZone, diskSize, k8sVersion, kubeletVersion string) string {
	return fmt.Sprintf(`
	resource "metakube_project" "acctest_project" {
		name = "%s"
	}

	resource "metakube_cluster" "acctest_cluster" {
		name = "%s"
		dc_name = "%s"
		project_id = metakube_project.acctest_project.id

		spec {
			version = "%s"
			cloud {
				aws {
					access_key_id = "%s"
					secret_access_key = "%s"
					vpc_id = "%s"
				}
			}
		}
	}

	resource "metakube_node_deployment" "acctest_nd" {
		cluster_id = metakube_cluster.acctest_cluster.id
		name = "%s"
		spec {
			replicas = 1
			template {
				cloud {
					aws {
						instance_type = "%s"
						disk_size = "%s"
						volume_type = "standard"
						subnet_id = "%s"
						availability_zone = "%s"
						assign_public_ip = true
					}
				}
				operating_system {
					ubuntu {
						dist_upgrade_on_boot = false
					}
				}
				versions {
					kubelet = "%s"
				}
			}
		}
	}`, n, n, nodeDC, k8sVersion, keyID, keySecret, vpcID, n, instanceType, diskSize, subnetID, availabilityZone, kubeletVersion)
}

func testAccCheckMetaKubeNodeDeploymentTypeValidation(n, keyID, keySecret, vpcID, nodeDC, k8sVersion, kubeletVersion string) string {
	return fmt.Sprintf(`
	resource "metakube_project" "acctest_project" {
		name = "%s"
	}

	resource "metakube_cluster" "acctest_cluster" {
		name = "%s"
		dc_name = "%s"
		project_id = metakube_project.acctest_project.id

		spec {
			version = "%s"
			cloud {
				aws {
					access_key_id = "%s"
					secret_access_key = "%s"
					vpc_id = "%s"
				}
			}
		}
	}

	resource "metakube_node_deployment" "acctest_nd" {
		cluster_id = metakube_cluster.acctest_cluster.id
		name = "%s"
		spec {
			replicas = 1
			template {
				cloud {
					azure {
						size = 2
					}
				}
				operating_system {
					ubuntu {
						dist_upgrade_on_boot = false
					}
				}
				versions {
					kubelet = "%s"
				}
			}
		}
	}`, n, n, nodeDC, k8sVersion, keyID, keySecret, vpcID, n, kubeletVersion)
}
