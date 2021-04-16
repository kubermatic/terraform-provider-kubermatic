package metakube

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/syseleven/go-metakube/client/project"
	"github.com/syseleven/go-metakube/client/serviceaccounts"
	"github.com/syseleven/go-metakube/models"
)

func metakubeResourceServiceAccount() *schema.Resource {
	return &schema.Resource{
		CreateContext: metakubeResourceServiceAccountCreate,
		ReadContext:   metakubeResourceServiceAccountRead,
		UpdateContext: metakubeResourceServiceAccountUpdate,
		DeleteContext: metakubeResourceServiceAccountDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Reference project identifier",
			},
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.NoZeroValues,
				Description:  "Service account's name",
			},
			"group": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringInSlice([]string{"editors", "viewers"}, false),
				Description:  "Service account's role in the project",
				// Group stored with random string suffix which is not stored by provider
				// and prefix which is either "editors-" or "viewers-"
				DiffSuppressFunc: metakubeResourceServiceAccountGroupDiffSuppress,
			},
		},
	}
}

func metakubeResourceServiceAccountGroupDiffSuppress(k, old, new string, d *schema.ResourceData) bool {
	if old == "" || new == "" {
		return false
	}
	l := len(old)
	if len(new) < l {
		l = len(new)
	}
	return old[:l] == new[:l]
}

func metakubeResourceServiceAccountCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	meta := m.(*metakubeProviderMeta)
	p := serviceaccounts.NewAddServiceAccountToProjectParams()
	p.SetContext(ctx)
	p.SetProjectID(d.Get("project_id").(string))
	p.SetBody(&models.ServiceAccount{
		Name:  d.Get("name").(string),
		Group: d.Get("group").(string),
	})
	r, err := meta.client.Serviceaccounts.AddServiceAccountToProject(p, meta.auth)
	if err != nil {
		return diag.Errorf("unable to create service account: %s", stringifyResponseError(err))
	}
	d.SetId(r.Payload.ID)
	return metakubeResourceServiceAccountRead(ctx, d, meta)
}

func metakubeResourceServiceAccountRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	meta := m.(*metakubeProviderMeta)
	projectID := d.Get("project_id").(string)
	if projectID == "" {
		var err error
		projectID, err = metakubeResourceServiceAccountFindProjectID(ctx, d.Id(), meta)
		if err != nil {
			return diag.FromErr(err)
		}
		if projectID == "" {
			meta.log.Debug("did not find owner project id, treat this resource as deleted")
			d.SetId("")
			return nil
		}
		d.Set("project_id", projectID)
	}
	serviceAccounts, err := metakubeResourceServiceAccountListAll(ctx, meta, projectID, d.Timeout(schema.TimeoutRead))
	if err != nil {
		return diag.FromErr(err)
	}
	var serviceAccount *models.ServiceAccount
	for _, sa := range serviceAccounts {
		if sa.ID == d.Id() {
			serviceAccount = sa
			break
		}
	}
	if serviceAccount == nil {
		d.SetId("")
		return nil
	}
	_ = d.Set("name", serviceAccount.Name)
	_ = d.Set("group", serviceAccount.Group)
	return nil
}

func metakubeResourceServiceAccountListAll(ctx context.Context, meta *metakubeProviderMeta, projectID string, timeout time.Duration) ([]*models.ServiceAccount, error) {
	const pending = "Unavailable"
	const target = "Ready"
	listStateConf := &resource.StateChangeConf{
		Pending: []string{
			pending,
		},
		Target: []string{
			target,
		},
		Refresh: func() (interface{}, string, error) {
			p := serviceaccounts.NewListServiceAccountsParams()
			p.SetContext(ctx)
			p.SetProjectID(projectID)
			s, err := meta.client.Serviceaccounts.ListServiceAccounts(p, meta.auth)
			if err != nil {
				// wait for the RBACs
				if _, ok := err.(*serviceaccounts.ListServiceAccountsForbidden); ok {
					return s, pending, nil
				}
				return nil, pending, fmt.Errorf("can not get service accounts: %v", err)
			}
			return s, target, nil
		},
		Timeout: timeout,
		Delay:   time.Second,
	}
	s, err := listStateConf.WaitForStateContext(ctx)
	if err != nil {
		meta.log.Debugf("error waiting for service account %v", err)
		return nil, fmt.Errorf("error waiting for service account %v", err)
	}
	sa := s.(*serviceaccounts.ListServiceAccountsOK)
	return sa.Payload, nil
}

func metakubeResourceServiceAccountUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	meta := m.(*metakubeProviderMeta)
	projectID := d.Get("project_id").(string)
	p := serviceaccounts.NewUpdateServiceAccountParams()
	p.SetContext(ctx)
	p.SetProjectID(projectID)
	p.SetServiceAccountID(d.Id())
	p.SetBody(&models.ServiceAccount{
		ID:    d.Id(),
		Name:  d.Get("name").(string),
		Group: d.Get("group").(string),
	})
	_, err := meta.client.Serviceaccounts.UpdateServiceAccount(p, meta.auth)
	if err != nil {
		return diag.Errorf("unable to update service account: %v", stringifyResponseError(err))
	}
	return metakubeResourceServiceAccountRead(ctx, d, meta)
}

func metakubeResourceServiceAccountDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	meta := m.(*metakubeProviderMeta)
	projectID := d.Get("project_id").(string)
	p := serviceaccounts.NewDeleteServiceAccountParams()
	p.SetContext(ctx)
	p.SetProjectID(projectID)
	p.SetServiceAccountID(d.Id())
	_, err := meta.client.Serviceaccounts.DeleteServiceAccount(p, meta.auth)
	if err != nil {
		return diag.Errorf("unable to delete service account: %v", stringifyResponseError(err))
	}
	return nil
}

func metakubeResourceServiceAccountFindProjectID(ctx context.Context, id string, meta *metakubeProviderMeta) (string, error) {
	r, err := meta.client.Project.ListProjects(project.NewListProjectsParams(), meta.auth)
	if err != nil {
		return "", fmt.Errorf("list projects: %v", err)
	}

	for _, project := range r.Payload {
		ok, err := metakubeResourceServiceAccountBelongsToProject(ctx, project.ID, id, meta)
		if ok {
			return project.ID, nil
		}
		if err != nil {
			return "", err
		}
	}

	meta.log.Info("owner project for service account with id(%s) not found", id)
	return "", nil
}

func metakubeResourceServiceAccountBelongsToProject(ctx context.Context, prj, id string, meta *metakubeProviderMeta) (bool, error) {
	r, err := meta.client.Serviceaccounts.ListServiceAccounts(serviceaccounts.NewListServiceAccountsParams().WithProjectID(prj), meta.auth)
	if err != nil {
		meta.log.Debugf("lookup owner project: list serviceaccounts: %v", err)
		return false, fmt.Errorf("list service accounts: %v", err)
	}
	for _, sa := range r.Payload {
		if sa.ID == id {
			return true, nil
		}
	}
	return false, nil
}
