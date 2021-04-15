package metakube

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/syseleven/terraform-provider-metakube/go-metakube/client/project"
	"github.com/syseleven/terraform-provider-metakube/go-metakube/models"
)

func init() {
	resource.AddTestSweepers("kubermetic_project", &resource.Sweeper{
		Name: "kubermetic_project",
		F:    testSweepProject,
	})
}

const projectTerminating = "Terminating"

func testSweepProject(region string) error {
	meta, err := sharedConfigForRegion(region)
	if err != nil {
		return err
	}

	records, err := meta.client.Project.ListProjects(project.NewListProjectsParams(), meta.auth)
	if err != nil {
		return fmt.Errorf("list projects: %v", err)
	}

	for _, rec := range records.Payload {
		if !strings.HasPrefix(rec.Name, testNamePrefix) || rec.Status == projectTerminating {
			continue
		}

		p := project.NewDeleteProjectParams()
		p.ProjectID = rec.ID
		if _, err := meta.client.Project.DeleteProject(p, meta.auth); err != nil {
			return fmt.Errorf("delete project: %v", err)
		}
	}

	return nil
}

func TestAccMetakubeProject_BasicAndImport(t *testing.T) {
	var project models.Project
	projectName := makeRandomString()
	resourceName := "metakube_project.foobar"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			checkEnv(t, testEnvOtherUserEmail)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckMetaKubeProjectDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccCheckMetaKubeProjectConfigBasic, projectName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckMetaKubeProjectExists("metakube_project.foobar", &project),
					testAccCheckMetaKubeProjectAttributes(&project, projectName, map[string]string{
						"foo": "bar",
					}),
					resource.TestCheckResourceAttr(
						"metakube_project.foobar", "name", projectName),
					resource.TestCheckResourceAttr(
						"metakube_project.foobar", "labels.foo", "bar"),
				),
			},
			{
				Config: fmt.Sprintf(testAccCheckMetaKubeProjectConfigBasic2, projectName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckMetaKubeProjectExists("metakube_project.foobar", &project),
					testAccCheckMetaKubeProjectAttributes(&project, projectName+"-changed", map[string]string{
						"foo":     "bar-changed",
						"new-key": "new-value",
					}),
					resource.TestCheckResourceAttr(
						"metakube_project.foobar", "name", projectName+"-changed"),
					resource.TestCheckResourceAttr(
						"metakube_project.foobar", "labels.foo", "bar-changed"),
					resource.TestCheckResourceAttr(
						"metakube_project.foobar", "labels.new-key", "new-value"),
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

func testAccCheckMetaKubeProjectDestroy(s *terraform.State) error {
	k := testAccProvider.Meta().(*metakubeProviderMeta)
	p := project.NewGetProjectParams()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "metakube_project" {
			continue
		}

		// Try to find the project
		r, err := k.client.Project.GetProject(p.WithProjectID(rs.Primary.ID), k.auth)
		if err == nil && r.Payload != nil {
			return fmt.Errorf("Project still exists")
		}
	}

	return nil
}

const testAccCheckMetaKubeProjectConfigBasic = `
resource "metakube_project" "foobar" {
	name = "%s"
	labels = {
		"foo" = "bar"
	}
}
`

const testAccCheckMetaKubeProjectConfigBasic2 = `
resource "metakube_project" "foobar" {
	name = "%s-changed"
	labels = {
		"foo" = "bar-changed"
		"new-key" = "new-value"
	}
}
`

func testAccCheckMetaKubeProjectExists(n string, rec *models.Project) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		k := testAccProvider.Meta().(*metakubeProviderMeta)
		p := project.NewGetProjectParams()

		r, err := k.client.Project.GetProject(p.WithProjectID(rs.Primary.ID), k.auth)
		if err != nil {
			return fmt.Errorf("GetProject %v", err)
		}
		if r.Payload == nil {
			return fmt.Errorf("Record not found")
		}

		*rec = *r.Payload

		return nil
	}
}

func testAccCheckMetaKubeProjectAttributes(rec *models.Project, name string, labels map[string]string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if rec.Name != name {
			return fmt.Errorf("want project.Name=%s, got %s", name, rec.Name)
		}
		if !reflect.DeepEqual(rec.Labels, labels) {
			return fmt.Errorf("want project.Labels=%+v, got %+v", labels, rec.Labels)
		}
		return nil
	}
}
