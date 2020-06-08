package kubermatic

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/kubermatic/go-kubermatic/client/project"
	"github.com/kubermatic/go-kubermatic/client/users"
	"github.com/kubermatic/go-kubermatic/models"
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

func TestAccKubermaticProject_Basic(t *testing.T) {
	var project models.Project
	var projectUsers []*models.User
	projectName := randomTestName()
	otherUsersEmail := os.Getenv(testEnvOtherUserEmail)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			checkEnv(t, testEnvOtherUserEmail)
		},
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubermaticProjectDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccCheckKubermaticProjectConfigBasic, projectName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubermaticProjectExists("kubermatic_project.foobar", &project, &projectUsers),
					testAccCheckKubermaticProjectAttributes(&project, &projectUsers, 1, projectName, map[string]string{
						"foo": "bar",
					}),
					resource.TestCheckResourceAttr(
						"kubermatic_project.foobar", "user.#", "0"),
					resource.TestCheckResourceAttr(
						"kubermatic_project.foobar", "name", projectName),
					resource.TestCheckResourceAttr(
						"kubermatic_project.foobar", "labels.foo", "bar"),
				),
			},
			{
				Config: fmt.Sprintf(testAccCheckKubermaticProjectConfigBasic2, projectName, otherUsersEmail),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubermaticProjectExists("kubermatic_project.foobar", &project, &projectUsers),
					testAccCheckKubermaticProjectAttributes(&project, &projectUsers, 2, projectName+"-changed", map[string]string{
						"foo":     "bar-changed",
						"new-key": "new-value",
					}),
					resource.TestCheckResourceAttr(
						"kubermatic_project.foobar", "user.#", "1"),
					resource.TestCheckResourceAttr(
						"kubermatic_project.foobar", "name", projectName+"-changed"),
					resource.TestCheckResourceAttr(
						"kubermatic_project.foobar", "labels.foo", "bar-changed"),
					resource.TestCheckResourceAttr(
						"kubermatic_project.foobar", "labels.new-key", "new-value"),
				),
			},
		},
	})
}

func testAccCheckKubermaticProjectDestroy(s *terraform.State) error {
	k := testAccProvider.Meta().(*kubermaticProviderMeta)
	p := project.NewGetProjectParams()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubermatic_project" {
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

const testAccCheckKubermaticProjectConfigBasic = `
resource "kubermatic_project" "foobar" {
	name = "%s"
	labels = {
		"foo" = "bar"
	}
}
`

const testAccCheckKubermaticProjectConfigBasic2 = `
resource "kubermatic_project" "foobar" {
	name = "%s-changed"
	labels = {
		"foo" = "bar-changed"
		"new-key" = "new-value"
	}

	user {
		email = "%s"
		group = "viewers"
	}
}
`

func testAccCheckKubermaticProjectExists(n string, rec *models.Project, u *[]*models.User) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Record ID is set")
		}

		k := testAccProvider.Meta().(*kubermaticProviderMeta)
		p := project.NewGetProjectParams()

		r, err := k.client.Project.GetProject(p.WithProjectID(rs.Primary.ID), k.auth)
		if err != nil {
			return fmt.Errorf("GetProject %v", err)
		}
		if r.Payload == nil {
			return fmt.Errorf("Record not found")
		}

		*rec = *r.Payload

		r2, err := k.client.Users.GetUsersForProject(users.NewGetUsersForProjectParams().WithProjectID(rec.ID), k.auth)
		if err != nil {
			return fmt.Errorf("GetUsersForProject: %v", err)
		}

		*u = r2.Payload

		return nil
	}
}

func testAccCheckKubermaticProjectAttributes(rec *models.Project, users *[]*models.User, wantUsers int, name string, labels map[string]string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if rec.Name != name {
			return fmt.Errorf("want project.Name=%s, got %s", name, rec.Name)
		}
		if !reflect.DeepEqual(rec.Labels, labels) {
			return fmt.Errorf("want project.Labels=%+v, got %+v", labels, rec.Labels)
		}

		if len(*users) != wantUsers {
			return fmt.Errorf("want %d project user, got %d", wantUsers, len(*users))
		}

		return nil
	}
}
