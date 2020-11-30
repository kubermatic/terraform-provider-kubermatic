package metakube

import (
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/syseleven/terraform-provider-metakube/go-metakube/client/project"
	"github.com/syseleven/terraform-provider-metakube/go-metakube/client/users"
	"github.com/syseleven/terraform-provider-metakube/go-metakube/models"
)

const (
	projectActive    = "Active"
	projectInactive  = "Inactive"
	usersReady       = "Ready"
	usersUnavailable = "Unavailable"
)

func resourceProject() *schema.Resource {
	return &schema.Resource{
		Create: resourceProjectCreate,
		Read:   resourceProjectRead,
		Update: resourceProjectUpdate,
		Delete: resourceProjectDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Project name",
			},
			"labels": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "Project labels",
				Elem:        schema.TypeString,
			},
			"user": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "Project user",
				Elem:        projectUsersSchema(),
			},
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Status represents the current state of the project",
			},
			"creation_timestamp": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Creation timestamp",
			},
			"deletion_timestamp": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Deletion timestamp",
			},
		},
	}
}

func projectUsersSchema() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"email": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.NoZeroValues,
				Description:  "User's email address",
			},
			"group": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringInSlice([]string{"owners", "editors", "viewers"}, false),
				Description:  "User's role in the project",
			},
		},
	}
}

func resourceProjectCreate(d *schema.ResourceData, m interface{}) error {
	k := m.(*metakubeProviderMeta)
	p := project.NewCreateProjectParams()

	p.Body.Name = d.Get("name").(string)
	if l, ok := d.GetOk("labels"); ok {
		p.Body.Labels = make(map[string]string)
		att := l.(map[string]interface{})
		for key, val := range att {
			p.Body.Labels[key] = val.(string)
		}
	}

	r, err := k.client.Project.CreateProject(p, k.auth)
	if err != nil {
		return fmt.Errorf("error when creating a project: %s", getErrorResponse(err))
	}
	d.SetId(r.Payload.ID)

	id := r.Payload.ID
	createStateConf := &resource.StateChangeConf{
		Pending: []string{
			projectInactive,
		},
		Target: []string{
			projectActive,
		},
		Refresh: func() (interface{}, string, error) {
			p := project.NewGetProjectParams()
			r, err := k.client.Project.GetProject(p.WithProjectID(id), k.auth)
			if err != nil {
				if e, ok := err.(*project.GetProjectDefault); ok && (e.Code() == http.StatusForbidden || e.Code() == http.StatusNotFound) {
					return r, projectInactive, fmt.Errorf("project not ready: %v", err)
				}
				return nil, "", err
			}
			k.log.Debugf("creating project '%s', currently in '%s' state", id, r.Payload.Status)
			return r, projectActive, nil
		},
		Timeout:    d.Timeout(schema.TimeoutCreate),
		MinTimeout: 5 * retryTimeout,
		Delay:      5 * requestDelay,
	}

	if _, err := createStateConf.WaitForState(); err != nil {
		k.log.Debugf("error while waiting for project '%s' to be created: %s", id, err)
		return fmt.Errorf("error while waiting for project '%s' to be created: %s", id, err)
	}

	if err := metakubeProjectUpdateUsers(k, d); err != nil {
		return fmt.Errorf("error updating project's users: %v", err)
	}

	return resourceProjectRead(d, m)
}

func resourceProjectRead(d *schema.ResourceData, m interface{}) error {
	k := m.(*metakubeProviderMeta)
	p := project.NewGetProjectParams()

	r, err := k.client.Project.GetProject(p.WithProjectID(d.Id()), k.auth)
	if err != nil {
		if e, ok := err.(*project.GetProjectDefault); ok && (e.Code() == http.StatusForbidden || e.Code() == http.StatusNotFound) {
			// remove a project from terraform state file that a user does not have access or does not exist
			k.log.Infof("removing project '%s' from terraform state file, code '%d' has been returned", d.Id(), e.Code())
			d.SetId("")
			return nil

		}

		return fmt.Errorf("unable to get project '%s': %s", d.Id(), getErrorResponse(err))
	}

	if err := d.Set("labels", r.Payload.Labels); err != nil {
		return err
	}
	d.Set("name", r.Payload.Name)
	d.Set("status", r.Payload.Status)
	d.Set("creation_timestamp", r.Payload.CreationTimestamp.String())
	d.Set("deletion_timestamp", r.Payload.DeletionTimestamp.String())

	users, err := metakubeProjectPersistedUsers(k, d.Id())
	if err != nil {
		return err
	}

	curUser, err := metakubeProjectCurrentUser(k)
	if err != nil {
		return err
	}

	return d.Set("user", flattendProjectUsers(curUser, users))
}

func flattendProjectUsers(cur *models.User, u map[string]models.User) *schema.Set {
	var items []interface{}
	for _, v := range u {
		if v.Email == cur.Email {
			continue
		}
		items = append(items, map[string]interface{}{
			"email": v.Email,
			"group": v.Projects[0].GroupPrefix,
		})
	}
	return schema.NewSet(schema.HashResource(projectUsersSchema()), items)
}

func resourceProjectUpdate(d *schema.ResourceData, m interface{}) error {
	k := m.(*metakubeProviderMeta)
	p := project.NewUpdateProjectParams()
	p.Body = &models.Project{
		// name is always required for update requests, otherwise bad request returns
		Name: d.Get("name").(string),
	}

	if d.HasChange("name") {
		p.Body.Name = d.Get("name").(string)
	}

	if d.HasChange("labels") {
		p.Body.Labels = make(map[string]string)
		for key, val := range d.Get("labels").(map[string]interface{}) {
			p.Body.Labels[key] = val.(string)
		}
	}

	_, err := k.client.Project.UpdateProject(p.WithProjectID(d.Id()), k.auth)
	if err != nil {
		return fmt.Errorf("unable to update project '%s': %s", d.Id(), getErrorResponse(err))
	}

	if d.HasChange("user") {
		if err := metakubeProjectUpdateUsers(k, d); err != nil {
			return fmt.Errorf("error updating project's users: %v", err)
		}
	}

	return resourceProjectRead(d, m)
}

func metakubeProjectUpdateUsers(k *metakubeProviderMeta, d *schema.ResourceData) error {
	curUser, err := metakubeProjectCurrentUser(k)
	if err != nil {
		return err
	}

	persistedUsers, err := metakubeProjectPersistedUsers(k, d.Id())
	if err != nil {
		return err
	}

	configuredUsers := metakubeProjectConfiguredUsers(d)

	for email, pu := range persistedUsers {
		if pu.Email == curUser.Email {
			continue
		}

		if cu, ok := configuredUsers[email]; ok {
			if pu.Projects[0].GroupPrefix != cu.Projects[0].GroupPrefix {
				pu.Projects[0].GroupPrefix = cu.Projects[0].GroupPrefix
				if err := metakubeProjectEditUser(k, d.Id(), &pu); err != nil {
					return err
				}
			}
			continue
		}
		if err := metakubeProjectDeleteUser(k, d.Id(), pu.ID); err != nil {
			return err
		}
	}

	for email, cu := range configuredUsers {
		if _, ok := persistedUsers[email]; !ok {
			if err := metakubeProjectAddUser(k, d.Id(), &cu); err != nil {
				return err
			}
		}
	}

	return nil
}

func metakubeProjectEditUser(k *metakubeProviderMeta, pid string, u *models.User) error {
	p := users.NewEditUserInProjectParams()
	p.SetProjectID(pid)
	p.SetUserID(u.ID)
	p.SetBody(u)
	_, err := k.client.Users.EditUserInProject(p, k.auth)
	if err != nil {
		if e, ok := err.(*users.EditUserInProjectDefault); ok && errorMessage(e.Payload) != "" {
			return fmt.Errorf("edit user in project errored: %s", errorMessage(e.Payload))
		}
		return fmt.Errorf("edit user in project errored: %v", err)
	}
	return nil
}

func metakubeProjectDeleteUser(k *metakubeProviderMeta, pid, uid string) error {
	p := users.NewDeleteUserFromProjectParams()
	p.SetProjectID(pid)
	p.SetUserID(uid)
	_, err := k.client.Users.DeleteUserFromProject(p, k.auth)
	if err != nil {
		if e, ok := err.(*users.DeleteUserFromProjectDefault); ok && errorMessage(e.Payload) != "" {
			return fmt.Errorf("delete user from project: %s", errorMessage(e.Payload))
		}
		return fmt.Errorf("delete user from project: %v", err)
	}
	return nil
}

func metakubeProjectAddUser(k *metakubeProviderMeta, pid string, u *models.User) error {
	p := users.NewAddUserToProjectParams()
	p.SetProjectID(pid)
	p.SetBody(u)
	if _, err := k.client.Users.AddUserToProject(p, k.auth); err != nil {
		if e, ok := err.(*users.AddUserToProjectDefault); ok && errorMessage(e.Payload) != "" {
			return fmt.Errorf("add user to project: %s", errorMessage(e.Payload))
		}
		return fmt.Errorf("add user to project: %v", err)
	}
	return nil
}

func metakubeProjectCurrentUser(k *metakubeProviderMeta) (*models.User, error) {
	r, err := k.client.Users.GetCurrentUser(users.NewGetCurrentUserParams(), k.auth)
	if err != nil {
		if e, ok := err.(*users.GetCurrentUserDefault); ok && errorMessage(e.Payload) != "" {
			return nil, fmt.Errorf("get current user errored: %s", errorMessage(e.Payload))
		}
		return nil, fmt.Errorf("get current user errored: %v", err)
	}
	return r.Payload, nil
}

func metakubeProjectPersistedUsers(k *metakubeProviderMeta, id string) (map[string]models.User, error) {
	listStateConf := &resource.StateChangeConf{
		Pending: []string{
			usersUnavailable,
		},
		Target: []string{
			usersReady,
		},
		Refresh: func() (interface{}, string, error) {
			p := users.NewGetUsersForProjectParams()
			p.SetProjectID(id)

			r, err := k.client.Users.GetUsersForProject(p, k.auth)
			if err != nil {
				// wait for the RBACs
				if _, ok := err.(*users.GetUsersForProjectForbidden); ok {
					return r, usersUnavailable, nil
				}
				return nil, usersUnavailable, fmt.Errorf("get users for project error: %v", err)
			}
			ret := make(map[string]models.User)
			for _, p := range r.Payload {
				ret[p.Email] = *p
			}
			return ret, usersReady, nil
		},
		Timeout: 10 * time.Second,
		Delay:   5 * requestDelay,
	}

	rawUsers, err := listStateConf.WaitForState()
	if err != nil {
		k.log.Debugf("error while waiting for the users %v", err)
		return nil, fmt.Errorf("error while waiting for the users %v", err)
	}
	users := rawUsers.(map[string]models.User)

	return users, nil
}

func metakubeProjectConfiguredUsers(d *schema.ResourceData) map[string]models.User {
	ret := make(map[string]models.User)
	users := d.Get("user").(*schema.Set).List()
	for _, u := range users {
		u := u.(map[string]interface{})
		ret[u["email"].(string)] = models.User{Email: u["email"].(string), Projects: []*models.ProjectGroup{
			{
				GroupPrefix: u["group"].(string),
				ID:          d.Id(),
			},
		}}
	}
	return ret
}

func resourceProjectDelete(d *schema.ResourceData, m interface{}) error {
	k := m.(*metakubeProviderMeta)
	p := project.NewDeleteProjectParams()
	_, err := k.client.Project.DeleteProject(p.WithProjectID(d.Id()), k.auth)
	if err != nil {
		return fmt.Errorf("unable to delete project '%s': %s", d.Id(), getErrorResponse(err))
	}

	return resource.Retry(d.Timeout(schema.TimeoutDelete), func() *resource.RetryError {
		p := project.NewGetProjectParams()
		r, err := k.client.Project.GetProject(p.WithProjectID(d.Id()), k.auth)
		if err != nil {
			e, ok := err.(*project.GetProjectDefault)
			if ok && (e.Code() == http.StatusForbidden || e.Code() == http.StatusNotFound) {
				k.log.Debugf("project '%s' has been destroyed, returned http code: %d", d.Id(), e.Code())
				return nil
			}
			return resource.NonRetryableError(err)
		}
		k.log.Debugf("project '%s' deletion in progress, deletionTimestamp: %s, status: %s",
			d.Id(), r.Payload.DeletionTimestamp.String(), r.Payload.Status)
		return resource.RetryableError(
			fmt.Errorf("project '%s' still exists, currently in '%s' state", d.Id(), r.Payload.Status),
		)
	})
}
