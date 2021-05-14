package metakube

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/syseleven/go-metakube/client/project"
	"github.com/syseleven/go-metakube/client/users"
	"github.com/syseleven/go-metakube/models"
)

const (
	projectSchemaName                        = "name"
	projectSchemaLabels                      = "labels"
	projectSchemaUsers                       = "user"
	projectSchemaStatus                      = "status"
	projectSchemaCreationTimestamp           = "creation_timestamp"
	projectSchemaDeletionTimestamp           = "deletion_timestamp"
	projectUserSchemaEmail                   = "email"
	projectUserSchemaGroup                   = "group"
	projectEnsureFlawlessCreateUUIDLabelName = "terraform-provider-metakube/ensure-flawless"
)

func metakubeResourceProject() *schema.Resource {
	return &schema.Resource{
		CreateContext: metakubeResourceProjectCreate,
		ReadContext:   metakubeResourceProjectRead,
		UpdateContext: metakubeResourceProjectUpdate,
		DeleteContext: metakubeResourceProjectDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			projectSchemaName: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Project name",
				ValidateFunc: validation.NoZeroValues,
			},

			projectSchemaLabels: {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "Project labels",
				Elem:        schema.TypeString,
			},

			projectSchemaUsers: {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "Project user",
				Elem:        projectUsersSchema(),
			},

			projectSchemaStatus: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Status represents the current state of the project",
			},

			projectSchemaCreationTimestamp: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Creation timestamp",
			},

			projectSchemaDeletionTimestamp: {
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
			projectUserSchemaEmail: {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.NoZeroValues,
				Description:  "User's email address",
			},

			projectUserSchemaGroup: {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringInSlice([]string{"owners", "editors", "viewers"}, false),
				Description:  "User's role in the project",
			},
		},
	}
}

func metakubeResourceProjectCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	k := m.(*metakubeProviderMeta)

	p := project.NewCreateProjectParams()
	p.SetContext(ctx)
	createUUID, err := uuid.GenerateUUID()
	if err != nil {
		return diag.FromErr(err)
	}
	labels := metakubeProjectConfiguredLabels(d)
	// HACK: (https://github.com/syseleven/terraform-provider-metakube/issues/26) API misbehaves and sometimes returns
	// 403 or 500 on project creation. This is API issue related to RBAC initialization. To create illusion of flawless
	// creation, we mack project with unique label and even if create return an error, we attempt to find project with\
	// that label.
	labels[projectEnsureFlawlessCreateUUIDLabelName] = createUUID
	p.SetBody(project.CreateProjectBody{
		Name:   d.Get(projectSchemaName).(string),
		Labels: labels,
	})
	_, err = k.client.Project.CreateProject(p, k.auth)
	if err != nil {
		k.log.Error(stringifyResponseError(err))
	}

	// Wait for project active status
	projectID, err := metakubeProjectWaitForActiveStatus(ctx, d, createUUID, k)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(projectID)

	// HACK: (https://github.com/syseleven/terraform-provider-metakube/issues/26) delete "ensure flawless" label
	if err := metakubeResourceProjectUpdateNameAndLabels(ctx, d, m); err != nil {
		return diag.Errorf("delete '%s' label: %v", projectEnsureFlawlessCreateUUIDLabelName, err)
	}

	ret := metakubeResourceProjectRead(ctx, d, m)

	if v, ok := d.GetOk(projectSchemaUsers); ok {
		if vv, ok := v.(*schema.Set); ok && vv.Len() > 0 {
			return append(ret, diag.Diagnostic{
				Severity:      diag.Error,
				Summary:       "MetaKube API Tokens ability to manage users is not available. We are working on fixing this.",
				AttributePath: cty.GetAttrPath(projectSchemaUsers),
			})
		}
	}
	return ret
}

func metakubeProjectWaitForActiveStatus(ctx context.Context, d *schema.ResourceData, createUUID string, k *metakubeProviderMeta) (string, error) {
	const statusActive = "Active"
	var projectID string
	err := resource.RetryContext(ctx, d.Timeout(schema.TimeoutCreate), func() *resource.RetryError {
		p := project.NewListProjectsParams().WithContext(ctx)
		ret, err := k.client.Project.ListProjects(p, k.auth)
		if err != nil {
			return resource.RetryableError(fmt.Errorf("list projects: %v", err))
		}

		var found *models.Project
		for _, prj := range ret.Payload {
			if prj.Labels != nil && prj.Labels[projectEnsureFlawlessCreateUUIDLabelName] == createUUID {
				found = prj
				break
			}
		}
		if found == nil {
			return resource.RetryableError(fmt.Errorf("project is not found"))
		}
		if found.Status != statusActive {
			return resource.RetryableError(fmt.Errorf("project is: %s", found.Status))
		}
		projectID = found.ID
		return nil
	})
	return projectID, err
}

func metakubeProjectConfiguredLabels(d *schema.ResourceData) map[string]string {
	if l, ok := d.GetOk(projectSchemaLabels); ok {
		ret := make(map[string]string)
		att := l.(map[string]interface{})
		for key, val := range att {
			ret[key] = val.(string)
		}
		return ret
	}
	return nil
}

func metakubeResourceProjectRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	k := m.(*metakubeProviderMeta)
	p := project.NewGetProjectParams()
	p.WithContext(ctx)
	p.SetProjectID(d.Id())

	r, err := k.client.Project.GetProject(p, k.auth)
	if err != nil {
		if e, ok := err.(*project.GetProjectDefault); ok && (e.Code() == http.StatusForbidden || e.Code() == http.StatusNotFound) {
			// remove a project from terraform state file that a user does not have access or does not exist
			comment := fmt.Sprintf("removing project '%s' from terraform state file, code '%d' has been returned", d.Id(), e.Code())
			k.log.Info(comment)
			d.SetId("")
			return nil
		}

		return diag.Errorf("unable to get project '%s': %s", d.Id(), stringifyResponseError(err))
	}

	if err := d.Set(projectSchemaLabels, r.Payload.Labels); err != nil {
		return diag.Diagnostics{{
			Severity:      diag.Error,
			Summary:       fmt.Sprintf("Can't set value: %v", err),
			AttributePath: cty.GetAttrPath(projectSchemaLabels),
		}}
	}
	_ = d.Set(projectSchemaName, r.Payload.Name)
	_ = d.Set(projectSchemaStatus, r.Payload.Status)
	_ = d.Set(projectSchemaCreationTimestamp, r.Payload.CreationTimestamp.String())
	_ = d.Set(projectSchemaDeletionTimestamp, r.Payload.DeletionTimestamp.String())

	var ret diag.Diagnostics

	projectUsers, err := metakubeProjectUsers(ctx, k, d.Id())
	if err != nil {
		ret = append(ret, diag.Diagnostic{
			Severity:      diag.Warning,
			Summary:       "MetaKube API Tokens ability to manage users is not available. We are working on fixing this.",
			AttributePath: cty.GetAttrPath(projectSchemaUsers),
		})
	}

	curUser, err := metakubeProjectCurrentUser(ctx, k)
	if err != nil {
		return append(ret, diag.Diagnostic{
			Severity:      diag.Error,
			Summary:       err.Error(),
			AttributePath: cty.GetAttrPath(projectSchemaUsers),
		})
	}
	if err := d.Set(projectSchemaUsers, flattenedProjectUsers(curUser, projectUsers)); err != nil {
		return append(ret, diag.Diagnostic{
			Severity:      diag.Error,
			Summary:       fmt.Sprintf("Can't set value: %v", err),
			AttributePath: cty.GetAttrPath(projectSchemaUsers),
		})
	}
	return ret
}

func flattenedProjectUsers(cur *models.User, u map[string]models.User) *schema.Set {
	var items []interface{}
	for _, v := range u {
		if v.Email == cur.Email {
			continue
		}
		items = append(items, map[string]interface{}{
			projectUserSchemaEmail: v.Email,
			projectUserSchemaGroup: v.Projects[0].GroupPrefix,
		})
	}
	return schema.NewSet(schema.HashResource(projectUsersSchema()), items)
}

func metakubeResourceProjectUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	err := metakubeResourceProjectUpdateNameAndLabels(ctx, d, m)
	if err != nil {
		return diag.Errorf("unable to update project '%s': %s", d.Id(), stringifyResponseError(err))
	}

	ret := metakubeResourceProjectRead(ctx, d, m)
	if d.HasChange(projectSchemaUsers) {
		return append(ret, diag.Diagnostic{
			Severity:      diag.Error,
			Summary:       "MetaKube API Tokens ability to manage users is not available. We are working on fixing this.",
			AttributePath: cty.GetAttrPath(projectSchemaUsers),
		})
	}
	return ret
}

func metakubeResourceProjectUpdateNameAndLabels(ctx context.Context, d *schema.ResourceData, m interface{}) error {
	k := m.(*metakubeProviderMeta)
	p := project.NewUpdateProjectParams().WithContext(ctx)
	p.Body = &models.Project{
		// name is always required for update requests, otherwise bad request returns
		Name:   d.Get(projectSchemaName).(string),
		Labels: metakubeProjectConfiguredLabels(d),
	}

	_, err := k.client.Project.UpdateProject(p.WithProjectID(d.Id()), k.auth)
	return err
}

func metakubeProjectUpdateUsers(ctx context.Context, k *metakubeProviderMeta, d *schema.ResourceData) error {
	curUser, err := metakubeProjectCurrentUser(ctx, k)
	if err != nil {
		return err
	}

	persistedUsers, err := metakubeProjectUsers(ctx, k, d.Id())
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
				if err := metakubeProjectEditUser(ctx, k, d.Id(), &pu); err != nil {
					return err
				}
			}
			continue
		}
		if err := metakubeProjectDeleteUser(ctx, k, d.Id(), pu.ID); err != nil {
			return err
		}
	}

	for email, cu := range configuredUsers {
		if _, ok := persistedUsers[email]; !ok {
			if err := metakubeProjectAddUser(ctx, k, d.Id(), &cu); err != nil {
				return err
			}
		}
	}

	return nil
}

func metakubeProjectEditUser(ctx context.Context, k *metakubeProviderMeta, pid string, u *models.User) error {
	p := users.NewEditUserInProjectParams()
	p.SetContext(ctx)
	p.SetProjectID(pid)
	p.SetUserID(u.ID)
	p.SetBody(u)
	_, err := k.client.Users.EditUserInProject(p, k.auth)
	if err != nil {
		return fmt.Errorf("edit user in project errored: %v", stringifyResponseError(err))
	}
	return nil
}

func metakubeProjectDeleteUser(ctx context.Context, k *metakubeProviderMeta, pid, uid string) error {
	p := users.NewDeleteUserFromProjectParams()
	p.SetContext(ctx)
	p.SetProjectID(pid)
	p.SetUserID(uid)
	_, err := k.client.Users.DeleteUserFromProject(p, k.auth)
	if err != nil {
		return fmt.Errorf("delete user from project: %v", stringifyResponseError(err))
	}
	return nil
}

func metakubeProjectAddUser(ctx context.Context, k *metakubeProviderMeta, pid string, u *models.User) error {
	p := users.NewAddUserToProjectParams()
	p.SetProjectID(pid)
	p.SetContext(ctx)
	p.SetBody(u)
	if _, err := k.client.Users.AddUserToProject(p, k.auth); err != nil {
		return fmt.Errorf("add user to project: %v", stringifyResponseError(err))
	}
	return nil
}

func metakubeProjectCurrentUser(ctx context.Context, k *metakubeProviderMeta) (*models.User, error) {
	r, err := k.client.Users.GetCurrentUser(users.NewGetCurrentUserParams().WithContext(ctx), k.auth)
	if err != nil {
		return nil, fmt.Errorf("get current user errored: %v", stringifyResponseError(err))
	}
	return r.Payload, nil
}

func metakubeProjectUsers(ctx context.Context, k *metakubeProviderMeta, id string) (map[string]models.User, error) {
	const (
		pending = "Unavailable"
		target  = "Ready"
	)
	listStateConf := &resource.StateChangeConf{
		Pending: []string{pending},
		Target:  []string{target},
		Refresh: func() (interface{}, string, error) {
			p := users.NewGetUsersForProjectParams()
			p.SetContext(ctx)
			p.SetProjectID(id)

			r, err := k.client.Users.GetUsersForProject(p, k.auth)
			if err != nil {
				return nil, pending, fmt.Errorf("%v", stringifyResponseError(err))
			}
			ret := make(map[string]models.User)
			for _, p := range r.Payload {
				ret[p.Email] = *p
			}
			return ret, target, nil
		},
		Timeout: 10 * time.Second,
		Delay:   5 * requestDelay,
	}

	rawUsers, err := listStateConf.WaitForStateContext(ctx)
	if err != nil {
		k.log.Debugf("error while waiting for the users %v", err)
		return nil, err
	}
	u := rawUsers.(map[string]models.User)

	return u, nil
}

func metakubeProjectConfiguredUsers(d *schema.ResourceData) map[string]models.User {
	ret := make(map[string]models.User)
	for _, u := range d.Get(projectSchemaUsers).(*schema.Set).List() {
		u := u.(map[string]interface{})
		ret[u[projectUserSchemaEmail].(string)] = models.User{Email: u[projectUserSchemaEmail].(string), Projects: []*models.ProjectGroup{
			{
				GroupPrefix: u[projectUserSchemaGroup].(string),
				ID:          d.Id(),
			},
		}}
	}
	return ret
}

func metakubeResourceProjectDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	k := m.(*metakubeProviderMeta)
	p := project.NewDeleteProjectParams()
	p.SetContext(ctx)
	_, err := k.client.Project.DeleteProject(p.WithProjectID(d.Id()), k.auth)
	if err != nil {
		if e, ok := err.(*project.DeleteProjectDefault); ok && e.Code() == http.StatusNotFound {
			k.log.Warnf("project '%s' was not found", d.Id())
			return nil
		}
		return diag.Errorf("unable to delete project '%s': %s", d.Id(), stringifyResponseError(err))
	}

	err = resource.RetryContext(ctx, d.Timeout(schema.TimeoutDelete), func() *resource.RetryError {
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
	if err != nil {
		return diag.FromErr(err)
	}
	return nil
}
