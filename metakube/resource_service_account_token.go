package metakube

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/syseleven/terraform-provider-metakube/go-metakube/client/tokens"
	"github.com/syseleven/terraform-provider-metakube/go-metakube/models"
)

const (
	serviceAccountTokenReady       = "Ready"
	serviceAccountTokenUnavailable = "Unavailable"
)

func resourceServiceAccountToken() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceServiceAccountTokenCreate,
		ReadContext:   resourceServiceAccountTokenRead,
		UpdateContext: resourceServiceAccountTokenUpdate,
		DeleteContext: resourceServiceAccountTokenDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"service_account_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Service account full identifier of format project_id:service_account_id",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Resource name",
			},
			"creation_timestamp": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Creation timestamp",
			},
			"expiry": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Expiration timestamp",
			},
			"token": {
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
				Description: "Token value",
			},
		},
	}
}

func metakubeServiceAccountTokenMakeID(p, s, t string) string {
	return fmt.Sprint(p, ":", s, ":", t)
}

func metakubeServiceAccountTokenParseID(id string) (string, string, string, error) {
	parts := strings.SplitN(id, ":", 3)
	if parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return "", "", "", fmt.Errorf("unexpected format of ID (%s), expected project_id:service_account_id", id)
	}
	return parts[0], parts[1], parts[2], nil
}

func resourceServiceAccountTokenCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	k := m.(*metakubeProviderMeta)

	projectID, serviceAccountID, err := metakubeServiceAccountParseID(d.Get("service_account_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	p := tokens.NewAddTokenToServiceAccountParams()
	p.SetContext(ctx)
	p.SetProjectID(projectID)
	p.SetServiceAccountID(serviceAccountID)
	p.SetBody(&models.ServiceAccountToken{
		Name: d.Get("name").(string),
	})
	r, err := k.client.Tokens.AddTokenToServiceAccount(p, k.auth)
	if err != nil {
		return diag.Errorf("unable to create token: %v", getErrorResponse(err))
	}
	_ = d.Set("token", r.Payload.Token)
	d.SetId(metakubeServiceAccountTokenMakeID(projectID, serviceAccountID, r.Payload.ID))
	return resourceServiceAccountTokenRead(ctx, d, m)
}

func resourceServiceAccountTokenRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	k := m.(*metakubeProviderMeta)
	projectID, serviceAccountID, tokenID, err := metakubeServiceAccountTokenParseID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	listStateConf := &resource.StateChangeConf{
		Pending: []string{
			serviceAccountTokenUnavailable,
		},
		Target: []string{
			serviceAccountTokenReady,
		},
		Refresh: func() (interface{}, string, error) {
			p := tokens.NewListServiceAccountTokensParams()
			p.SetProjectID(projectID)
			p.SetServiceAccountID(serviceAccountID)
			t, err := k.client.Tokens.ListServiceAccountTokens(p, k.auth)
			if err != nil {
				// wait for the RBACs
				if _, ok := err.(*tokens.ListServiceAccountTokensForbidden); ok {
					return t, serviceAccountTokenUnavailable, nil
				}
				return nil, serviceAccountTokenUnavailable, err
			}
			return t, serviceAccountTokenReady, nil
		},
		Timeout: 30 * time.Second,
		Delay:   requestDelay,
	}

	s, err := listStateConf.WaitForState()
	if err != nil {
		k.log.Debugf("error while waiting for the tokens: %v", err)
		return diag.Errorf("error while waiting for the tokens: %v", err)
	}
	saTokens := s.(*tokens.ListServiceAccountTokensOK)
	var token *models.PublicServiceAccountToken
	for _, v := range saTokens.Payload {
		if v.ID == tokenID {
			token = v
			break
		}
	}
	if token == nil {
		d.SetId("")
		return nil
	}

	_ = d.Set("name", token.Name)
	_ = d.Set("creation_timestamp", token.CreationTimestamp.String())
	_ = d.Set("expiry", token.Expiry.String())
	return nil
}

func resourceServiceAccountTokenUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	k := m.(*metakubeProviderMeta)

	projectID, serviceAccountID, tokenID, err := metakubeServiceAccountTokenParseID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	p := tokens.NewPatchServiceAccountTokenParams()
	p.SetProjectID(projectID)
	p.SetServiceAccountID(serviceAccountID)
	p.SetTokenID(tokenID)
	// Only name is editable
	name := d.Get("name").(string)
	bodyStr := fmt.Sprintf("{\"name\":\"%s\"}", name)
	p.SetBody([]byte(bodyStr))
	_, err = k.client.Tokens.PatchServiceAccountToken(p, k.auth)
	if err != nil {
		return diag.Errorf("failed to update token: %s", getErrorResponse(err))
	}
	return resourceServiceAccountTokenRead(ctx, d, m)
}

func resourceServiceAccountTokenDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	k := m.(*metakubeProviderMeta)

	projectID, serviceAccountID, tokenID, err := metakubeServiceAccountTokenParseID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	p := tokens.NewDeleteServiceAccountTokenParams()
	p.SetProjectID(projectID)
	p.SetServiceAccountID(serviceAccountID)
	p.SetTokenID(tokenID)
	_, err = k.client.Tokens.DeleteServiceAccountToken(p, k.auth)
	if err != nil {
		return diag.Errorf("failed to delete token: %v", getErrorResponse(err))
	}
	return nil
}
