package metakube

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/syseleven/terraform-provider-metakube/go-metakube/client/serviceaccounts"
	"github.com/syseleven/terraform-provider-metakube/go-metakube/models"
)

const (
	serviceAccountReady       = "Ready"
	serviceAccountUnavailable = "Unavailable"
)

func resourceServiceAccount() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceServiceAccountCreate,
		ReadContext:   resourceServiceAccountRead,
		UpdateContext: resourceServiceAccountUpdate,
		DeleteContext: resourceServiceAccountDelete,
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
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					if old == "" || new == "" {
						return false
					}
					l := len(old)
					if len(new) < l {
						l = len(new)
					}
					return old[:l] == new[:l]
				},
			},
		},
	}
}

func metakubeServiceAccountMakeID(p, id string) string {
	return fmt.Sprint(p, ":", id)
}

func metakubeServiceAccountParseID(id string) (string, string, error) {
	parts := strings.SplitN(id, ":", 2)
	if parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("unexpected format of ID (%s), expected project_id:id", id)
	}
	return parts[0], parts[1], nil
}

func resourceServiceAccountCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	k := m.(*metakubeProviderMeta)

	p := serviceaccounts.NewAddServiceAccountToProjectParams()
	p.SetContext(ctx)
	p.SetProjectID(d.Get("project_id").(string))
	p.SetBody(&models.ServiceAccount{
		Name:  d.Get("name").(string),
		Group: d.Get("group").(string),
	})
	r, err := k.client.Serviceaccounts.AddServiceAccountToProject(p, k.auth)
	if err != nil {
		return diag.Errorf("unable to create service account: %s", getErrorResponse(err))
	}
	d.SetId(metakubeServiceAccountMakeID(d.Get("project_id").(string), r.Payload.ID))

	return resourceServiceAccountRead(ctx, d, m)
}

func resourceServiceAccountRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	k := m.(*metakubeProviderMeta)

	projectID, serviceAccountID, err := metakubeServiceAccountParseID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	all, err := metakubeServiceAccountList(ctx, k, projectID)
	if err != nil {
		return diag.FromErr(err)
	}

	var serviceAccount *models.ServiceAccount
	for _, item := range all {
		if item.ID == serviceAccountID {
			serviceAccount = item
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

func metakubeServiceAccountList(ctx context.Context, k *metakubeProviderMeta, projectID string) ([]*models.ServiceAccount, error) {
	listStateConf := &resource.StateChangeConf{
		Pending: []string{
			serviceAccountUnavailable,
		},
		Target: []string{
			serviceAccountReady,
		},
		Refresh: func() (interface{}, string, error) {
			p := serviceaccounts.NewListServiceAccountsParams()
			p.SetContext(ctx)
			p.SetProjectID(projectID)
			s, err := k.client.Serviceaccounts.ListServiceAccounts(p, k.auth)
			if err != nil {
				// wait for the RBACs
				if _, ok := err.(*serviceaccounts.ListServiceAccountsForbidden); ok {
					return s, usersUnavailable, nil
				}
				return nil, serviceAccountUnavailable, fmt.Errorf("can not get service accounts: %v", err)
			}
			return s, serviceAccountReady, nil
		},
		Timeout: 20 * time.Second,
		Delay:   requestDelay,
	}

	s, err := listStateConf.WaitForStateContext(ctx)
	if err != nil {
		k.log.Debugf("error while waiting for the service account %v", err)
		return nil, fmt.Errorf("error while waiting for the service account %v", err)
	}
	sa := s.(*serviceaccounts.ListServiceAccountsOK)

	return sa.Payload, nil
}

func resourceServiceAccountUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	k := m.(*metakubeProviderMeta)

	projectID, serviceAccountID, err := metakubeServiceAccountParseID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	p := serviceaccounts.NewUpdateServiceAccountParams()
	p.SetContext(ctx)
	p.SetProjectID(projectID)
	p.SetServiceAccountID(serviceAccountID)
	p.SetBody(&models.ServiceAccount{
		ID:    serviceAccountID,
		Name:  d.Get("name").(string),
		Group: d.Get("group").(string),
	})
	_, err = k.client.Serviceaccounts.UpdateServiceAccount(p, k.auth)
	if err != nil {
		return diag.Errorf("unable to update service account: %v", getErrorResponse(err))
	}
	return resourceServiceAccountRead(ctx, d, m)
}

func resourceServiceAccountDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	k := m.(*metakubeProviderMeta)

	projectID, serviceAccountID, err := metakubeServiceAccountParseID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	p := serviceaccounts.NewDeleteServiceAccountParams()
	p.SetContext(ctx)
	p.SetProjectID(projectID)
	p.SetServiceAccountID(serviceAccountID)
	_, err = k.client.Serviceaccounts.DeleteServiceAccount(p, k.auth)
	if err != nil {
		return diag.Errorf("unable to delete service account: %v", getErrorResponse(err))
	}
	return nil
}
