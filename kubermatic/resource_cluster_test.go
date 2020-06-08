package kubermatic

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/kubermatic/go-kubermatic/client/project"
	"github.com/kubermatic/go-kubermatic/models"
)

const testClusterVersion16 = "1.16.9"
const testClusterVersion17 = "1.17.5"

func TestAccKubermaticCluster_Openstack_Basic(t *testing.T) {
	var cluster models.Cluster
	testName := randomTestName()

	username := os.Getenv(testEnvOpenstackUsername)
	password := os.Getenv(testEnvOpenstackPassword)
	tenant := os.Getenv(testEnvOpenstackTenant)
	nodeDC := os.Getenv(testEnvOpenstackNodeDC)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheckForOpenstack(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubermaticClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckKubermaticClusterOpenstackBasic(testName, username, password, tenant, nodeDC, testClusterVersion17),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubermaticClusterExists(&cluster),
					testAccCheckKubermaticClusterOpenstackAttributes(&cluster, testName, username, password, tenant, nodeDC, nil, false),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "dc_name", nodeDC),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "name", testName),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "labels.%", "0"),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "spec.#", "1"),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "spec.0.version", "1.17.5"),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "spec.0.cloud.#", "1"),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "spec.0.cloud.0.bringyourown.#", "0"),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "spec.0.cloud.0.aws.#", "0"),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "spec.0.cloud.0.openstack.#", "1"),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "spec.0.cloud.0.openstack.0.tenant", tenant),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "spec.0.cloud.0.openstack.0.username", username),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "spec.0.cloud.0.openstack.0.password", password),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "spec.0.cloud.0.openstack.0.floating_ip_pool", "ext-net"),
					// Test spec.0.machine_networks value
					testResourceInstanceState("kubermatic_cluster.acctest_cluster", func(is *terraform.InstanceState) error {
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
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "spec.0.audit_logging", "false"),
					// Test credential
					testResourceInstanceState("kubermatic_cluster.acctest_cluster", func(is *terraform.InstanceState) error {
						v, ok := is.Attributes["credential"]
						if !ok && cluster.Credential != "" {
							return fmt.Errorf("cluster credential not set")
						}
						if want := cluster.Credential; want != v {
							return fmt.Errorf("want .Credential=%s, got %s", want, v)
						}

						return nil
					}),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "type", "kubernetes"),
					resource.TestCheckResourceAttrSet("kubermatic_cluster.acctest_cluster", "creation_timestamp"),
					resource.TestCheckResourceAttrSet("kubermatic_cluster.acctest_cluster", "deletion_timestamp"),
				),
			},
			{
				Config: testAccCheckKubermaticClusterOpenstackBasic2(testName+"-changed", username, password, tenant, nodeDC),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubermaticClusterExists(&cluster),
					testAccCheckKubermaticClusterOpenstackAttributes(&cluster, testName+"-changed", username, password, tenant, nodeDC, map[string]string{
						"foo":      "bar", // label propogated from project
						"test-key": "test-value",
					}, true),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "name", testName+"-changed"),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "labels.%", "1"),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "labels.test-key", "test-value"),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "spec.#", "1"),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "spec.0.version", "1.17.5"),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "spec.0.cloud.#", "1"),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "spec.0.cloud.0.bringyourown.#", "0"),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "spec.0.cloud.0.aws.#", "0"),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "spec.0.cloud.0.openstack.#", "1"),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "spec.0.cloud.0.openstack.0.tenant", tenant),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "spec.0.cloud.0.openstack.0.username", username),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "spec.0.cloud.0.openstack.0.password", password),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "spec.0.cloud.0.openstack.0.floating_ip_pool", "ext-net"),
					// Test spec.0.machine_networks value
					testResourceInstanceState("kubermatic_cluster.acctest_cluster", func(is *terraform.InstanceState) error {
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
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "spec.0.audit_logging", "true"),
					// Test credential
					testResourceInstanceState("kubermatic_cluster.acctest_cluster", func(is *terraform.InstanceState) error {
						v, ok := is.Attributes["credential"]
						if !ok && cluster.Credential != "" {
							return fmt.Errorf("cluster credential not set")
						}
						if want := cluster.Credential; want != v {
							return fmt.Errorf("want .Credential=%s, got %s", want, v)
						}

						return nil
					}),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "type", "kubernetes"),
					resource.TestCheckResourceAttrSet("kubermatic_cluster.acctest_cluster", "creation_timestamp"),
					resource.TestCheckResourceAttrSet("kubermatic_cluster.acctest_cluster", "deletion_timestamp"),
				),
			},
		},
	})
}

func TestAccKubermaticCluster_Openstack_UpgradeVersion(t *testing.T) {
	var cluster models.Cluster
	testName := randomTestName()
	username := os.Getenv(testEnvOpenstackUsername)
	password := os.Getenv(testEnvOpenstackPassword)
	tenant := os.Getenv(testEnvOpenstackTenant)
	nodeDC := os.Getenv(testEnvOpenstackNodeDC)
	versionedConfig := func(version string) string {
		return testAccCheckKubermaticClusterOpenstackBasic(testName, username, password, tenant, nodeDC, version)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheckForOpenstack(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubermaticClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: versionedConfig(testClusterVersion16),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubermaticClusterExists(&cluster),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "spec.0.version", testClusterVersion16),
				),
			},
			{
				Config: versionedConfig(testClusterVersion17),
				Check: resource.ComposeAggregateTestCheckFunc(
					testResourceInstanceState("kubermatic_cluster.acctest_cluster", func(is *terraform.InstanceState) error {
						_, _, id, err := kubermaticClusterParseID(is.ID)
						if err != nil {
							return err
						}
						if id != cluster.ID {
							return fmt.Errorf("cluster not upgraded. Want cluster id=%v, got %v", cluster.ID, id)
						}
						return nil
					}),
					testAccCheckKubermaticClusterExists(&cluster),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "spec.0.version", testClusterVersion17),
				),
			},
		},
	})
}
func testAccCheckKubermaticClusterOpenstackBasic(testName, username, password, tenant, nodeDC, version string) string {
	config := `
	provider "kubermatic" {}

	resource "kubermatic_project" "acctest_project" {
		name = "%s"
	}

	resource "kubermatic_cluster" "acctest_cluster" {
		name = "%s"
		dc_name = "%s"
		project_id = kubermatic_project.acctest_project.id

		spec {
			version = "%s"
			cloud {
				openstack {
					tenant = "%s"
					username = "%s"
					password = "%s"
					floating_ip_pool = "ext-net"
				}
			}
		}
	}`

	return fmt.Sprintf(config, testName, testName, nodeDC, version, tenant, username, password)
}

func testAccCheckKubermaticClusterOpenstackBasic2(testName, username, password, tenant, nodeDC string) string {
	config := `
	provider "kubermatic" {}

	resource "kubermatic_project" "acctest_project" {
		name = "%s"
		labels = {
			"foo" = "bar"
		}
	}

	resource "kubermatic_cluster" "acctest_cluster" {
		name = "%s"
		dc_name = "%s"
		project_id = kubermatic_project.acctest_project.id

		type = "kubernetes" # should not introduce any change hence type should be computed to this value anyway

		# add labels
		labels = {
			"test-key" = "test-value"
		}

		spec {
			version = "1.17.5"
			cloud {
				openstack {
					tenant = "%s"
					username = "%s"
					password = "%s"
					floating_ip_pool = "ext-net"
				}
			}

			# enable audit logging
			audit_logging = true
		}
	}`
	return fmt.Sprintf(config, testName, testName, nodeDC, tenant, username, password)
}

func testAccCheckKubermaticClusterOpenstackAttributes(cluster *models.Cluster, name, username, password, tenant, nodeDC string, labels map[string]string, auditLogging bool) resource.TestCheckFunc {
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
		} else if v != "1.17.5" {
			return fmt.Errorf("want .Spec.Version=1.7.4, got %s", v)
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

func TestAccKubermaticCluster_SSHKeys(t *testing.T) {
	var cluster models.Cluster
	var sshkey models.SSHKey
	testName := randomTestName()
	username := os.Getenv(testEnvOpenstackUsername)
	password := os.Getenv(testEnvOpenstackPassword)
	tenant := os.Getenv(testEnvOpenstackTenant)
	nodeDC := os.Getenv(testEnvOpenstackNodeDC)

	configClusterWithKey1 := testAccCheckKubermaticClusterOpenstackBasicWithSSHKey1(testName, username, password, tenant, nodeDC)
	configClusterWithKey2 := testAccCheckKubermaticClusterOpenstackBasicWithSSHKey2(testName, username, password, tenant, nodeDC)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheckForOpenstack(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubermaticClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: configClusterWithKey1,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubermaticClusterExists(&cluster),
					testAccCheckKubermaticSSHKeyExists("kubermatic_sshkey.acctest_sshkey1", "kubermatic_project.acctest_project", &sshkey),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "sshkeys.#", "1"),
					testAccCheckKubermaticClusterHasSSHKey(&cluster.ID, &sshkey.ID),
				),
			},
			{
				Config: configClusterWithKey2,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubermaticClusterExists(&cluster),
					testAccCheckKubermaticSSHKeyExists("kubermatic_sshkey.acctest_sshkey2", "kubermatic_project.acctest_project", &sshkey),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "sshkeys.#", "1"),
					testAccCheckKubermaticClusterHasSSHKey(&cluster.ID, &sshkey.ID),
				),
			},
		},
	})
}

func testAccCheckKubermaticClusterOpenstackBasicWithSSHKey1(testName, username, password, tenant, nodeDC string) string {
	config := `
	provider "kubermatic" {}

	resource "kubermatic_project" "acctest_project" {
		name = "%s"
	}

	resource "kubermatic_cluster" "acctest_cluster" {
		name = "%s"
		dc_name = "%s"
		project_id = kubermatic_project.acctest_project.id

		sshkeys = [
			kubermatic_sshkey.acctest_sshkey1.id
		]

		spec {
			version = "1.17.5"
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

	resource "kubermatic_sshkey" "acctest_sshkey1" {
		project_id = kubermatic_project.acctest_project.id
		name = "acctest-sshkey-1"
		public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCut5oRyqeqYci3E9m6Z6mtxfqkiyb+xNFJM6+/sllhnMDX0vzrNj8PuIFfGkgtowKY//QWLgoB+RpvXqcD4bb4zPkLdXdJPtUf1eAoMh/qgyThUjBs3n7BXvXMDg1Wdj0gq/sTnPLvXsfrSVPjiZvWN4h0JdID2NLnwYuKIiltIn+IbUa6OnyFfOEpqb5XJ7H7LK1mUKTlQ/9CFROxSQf3YQrR9UdtASIeyIZL53WgYgU31Yqy7MQaY1y0fGmHsFwpCK6qFZj1DNruKl/IR1lLx/Bg3z9sDcoBnHKnzSzVels9EVlDOG6bW738ho269QAIrWQYBtznsvWKu5xZPuuj user@machine"
	}`
	return fmt.Sprintf(config, testName, testName, nodeDC, tenant, username, password)
}

func testAccCheckKubermaticClusterOpenstackBasicWithSSHKey2(testName, username, password, tenant, nodeDC string) string {
	config := `
	provider "kubermatic" {}

	resource "kubermatic_project" "acctest_project" {
		name = "%s"
	}

	resource "kubermatic_cluster" "acctest_cluster" {
		name = "%s"
		dc_name = "%s"
		project_id = kubermatic_project.acctest_project.id

		sshkeys = [
			kubermatic_sshkey.acctest_sshkey2.id
		]

		spec {
			version = "1.17.5"
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

	resource "kubermatic_sshkey" "acctest_sshkey2" {
		project_id = kubermatic_project.acctest_project.id
		name = "acctest-sshkey-2"
		public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCut5oRyqeqYci3E9m6Z6mtxfqkiyb+xNFJM6+/sllhnMDX0vzrNj8PuIFfGkgtowKY//QWLgoB+RpvXqcD4bb4zPkLdXdJPtUf1eAoMh/qgyThUjBs3n7BXvXMDg1Wdj0gq/sTnPLvXsfrSVPjiZvWN4h0JdID2NLnwYuKIiltIn+IbUa6OnyFfOEpqb5XJ7H7LK1mUKTlQ/9CFROxSQf3YQrR9UdtASIeyIZL53WgYgU31Yqy7MQaY1y0fGmHsFwpCK6qFZj1DNruKl/IR1lLx/Bg3z9sDcoBnHKnzSzVels9EVlDOG6bW738ho269QAIrWQYBtznsvWKu5xZPuuj user@machine"
	}`
	return fmt.Sprintf(config, testName, testName, nodeDC, tenant, username, password)
}

func testAccCheckKubermaticClusterHasSSHKey(cluster, sshkey *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources["kubermatic_cluster.acctest_cluster"]
		if !ok {
			return fmt.Errorf("Not found: %s", "kubermatic_project.acctest_project")
		}

		projectID, seedDC, _, err := kubermaticClusterParseID(rs.Primary.ID)
		if err != nil {
			return err
		}
		k := testAccProvider.Meta().(*kubermaticProviderMeta)
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

func TestAccKubermaticCluster_Azure_Basic(t *testing.T) {
	t.Skip()

	var cluster models.Cluster
	testName := randomTestName()

	clientID := os.Getenv(testEnvAzureClientID)
	clientSecret := os.Getenv(testEnvAzureClientSecret)
	tenantID := os.Getenv(testEnvAzureTenantID)
	subsID := os.Getenv(testEnvAzureSubscriptionID)
	nodeDC := os.Getenv(testEnvAzureNodeDC)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheckForAzure(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubermaticClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckKubermaticClusterAzureBasic(testName, clientID, clientSecret, tenantID, subsID, nodeDC),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubermaticClusterExists(&cluster),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "spec.0.cloud.0.azure.0.client_id", clientID),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "spec.0.cloud.0.azure.0.client_secret", clientSecret),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "spec.0.cloud.0.azure.0.tenant_id", tenantID),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "spec.0.cloud.0.azure.0.subscription_id", subsID),
				),
			},
		},
	})
}

func testAccCheckKubermaticClusterAzureBasic(n, clientID, clientSecret, tenantID, subscID, nodeDC string) string {
	return fmt.Sprintf(`
	provider "kubermatic" {}

	resource "kubermatic_project" "acctest_project" {
		name = "%s"
	}

	resource "kubermatic_cluster" "acctest_cluster" {
		name = "%s"
		dc_name = "%s"
		project_id = kubermatic_project.acctest_project.id

		spec {
			version = "1.17.5"
			cloud {
				azure {
					client_id = "%s"
					client_secret = "%s"
					tenant_id = "%s"
					subscription_id = "%s"
				}
			}
		}
	}`, n, n, nodeDC, clientID, clientSecret, tenantID, subscID)
}

func testAccCheckKubermaticClusterDestroy(s *terraform.State) error {
	k := testAccProvider.Meta().(*kubermaticProviderMeta)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubermatic_cluster" {
			continue
		}

		// Try to find the cluster
		p := project.NewGetClusterParams()
		projectID, seedDC, clusterID, err := kubermaticClusterParseID(rs.Primary.ID)
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

func testAccCheckKubermaticClusterExists(cluster *models.Cluster) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources["kubermatic_cluster.acctest_cluster"]
		if !ok {
			return fmt.Errorf("Not found: %s", "kubermatic_cluster.acctest_cluster")
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		projectID, seedDC, clusterID, err := kubermaticClusterParseID(rs.Primary.ID)
		if err != nil {
			return err
		}

		k := testAccProvider.Meta().(*kubermaticProviderMeta)
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
