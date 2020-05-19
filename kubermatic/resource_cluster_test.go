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

const testClusterVersion17 = "1.17.4"

func TestAccKubermaticCluster_Openstack_Basic(t *testing.T) {
	var cluster models.Cluster
	testName := randomTestName()

	username := os.Getenv(testEnvOpenstackUsername)
	password := os.Getenv(testEnvOpenstackPassword)
	tenant := os.Getenv(testEnvOpenstackTenant)
	seedDC := os.Getenv(testEnvOpenstackSeedDC)
	nodeDC := os.Getenv(testEnvOpenstackNodeDC)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheckForOpenstack(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubermaticClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckKubermaticClusterOpenstackBasic(testName, username, password, tenant, seedDC, nodeDC, testClusterVersion17),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubermaticClusterExists(seedDC, &cluster),
					testAccCheckKubermaticClusterOpenstackAttributes(&cluster, testName, username, password, tenant, nodeDC, nil, false),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "dc", seedDC),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "name", testName),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "labels.%", "0"),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "spec.#", "1"),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "spec.0.version", "1.17.4"),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "spec.0.cloud.#", "1"),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "spec.0.cloud.0.bringyourown.#", "0"),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "spec.0.cloud.0.aws.#", "0"),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "spec.0.cloud.0.openstack.#", "1"),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "spec.0.cloud.0.openstack.0.tenant", tenant),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "spec.0.cloud.0.openstack.0.username", username),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "spec.0.cloud.0.openstack.0.password", password),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "spec.0.cloud.0.openstack.0.floating_ip_pool", "ext-net"),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "spec.0.cloud.0.dc", nodeDC),
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
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "spec.0.audit_logging.#", "0"),
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
				Config: testAccCheckKubermaticClusterOpenstackBasic2(testName+"-changed", username, password, tenant, seedDC, nodeDC),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubermaticClusterExists(seedDC, &cluster),
					testAccCheckKubermaticClusterOpenstackAttributes(&cluster, testName+"-changed", username, password, tenant, nodeDC, map[string]string{
						"foo":      "bar", // label propogated from project
						"test-key": "test-value",
					}, true),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "dc", seedDC),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "name", testName+"-changed"),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "labels.%", "1"),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "labels.test-key", "test-value"),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "spec.#", "1"),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "spec.0.version", "1.17.4"),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "spec.0.cloud.#", "1"),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "spec.0.cloud.0.bringyourown.#", "0"),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "spec.0.cloud.0.aws.#", "0"),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "spec.0.cloud.0.openstack.#", "1"),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "spec.0.cloud.0.openstack.0.tenant", tenant),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "spec.0.cloud.0.openstack.0.username", username),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "spec.0.cloud.0.openstack.0.password", password),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "spec.0.cloud.0.openstack.0.floating_ip_pool", "ext-net"),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "spec.0.cloud.0.dc", nodeDC),
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
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "spec.0.audit_logging.#", "1"),
					resource.TestCheckResourceAttr("kubermatic_cluster.acctest_cluster", "spec.0.audit_logging.0.enabled", "true"),
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

func testAccCheckKubermaticClusterDestroy(s *terraform.State) error {
	k := testAccProvider.Meta().(*kubermaticProviderMeta)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubermatic_cluster" {
			continue
		}

		// Try to find the cluster
		p := project.NewGetClusterParams()
		p.SetClusterID(rs.Primary.ID)
		p.SetDC(rs.Primary.Attributes["dc"])
		p.SetProjectID(rs.Primary.Attributes["project_id"])
		r, err := k.client.Project.GetCluster(p, k.auth)
		if err == nil && r.Payload != nil {
			return fmt.Errorf("Cluster still exists")
		}
	}

	return nil
}

func testAccCheckKubermaticClusterOpenstackBasic(testName, username, password, tenant, seedDC, nodeDC, version string) string {
	config := `
	provider "kubermatic" {}

	resource "kubermatic_project" "acctest_project" {
		name = "%s"
	}

	resource "kubermatic_cluster" "acctest_cluster" {
		name = "%s"
		dc = "%s"
		project_id = kubermatic_project.acctest_project.id

		spec {
			version = "%s"
			cloud {
				dc = "%s"
				openstack {
					tenant = "%s"
					username = "%s"
					password = "%s"
					floating_ip_pool = "ext-net"
				}
			}
		}
	}`

	return fmt.Sprintf(config, testName, testName, seedDC, version, nodeDC, tenant, username, password)
}

func testAccCheckKubermaticClusterOpenstackBasic2(testName, username, password, tenant, seedDC, nodeDC string) string {
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
		dc = "%s"
		project_id = kubermatic_project.acctest_project.id

		type = "kubernetes" # should not introduce any change hence type should be computed to this value anyway

		# add labels
		labels = {
			"test-key" = "test-value"
		}

		spec {
			version = "1.17.4"
			cloud {
				dc = "%s"
				openstack {
					tenant = "%s"
					username = "%s"
					password = "%s"
					floating_ip_pool = "ext-net"
				}
			}

			# enable audit logging
			audit_logging {
				enabled = true
			}
		}
	}`
	return fmt.Sprintf(config, testName, testName, seedDC, nodeDC, tenant, username, password)
}

func testAccCheckKubermaticClusterExists(seedDC string, cluster *models.Cluster) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		var projectID, clusterID string

		rs, ok := s.RootModule().Resources["kubermatic_project.acctest_project"]
		if !ok {
			return fmt.Errorf("Not found: %s", "kubermatic_project.acctest_project")
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}
		projectID = rs.Primary.ID

		rs, ok = s.RootModule().Resources["kubermatic_cluster.acctest_cluster"]
		if !ok {
			return fmt.Errorf("Not found: %s", "kubermatic_cluster.acctest_cluster")
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}
		clusterID = rs.Primary.ID

		k := testAccProvider.Meta().(*kubermaticProviderMeta)
		p := project.NewGetClusterParams()
		p.SetProjectID(projectID)
		p.SetClusterID(clusterID)
		p.SetDC(seedDC)
		ret, err := k.client.Project.GetCluster(p, k.auth)
		if err != nil {
			return fmt.Errorf("GetCluster %w", err)
		}
		if ret.Payload == nil {
			return fmt.Errorf("Record not found")
		}

		*cluster = *ret.Payload

		return nil
	}
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
		} else if v != "1.17.4" {
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
