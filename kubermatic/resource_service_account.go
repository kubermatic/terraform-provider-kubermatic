package kubermatic

import (
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/kubermatic/go-kubermatic/client/serviceaccounts"
	"github.com/kubermatic/go-kubermatic/models"
)

const (
	serviceAccountReady       = "Ready"
	serviceAccountUnavailable = "Unavailable"
)

func resourceServiceAccount() *schema.Resource {
	return &schema.Resource{
		Create: resourceServiceAccountCreate,
		Read:   resourceServiceAccountRead,
		Update: resourceServiceAccountUpdate,
		Delete: resourceServiceAccountDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
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

func kubermaticServiceAccountMakeID(p, id string) string {
	return fmt.Sprint(p, ":", id)
}

func kubermaticServiceAccountParseID(id string) (string, string, error) {
	parts := strings.SplitN(id, ":", 2)
	if parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("unexpected format of ID (%s), expected project_id:id", id)
	}
	return parts[0], parts[1], nil
}

func resourceServiceAccountCreate(d *schema.ResourceData, m interface{}) error {
	k := m.(*kubermaticProviderMeta)

	p := serviceaccounts.NewAddServiceAccountToProjectParams()
	p.SetProjectID(d.Get("project_id").(string))
	p.SetBody(&models.ServiceAccount{
		Name:  d.Get("name").(string),
		Group: d.Get("group").(string),
	})
	r, err := k.client.Serviceaccounts.AddServiceAccountToProject(p, k.auth)
	if err != nil {
		if e, ok := err.(*serviceaccounts.AddServiceAccountToProjectDefault); ok && errorMessage(e.Payload) != "" {
			return fmt.Errorf("unable to create service account: %s", errorMessage(e.Payload))
		}
		return fmt.Errorf("unable to create service account: %v", err)
	}
	d.SetId(kubermaticServiceAccountMakeID(d.Get("project_id").(string), r.Payload.ID))

	return resourceServiceAccountRead(d, m)
}

func resourceServiceAccountRead(d *schema.ResourceData, m interface{}) error {
	k := m.(*kubermaticProviderMeta)

	projectID, serviceAccountID, err := kubermaticServiceAccountParseID(d.Id())
	if err != nil {
		return err
	}

	all, err := kubermaticServiceAccountList(k, projectID)
	if err != nil {
		return err
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

	d.Set("name", serviceAccount.Name)
	d.Set("group", serviceAccount.Group)

	return nil
}

func kubermaticServiceAccountList(k *kubermaticProviderMeta, projectID string) ([]*models.ServiceAccount, error) {
	listStateConf := &resource.StateChangeConf{
		Pending: []string{
			serviceAccountUnavailable,
		},
		Target: []string{
			serviceAccountReady,
		},
		Refresh: func() (interface{}, string, error) {
			p := serviceaccounts.NewListServiceAccountsParams()
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

	s, err := listStateConf.WaitForState()
	if err != nil {
		k.log.Debugf("error while waiting for the service account %v", err)
		return nil, fmt.Errorf("error while waiting for the service account %v", err)
	}
	sa := s.(*serviceaccounts.ListServiceAccountsOK)

	return sa.Payload, nil
}

func resourceServiceAccountUpdate(d *schema.ResourceData, m interface{}) error {
	// TODO(furkhat): uncomment when fix to `assignment to entry in nil map` released.
	// k := m.(*kubermaticProviderMeta)

	// projectID, serviceAccountID, err := kubermaticServiceAccountParseID(d.Id())
	// if err != nil {
	// 	return err
	// }

	// p := serviceaccounts.NewUpdateServiceAccountParams()
	// p.SetProjectID(projectID)
	// p.SetServiceAccountID(serviceAccountID)
	// p.SetBody(&models.ServiceAccount{m
	// 	ID:    serviceAccountID,
	// 	Name:  d.Get("name").(string),
	// 	Group: d.Get("group").(string),
	// })
	// _, err = k.client.Serviceaccounts.UpdateServiceAccount(p, k.auth)
	// if err != nil {
	// 	if e, ok := err.(*serviceaccounts.UpdateServiceAccountDefault); ok && errorMessage(e.Payload) != "" {
	// 		return fmt.Errorf("unable to update service account: %s", errorMessage(e.Payload))
	// 	}
	// 	return fmt.Errorf("unable to update service account: %v", err)
	// }
	return resourceServiceAccountRead(d, m)
}

func resourceServiceAccountDelete(d *schema.ResourceData, m interface{}) error {
	k := m.(*kubermaticProviderMeta)

	projectID, serviceAccountID, err := kubermaticServiceAccountParseID(d.Id())
	if err != nil {
		return err
	}

	p := serviceaccounts.NewDeleteServiceAccountParams()
	p.SetProjectID(projectID)
	p.SetServiceAccountID(serviceAccountID)
	_, err = k.client.Serviceaccounts.DeleteServiceAccount(p, k.auth)
	if err != nil {
		if e, ok := err.(*serviceaccounts.DeleteServiceAccountDefault); ok && errorMessage(e.Payload) != "" {
			return fmt.Errorf("unable to delete service account: %s", errorMessage(e.Payload))
		}
		return fmt.Errorf("unable to delete service account: %v", err)
	}
	return nil
}
