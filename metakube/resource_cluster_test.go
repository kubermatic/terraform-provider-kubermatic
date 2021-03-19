package metakube

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/syseleven/terraform-provider-metakube/go-metakube/client/project"
	"github.com/syseleven/terraform-provider-metakube/go-metakube/models"
)

func TestAccMetaKubeCluster_Openstack_Basic(t *testing.T) {
	var cluster models.Cluster
	testName := randomTestName()

	username := os.Getenv(testEnvOpenstackUsername)
	password := os.Getenv(testEnvOpenstackPassword)
	tenant := os.Getenv(testEnvOpenstackTenant)
	nodeDC := os.Getenv(testEnvOpenstackNodeDC)
	versionK8s17 := os.Getenv(testEnvK8sVersion)
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheckForOpenstack(t)
			checkEnv(t, "OS_AUTH_URL")
			checkEnv(t, "OS_USERNAME")
			checkEnv(t, "OS_PASSWORD")
		},
		Providers: testAccProviders,
		ExternalProviders: map[string]resource.ExternalProvider{
			"openstack": {
				Source: "terraform-provider-openstack/openstack",
			},
		},
		CheckDestroy: testAccCheckMetaKubeClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckMetaKubeClusterOpenstackBasic(testName, username, password, tenant, nodeDC, versionK8s17),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckMetaKubeClusterExists(&cluster),
					testAccCheckMetaKubeClusterOpenstackAttributes(&cluster, testName, username, password, tenant, nodeDC, versionK8s17, nil, false),
					resource.TestCheckResourceAttr("metakube_cluster.acctest_cluster", "dc_name", nodeDC),
					resource.TestCheckResourceAttr("metakube_cluster.acctest_cluster", "name", testName),
					resource.TestCheckResourceAttr("metakube_cluster.acctest_cluster", "labels.%", "0"),
					resource.TestCheckResourceAttr("metakube_cluster.acctest_cluster", "spec.#", "1"),
					resource.TestCheckResourceAttr("metakube_cluster.acctest_cluster", "spec.0.version", versionK8s17),
					resource.TestCheckResourceAttr("metakube_cluster.acctest_cluster", "spec.0.domain_name", "foodomain.local"),
					resource.TestCheckResourceAttr("metakube_cluster.acctest_cluster", "spec.0.services_cidr", "10.240.16.0/18"),
					resource.TestCheckResourceAttr("metakube_cluster.acctest_cluster", "spec.0.pods_cidr", "172.25.0.0/18"),
					resource.TestCheckResourceAttr("metakube_cluster.acctest_cluster", "spec.0.cloud.#", "1"),
					resource.TestCheckResourceAttr("metakube_cluster.acctest_cluster", "spec.0.cloud.0.bringyourown.#", "0"),
					resource.TestCheckResourceAttr("metakube_cluster.acctest_cluster", "spec.0.cloud.0.aws.#", "0"),
					resource.TestCheckResourceAttr("metakube_cluster.acctest_cluster", "spec.0.cloud.0.openstack.#", "1"),
					resource.TestCheckResourceAttr("metakube_cluster.acctest_cluster", "spec.0.cloud.0.openstack.0.tenant", tenant),
					resource.TestCheckResourceAttr("metakube_cluster.acctest_cluster", "spec.0.cloud.0.openstack.0.username", username),
					resource.TestCheckResourceAttr("metakube_cluster.acctest_cluster", "spec.0.cloud.0.openstack.0.password", password),
					resource.TestCheckResourceAttrSet("metakube_cluster.acctest_cluster", "spec.0.cloud.0.openstack.0.security_group"),
					resource.TestCheckResourceAttrSet("metakube_cluster.acctest_cluster", "spec.0.cloud.0.openstack.0.network"),
					resource.TestCheckResourceAttrSet("metakube_cluster.acctest_cluster", "spec.0.cloud.0.openstack.0.subnet_id"),
					resource.TestCheckResourceAttr("metakube_cluster.acctest_cluster", "spec.0.cloud.0.openstack.0.subnet_cidr", "192.168.2.0/24"),
					// Test spec.0.machine_networks value
					testResourceInstanceState("metakube_cluster.acctest_cluster", func(is *terraform.InstanceState) error {
						n, err := strconv.Atoi(is.Attributes["spec.0.machine_networks.#"])
						if err != nil {
							return err
						}

						if want := len(cluster.Spec.MachineNetworks); n != want {
							return fmt.Errorf("want len(cluster.Spec.MachineNetworks)=%d, got %d", want, n)
						}

						for i, networks := range cluster.Spec.MachineNetworks {
							prefix := fmt.Sprintf("spec.0.machine_networks.%d.", i)

							var k string

							k = prefix + "cidr"
							if v := is.Attributes[k]; v != networks.CIDR {
								return fmt.Errorf("want %s=%s, got %s", k, networks.CIDR, v)
							}

							k = prefix + "gateway"
							if v := is.Attributes[k]; v != networks.Gateway {
								return fmt.Errorf("want %s=%s, got %s", k, networks.Gateway, v)
							}

							k = prefix + "dns_servers.#"
							n, err = strconv.Atoi(is.Attributes[k])
							if err != nil {
								return err
							}
							if w := len(networks.DNSServers); n != w {
								return fmt.Errorf("want %s=%d, got %d", k, w, n)
							}
							for i, want := range networks.DNSServers {
								k = prefix + fmt.Sprintf("dns_server.%d", i)
								if v := is.Attributes[k]; v != want {
									return fmt.Errorf("want %s=%s, got %s", k, want, v)
								}
							}
						}

						return nil
					}),
					resource.TestCheckResourceAttr("metakube_cluster.acctest_cluster", "spec.0.audit_logging", "false"),
					resource.TestCheckResourceAttr("metakube_cluster.acctest_cluster", "type", "kubernetes"),
					resource.TestCheckResourceAttrSet("metakube_cluster.acctest_cluster", "creation_timestamp"),
					resource.TestCheckResourceAttrSet("metakube_cluster.acctest_cluster", "deletion_timestamp"),
				),
			},
			{
				Config: testAccCheckMetaKubeClusterOpenstackBasic2(testName+"-changed", username, password, tenant, nodeDC, versionK8s17),
				Check: resource.ComposeAggregateTestCheckFunc(
					testResourceInstanceState("metakube_cluster.acctest_cluster", func(is *terraform.InstanceState) error {
						_, _, id, err := metakubeClusterParseID(is.ID)
						if err != nil {
							return err
						}
						if id != cluster.ID {
							return fmt.Errorf("cluster not updated: wrong ID")
						}
						return nil
					}),
					testAccCheckMetaKubeClusterExists(&cluster),
					testAccCheckMetaKubeClusterOpenstackAttributes(&cluster, testName+"-changed", username, password, tenant, nodeDC, versionK8s17, map[string]string{
						"foo":      "bar", // label propagated from project
						"test-key": "test-value",
					}, true),
					resource.TestCheckResourceAttr("metakube_cluster.acctest_cluster", "name", testName+"-changed"),
					resource.TestCheckResourceAttr("metakube_cluster.acctest_cluster", "labels.%", "1"),
					resource.TestCheckResourceAttr("metakube_cluster.acctest_cluster", "labels.test-key", "test-value"),
					resource.TestCheckResourceAttr("metakube_cluster.acctest_cluster", "spec.#", "1"),
					resource.TestCheckResourceAttr("metakube_cluster.acctest_cluster", "spec.0.version", versionK8s17),
					resource.TestCheckResourceAttr("metakube_cluster.acctest_cluster", "spec.0.domain_name", "foodomain.local"),
					resource.TestCheckResourceAttr("metakube_cluster.acctest_cluster", "spec.0.services_cidr", "10.240.16.0/18"),
					resource.TestCheckResourceAttr("metakube_cluster.acctest_cluster", "spec.0.pods_cidr", "172.25.0.0/18"),
					resource.TestCheckResourceAttr("metakube_cluster.acctest_cluster", "spec.0.pod_node_selector", "true"),
					resource.TestCheckResourceAttr("metakube_cluster.acctest_cluster", "spec.0.pod_security_policy", "true"),
					resource.TestCheckResourceAttr("metakube_cluster.acctest_cluster", "spec.0.cloud.#", "1"),
					resource.TestCheckResourceAttr("metakube_cluster.acctest_cluster", "spec.0.cloud.0.bringyourown.#", "0"),
					resource.TestCheckResourceAttr("metakube_cluster.acctest_cluster", "spec.0.cloud.0.aws.#", "0"),
					resource.TestCheckResourceAttr("metakube_cluster.acctest_cluster", "spec.0.cloud.0.openstack.#", "1"),
					resource.TestCheckResourceAttr("metakube_cluster.acctest_cluster", "spec.0.cloud.0.openstack.0.tenant", tenant),
					resource.TestCheckResourceAttr("metakube_cluster.acctest_cluster", "spec.0.cloud.0.openstack.0.username", username),
					resource.TestCheckResourceAttr("metakube_cluster.acctest_cluster", "spec.0.cloud.0.openstack.0.password", password),
					resource.TestCheckResourceAttr("metakube_cluster.acctest_cluster", "spec.0.cloud.0.openstack.0.floating_ip_pool", "ext-net"),
					resource.TestCheckResourceAttrSet("metakube_cluster.acctest_cluster", "spec.0.cloud.0.openstack.0.security_group"),
					resource.TestCheckResourceAttrSet("metakube_cluster.acctest_cluster", "spec.0.cloud.0.openstack.0.network"),
					resource.TestCheckResourceAttrSet("metakube_cluster.acctest_cluster", "spec.0.cloud.0.openstack.0.subnet_id"),
					resource.TestCheckResourceAttr("metakube_cluster.acctest_cluster", "spec.0.cloud.0.openstack.0.subnet_cidr", "192.168.2.0/24"),
					// Test spec.0.machine_networks value
					testResourceInstanceState("metakube_cluster.acctest_cluster", func(is *terraform.InstanceState) error {
						n, err := strconv.Atoi(is.Attributes["spec.0.machine_networks.#"])
						if err != nil {
							return err
						}

						if want := len(cluster.Spec.MachineNetworks); n != want {
							return fmt.Errorf("want len(cluster.Spec.MachineNetworks)=%d, got %d", want, n)
						}

						for i, networks := range cluster.Spec.MachineNetworks {
							prefix := fmt.Sprintf("spec.0.machine_networks.%d.", i)

							var k string

							k = prefix + "cidr"
							if v := is.Attributes[k]; v != networks.CIDR {
								return fmt.Errorf("want %s=%s, got %s", k, networks.CIDR, v)
							}

							k = prefix + "gateway"
							if v := is.Attributes[k]; v != networks.Gateway {
								return fmt.Errorf("want %s=%s, got %s", k, networks.Gateway, v)
							}

							k = prefix + "dns_servers.#"
							n, err = strconv.Atoi(is.Attributes[k])
							if err != nil {
								return err
							}
							if w := len(networks.DNSServers); n != w {
								return fmt.Errorf("want %s=%d, got %d", k, w, n)
							}
							for i, want := range networks.DNSServers {
								k = prefix + fmt.Sprintf("dns_server.%d", i)
								if v := is.Attributes[k]; v != want {
									return fmt.Errorf("want %s=%s, got %s", k, want, v)
								}
							}
						}

						return nil
					}),
					resource.TestCheckResourceAttr("metakube_cluster.acctest_cluster", "spec.0.audit_logging", "true"),
					resource.TestCheckResourceAttr("metakube_cluster.acctest_cluster", "type", "kubernetes"),
					resource.TestCheckResourceAttrSet("metakube_cluster.acctest_cluster", "creation_timestamp"),
					resource.TestCheckResourceAttrSet("metakube_cluster.acctest_cluster", "deletion_timestamp"),
				),
			},
		},
	})
}

func TestAccMetaKubeCluster_Openstack_UpgradeVersion(t *testing.T) {
	var cluster models.Cluster
	testName := randomTestName()
	username := os.Getenv(testEnvOpenstackUsername)
	password := os.Getenv(testEnvOpenstackPassword)
	tenant := os.Getenv(testEnvOpenstackTenant)
	nodeDC := os.Getenv(testEnvOpenstackNodeDC)
	versionedConfig := func(version string) string {
		return testAccCheckMetaKubeClusterOpenstackBasic(testName, username, password, tenant, nodeDC, version)
	}
	versionK8s16 := os.Getenv(testEnvK8sOlderVersion)
	versionK8s17 := os.Getenv(testEnvK8sVersion)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheckForOpenstack(t) },
		Providers: testAccProviders,
		ExternalProviders: map[string]resource.ExternalProvider{
			"openstack": {
				Source: "terraform-provider-openstack/openstack",
			},
		},
		CheckDestroy: testAccCheckMetaKubeClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: versionedConfig(versionK8s16),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckMetaKubeClusterExists(&cluster),
					resource.TestCheckResourceAttr("metakube_cluster.acctest_cluster", "spec.0.version", versionK8s16),
				),
			},
			{
				Config: versionedConfig(versionK8s17),
				Check: resource.ComposeAggregateTestCheckFunc(
					testResourceInstanceState("metakube_cluster.acctest_cluster", func(is *terraform.InstanceState) error {
						_, _, id, err := metakubeClusterParseID(is.ID)
						if err != nil {
							return err
						}
						if id != cluster.ID {
							return fmt.Errorf("cluster not upgraded. Want cluster id=%v, got %v", cluster.ID, id)
						}
						return nil
					}),
					testAccCheckMetaKubeClusterExists(&cluster),
					resource.TestCheckResourceAttr("metakube_cluster.acctest_cluster", "spec.0.version", versionK8s17),
				),
			},
		},
	})
}
func testAccCheckMetaKubeClusterOpenstackBasic(testName, username, password, tenant, nodeDC, version string) string {
	config := `
	terraform {
		required_providers {
			openstack = {
				source = "terraform-provider-openstack/openstack"
			}
		}
	}

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
				openstack {
					tenant = "%s"
					username = "%s"
					password = "%s"
					floating_ip_pool = "ext-net"
					security_group = openstack_networking_secgroup_v2.cluster-net.name
					network = openstack_networking_network_v2.network_tf_test.name
					subnet_id = openstack_networking_subnet_v2.subnet_tf_test.id
					subnet_cidr = "192.168.2.0/24"
				}
			}
			domain_name = "foodomain.local"
			services_cidr = "10.240.16.0/18"
			pods_cidr = "172.25.0.0/18"
		}
	}

	resource "openstack_networking_secgroup_v2" "cluster-net" {
	  name = "tf-test"
	}
	
	resource "openstack_networking_network_v2" "network_tf_test" {
	  name = "network_tf_test"
	}
	
	resource "openstack_networking_subnet_v2" "subnet_tf_test" {
	  name = "subnet_tf_test"
	  network_id = openstack_networking_network_v2.network_tf_test.id
	  cidr = "192.168.0.0/16"
	  ip_version = 4
	}
`

	return fmt.Sprintf(config, testName, testName, nodeDC, version, tenant, username, password)
}

func testAccCheckMetaKubeClusterOpenstackBasic2(testName, username, password, tenant, nodeDC, k8sVersion string) string {
	config := `
	terraform {
		required_providers {
			openstack = {
				source = "terraform-provider-openstack/openstack"
			}
		}
	}
	resource "metakube_project" "acctest_project" {
		name = "%s"
		labels = {
			"foo" = "bar"
		}
	}

	resource "metakube_cluster" "acctest_cluster" {
		name = "%s"
		dc_name = "%s"
		project_id = metakube_project.acctest_project.id

		type = "kubernetes" # should not introduce any change hence type should be computed to this value anyway

		# add labels
		labels = {
			"test-key" = "test-value"
		}

		spec {
			version = "%s"
			cloud {
				openstack {
					tenant = "%s"
					username = "%s"
					password = "%s"
					floating_ip_pool = "ext-net"
					security_group = openstack_networking_secgroup_v2.cluster-net.name
					network = openstack_networking_network_v2.network_tf_test.name
					subnet_id = openstack_networking_subnet_v2.subnet_tf_test.id
					subnet_cidr = "192.168.2.0/24"
				}
			}

			# enable audit logging
			audit_logging = true

			pod_node_selector = true
			pod_security_policy = true
			domain_name = "foodomain.local"
			services_cidr = "10.240.16.0/18"
			pods_cidr = "172.25.0.0/18"
		}
	}

	resource "openstack_networking_secgroup_v2" "cluster-net" {
	  name = "tf-test"
	}
	
	resource "openstack_networking_network_v2" "network_tf_test" {
	  name = "network_tf_test"
	}
	
	resource "openstack_networking_subnet_v2" "subnet_tf_test" {
	  name = "subnet_tf_test"
	  network_id = openstack_networking_network_v2.network_tf_test.id
	  cidr = "192.168.0.0/16"
	  ip_version = 4
	}`
	return fmt.Sprintf(config, testName, testName, nodeDC, k8sVersion, tenant, username, password)
}

func testAccCheckMetaKubeClusterOpenstackAttributes(cluster *models.Cluster, name, username, password, tenant, nodeDC, k8sVersion string, labels map[string]string, auditLogging bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if cluster.Name != name {
			return fmt.Errorf("want .Name=%s, got %s", name, cluster.Name)
		}

		if diff := cmp.Diff(labels, cluster.Labels); diff != "" {
			return fmt.Errorf("wrong labels: %s", diff)
		}

		if cluster.Spec.AuditLogging != nil && cluster.Spec.AuditLogging.Enabled != auditLogging {
			return fmt.Errorf("want .Spec.AuditLogging.Enabled=%v, got %v", auditLogging, cluster.Spec.AuditLogging.Enabled)
		}

		if cluster.Spec.Cloud.DatacenterName != nodeDC {
			return fmt.Errorf("want .Spec.Cloud.DatacenterName=%s, got %s", nodeDC, cluster.Spec.Cloud.DatacenterName)
		}

		if v, ok := cluster.Spec.Version.(string); !ok || v == "" {
			return fmt.Errorf("cluster version is empty")
		} else if v != k8sVersion {
			return fmt.Errorf("want .Spec.Version=%s, got %s", k8sVersion, v)
		}

		openstack := cluster.Spec.Cloud.Openstack

		if openstack == nil {
			return fmt.Errorf("Cluster cloud is not Openstack")
		}

		if openstack.FloatingIPPool != "ext-net" {
			return fmt.Errorf("want .Spec.Cloud.Openstack.FloatingIPPool=%s, got %s", "ext-net", openstack.FloatingIPPool)
		}

		// TODO(furkhat): uncomment if API is fixed.
		// if openstack.Username != username {
		// 	return fmt.Errorf("want .Spec.Cloud.Openstack.Username=%s, got %s", openstack.Username, username)
		// }

		// if openstack.Password != password {
		// 	return fmt.Errorf("want .Spec.Cloud.Openstack.Password=%s, got %s", openstack.Password, password)
		// }

		// if openstack.Tenant != tenant {
		// 	return fmt.Errorf("want .Spec.Cloud.Openstack.Tenant=%s, got %s", openstack.Tenant, tenant)
		// }

		return nil
	}
}

func TestAccMetaKubeCluster_SSHKeys(t *testing.T) {
	var cluster models.Cluster
	var sshkey models.SSHKey
	testName := randomTestName()
	username := os.Getenv(testEnvOpenstackUsername)
	password := os.Getenv(testEnvOpenstackPassword)
	tenant := os.Getenv(testEnvOpenstackTenant)
	nodeDC := os.Getenv(testEnvOpenstackNodeDC)
	k8sVersion17 := os.Getenv(testEnvK8sVersion)

	configClusterWithKey1 := testAccCheckMetaKubeClusterOpenstackBasicWithSSHKey1(testName, username, password, tenant, nodeDC, k8sVersion17)
	configClusterWithKey2 := testAccCheckMetaKubeClusterOpenstackBasicWithSSHKey2(testName, username, password, tenant, nodeDC, k8sVersion17)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheckForOpenstack(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckMetaKubeClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: configClusterWithKey1,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckMetaKubeClusterExists(&cluster),
					testAccCheckMetaKubeSSHKeyExists("metakube_sshkey.acctest_sshkey1", "metakube_project.acctest_project", &sshkey),
					resource.TestCheckResourceAttr("metakube_cluster.acctest_cluster", "sshkeys.#", "1"),
					testAccCheckMetaKubeClusterHasSSHKey(&cluster.ID, &sshkey.ID),
				),
			},
			{
				Config: configClusterWithKey2,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckMetaKubeClusterExists(&cluster),
					testAccCheckMetaKubeSSHKeyExists("metakube_sshkey.acctest_sshkey2", "metakube_project.acctest_project", &sshkey),
					resource.TestCheckResourceAttr("metakube_cluster.acctest_cluster", "sshkeys.#", "1"),
					testAccCheckMetaKubeClusterHasSSHKey(&cluster.ID, &sshkey.ID),
				),
			},
		},
	})
}

func testAccCheckMetaKubeClusterOpenstackBasicWithSSHKey1(testName, username, password, tenant, nodeDC, k8sVersion string) string {
	config := `
	resource "metakube_project" "acctest_project" {
		name = "%s"
	}

	resource "metakube_cluster" "acctest_cluster" {
		name = "%s"
		dc_name = "%s"
		project_id = metakube_project.acctest_project.id

		sshkeys = [
			metakube_sshkey.acctest_sshkey1.id
		]

		spec {
			version = "%s"
			enable_ssh_agent = true
			cloud {
				openstack {
					tenant = "%s"
					username = "%s"
					password = "%s"
					floating_ip_pool = "ext-net"
				}
			}
		}
	}

	resource "metakube_sshkey" "acctest_sshkey1" {
		project_id = metakube_project.acctest_project.id
		name = "acctest-sshkey-1"
		public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCut5oRyqeqYci3E9m6Z6mtxfqkiyb+xNFJM6+/sllhnMDX0vzrNj8PuIFfGkgtowKY//QWLgoB+RpvXqcD4bb4zPkLdXdJPtUf1eAoMh/qgyThUjBs3n7BXvXMDg1Wdj0gq/sTnPLvXsfrSVPjiZvWN4h0JdID2NLnwYuKIiltIn+IbUa6OnyFfOEpqb5XJ7H7LK1mUKTlQ/9CFROxSQf3YQrR9UdtASIeyIZL53WgYgU31Yqy7MQaY1y0fGmHsFwpCK6qFZj1DNruKl/IR1lLx/Bg3z9sDcoBnHKnzSzVels9EVlDOG6bW738ho269QAIrWQYBtznsvWKu5xZPuuj user@machine"
	}`
	return fmt.Sprintf(config, testName, testName, nodeDC, k8sVersion, tenant, username, password)
}

func testAccCheckMetaKubeClusterOpenstackBasicWithSSHKey2(testName, username, password, tenant, nodeDC, k8sVersion string) string {
	config := `
	resource "metakube_project" "acctest_project" {
		name = "%s"
	}

	resource "metakube_cluster" "acctest_cluster" {
		name = "%s"
		dc_name = "%s"
		project_id = metakube_project.acctest_project.id

		sshkeys = [
			metakube_sshkey.acctest_sshkey2.id
		]

		spec {
			version = "%s"
			enable_ssh_agent = true
			cloud {
				openstack {
					tenant = "%s"
					username = "%s"
					password = "%s"
					floating_ip_pool = "ext-net"
				}
			}
		}
	}

	resource "metakube_sshkey" "acctest_sshkey2" {
		project_id = metakube_project.acctest_project.id
		name = "acctest-sshkey-2"
		public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCut5oRyqeqYci3E9m6Z6mtxfqkiyb+xNFJM6+/sllhnMDX0vzrNj8PuIFfGkgtowKY//QWLgoB+RpvXqcD4bb4zPkLdXdJPtUf1eAoMh/qgyThUjBs3n7BXvXMDg1Wdj0gq/sTnPLvXsfrSVPjiZvWN4h0JdID2NLnwYuKIiltIn+IbUa6OnyFfOEpqb5XJ7H7LK1mUKTlQ/9CFROxSQf3YQrR9UdtASIeyIZL53WgYgU31Yqy7MQaY1y0fGmHsFwpCK6qFZj1DNruKl/IR1lLx/Bg3z9sDcoBnHKnzSzVels9EVlDOG6bW738ho269QAIrWQYBtznsvWKu5xZPuuj user@machine"
	}`
	return fmt.Sprintf(config, testName, testName, nodeDC, k8sVersion, tenant, username, password)
}

func testAccCheckMetaKubeClusterHasSSHKey(cluster, sshkey *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources["metakube_cluster.acctest_cluster"]
		if !ok {
			return fmt.Errorf("Not found: %s", "metakube_project.acctest_project")
		}

		projectID, seedDC, _, err := metakubeClusterParseID(rs.Primary.ID)
		if err != nil {
			return err
		}
		k := testAccProvider.Meta().(*metakubeProviderMeta)
		p := project.NewListSSHKeysAssignedToClusterParams()
		p.SetProjectID(projectID)
		p.SetDC(seedDC)
		p.SetClusterID(*cluster)
		ret, err := k.client.Project.ListSSHKeysAssignedToCluster(p, k.auth)
		if err != nil {
			return fmt.Errorf("ListSSHKeysAssignedToCluster %v", err)
		}

		var ids []string
		for _, v := range ret.Payload {
			ids = append(ids, v.ID)
		}

		var sshkeys []string
		if *sshkey != "" {
			sshkeys = []string{*sshkey}
		}
		if diff := cmp.Diff(sshkeys, ids); diff != "" {
			return fmt.Errorf("wrong sshkeys: %s, %s", *sshkey, diff)
		}

		return nil
	}
}

func TestAccMetaKubeCluster_Azure_Basic(t *testing.T) {
	var cluster models.Cluster
	testName := randomTestName()

	clientID := os.Getenv(testEnvAzureClientID)
	clientSecret := os.Getenv(testEnvAzureClientSecret)
	tenantID := os.Getenv(testEnvAzureTenantID)
	subsID := os.Getenv(testEnvAzureSubscriptionID)
	nodeDC := os.Getenv(testEnvAzureNodeDC)
	k8sVersion := os.Getenv(testEnvK8sVersion)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheckForAzure(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckMetaKubeClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckMetaKubeClusterAzureBasic(testName, clientID, clientSecret, tenantID, subsID, nodeDC, k8sVersion),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckMetaKubeClusterExists(&cluster),
					resource.TestCheckResourceAttr("metakube_cluster.acctest_cluster", "spec.0.cloud.0.azure.0.client_id", clientID),
					resource.TestCheckResourceAttr("metakube_cluster.acctest_cluster", "spec.0.cloud.0.azure.0.client_secret", clientSecret),
					resource.TestCheckResourceAttr("metakube_cluster.acctest_cluster", "spec.0.cloud.0.azure.0.tenant_id", tenantID),
					resource.TestCheckResourceAttr("metakube_cluster.acctest_cluster", "spec.0.cloud.0.azure.0.subscription_id", subsID),
				),
			},
		},
	})
}

func testAccCheckMetaKubeClusterAzureBasic(n, clientID, clientSecret, tenantID, subscID, nodeDC, k8sVersion string) string {
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
				azure {
					client_id = "%s"
					client_secret = "%s"
					tenant_id = "%s"
					subscription_id = "%s"
				}
			}
		}
	}`, n, n, nodeDC, k8sVersion, clientID, clientSecret, tenantID, subscID)
}

func TestAccMetaKubeCluster_AWS_Basic(t *testing.T) {
	var cluster models.Cluster
	testName := randomTestName()

	awsAccessKeyID := os.Getenv(testEnvAWSAccessKeyID)
	awsSecretAccessKey := os.Getenv(testAWSSecretAccessKey)
	vpcID := os.Getenv(testEnvAWSVPCID)
	nodeDC := os.Getenv(testEnvAWSNodeDC)
	k8sVersion17 := os.Getenv(testEnvK8sVersion)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheckForAWS(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckMetaKubeClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckMetaKubeClusterAWSBasic(testName, awsAccessKeyID, awsSecretAccessKey, vpcID, nodeDC, k8sVersion17),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckMetaKubeClusterExists(&cluster),
					resource.TestCheckResourceAttr("metakube_cluster.acctest_cluster", "spec.0.cloud.0.aws.#", "1"),
				),
			},
		},
	})
}

func testAccCheckMetaKubeClusterAWSBasic(n, keyID, keySecret, vpcID, nodeDC, k8sVersion string) string {
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
	}`, n, n, nodeDC, k8sVersion, keyID, keySecret, vpcID)
}

func testAccCheckMetaKubeClusterDestroy(s *terraform.State) error {
	k := testAccProvider.Meta().(*metakubeProviderMeta)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "metakube_cluster" {
			continue
		}

		// Try to find the cluster
		p := project.NewGetClusterParams()
		projectID, seedDC, clusterID, err := metakubeClusterParseID(rs.Primary.ID)
		if err != nil {
			return err
		}
		p.SetProjectID(projectID)
		p.SetDC(seedDC)
		p.SetClusterID(clusterID)
		r, err := k.client.Project.GetCluster(p, k.auth)
		if err == nil && r.Payload != nil {
			return fmt.Errorf("Cluster still exists")
		}
	}

	return nil
}

func testAccCheckMetaKubeClusterExists(cluster *models.Cluster) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources["metakube_cluster.acctest_cluster"]
		if !ok {
			return fmt.Errorf("Not found: %s", "metakube_cluster.acctest_cluster")
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		projectID, seedDC, clusterID, err := metakubeClusterParseID(rs.Primary.ID)
		if err != nil {
			return err
		}

		k := testAccProvider.Meta().(*metakubeProviderMeta)
		p := project.NewGetClusterParams()
		p.SetProjectID(projectID)
		p.SetDC(seedDC)
		p.SetClusterID(clusterID)
		ret, err := k.client.Project.GetCluster(p, k.auth)
		if err != nil {
			return fmt.Errorf("GetCluster %v", err)
		}
		if ret.Payload == nil {
			return fmt.Errorf("Record not found")
		}

		*cluster = *ret.Payload

		return nil
	}
}
