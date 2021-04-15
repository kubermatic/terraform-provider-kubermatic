package metakube

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/syseleven/terraform-provider-metakube/go-metakube/client/tokens"
	"github.com/syseleven/terraform-provider-metakube/go-metakube/models"
)

func metakubeResourceServiceAccountToken() *schema.Resource {
	return &schema.Resource{
		CreateContext: metakubeResourceServiceAccountTokenCreate,
		ReadContext:   metakubeResourceServiceAccountTokenRead,
		UpdateContext: metakubeResourceServiceAccountTokenUpdate,
		DeleteContext: metakubeResourceServiceAccountTokenDelete,
		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Project id the service account belongs to",
			},
			"service_account_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Service account id token belongs to",
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

func metakubeResourceServiceAccountTokenCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	meta := m.(*metakubeProviderMeta)
	prj, svcacc, err := metakubeResourceServiceAccountTokenParentIDs(ctx, d, meta)
	if err != nil {
		return diag.FromErr(err)
	}
	if prj == "" {
		meta.log.Debug("did not find parent objects, treat as deleted")
		return diag.Errorf("project resource containing service account with id(%s) not found", svcacc)
	}

	parms := tokens.NewAddTokenToServiceAccountParams().
		WithContext(ctx).
		WithProjectID(prj).
		WithServiceAccountID(svcacc).
		WithBody(&models.ServiceAccountToken{
			Name: d.Get("name").(string),
		})
	res, err := meta.client.Tokens.AddTokenToServiceAccount(parms, meta.auth)
	if err != nil {
		return diag.Errorf("create token: %v", stringifyResponseError(err))
	}

	_ = d.Set("token", res.Payload.Token)
	_ = d.Set("project_id", prj)
	d.SetId(res.Payload.ID)
	return metakubeResourceServiceAccountTokenRead(ctx, d, m)
}

func metakubeResourceServiceAccountTokenRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	meta := m.(*metakubeProviderMeta)
	prj, svcacc, err := metakubeResourceServiceAccountTokenParentIDs(ctx, d, meta)
	if err != nil {
		return diag.FromErr(err)
	}
	if prj == "" {
		meta.log.Info("project resource containing service account with id(%s) not found", svcacc)
		d.SetId("")
		return nil
	}
	if svcacc == "" {
		meta.log.Info("owner service account not found")
		d.SetId("")
		return nil
	}

	token, err := metakubeResourceServiceAccountTokenFind(ctx, prj, svcacc, d.Id(), d.Timeout(schema.TimeoutRead), meta)
	if err != nil {
		return diag.FromErr(err)
	}
	if token == nil {
		d.SetId("")
		return nil
	}

	_ = d.Set("project_id", prj)
	_ = d.Set("service_account_id", svcacc)
	_ = d.Set("name", token.Name)
	_ = d.Set("creation_timestamp", token.CreationTimestamp.String())
	_ = d.Set("expiry", token.Expiry.String())
	return nil
}

func metakubeResourceServiceAccountTokenParentIDs(ctx context.Context, d *schema.ResourceData, meta *metakubeProviderMeta) (string, string, error) {
	svcacc := d.Get("service_account_id").(string)
	prj := d.Get("project_id").(string)
	if prj == "" {
		var err error
		prj, err = metakubeResourceServiceAccountFindProjectID(ctx, svcacc, meta)
		if err != nil {
			return "", "", err
		}
	}
	return prj, svcacc, nil
}

func metakubeResourceServiceAccountTokenFind(ctx context.Context, prj, svcacc, id string, timeout time.Duration, meta *metakubeProviderMeta) (*models.PublicServiceAccountToken, error) {
	const (
		pending = "Unavailable"
		target  = "Ready"
	)
	w := &resource.StateChangeConf{
		Pending: []string{pending},
		Target:  []string{target},
		Refresh: func() (interface{}, string, error) {
			p := tokens.NewListServiceAccountTokensParams().
				WithProjectID(prj).
				WithServiceAccountID(svcacc)
			t, err := meta.client.Tokens.ListServiceAccountTokens(p, meta.auth)
			if err != nil {
				// wait for the RBACs
				if _, ok := err.(*tokens.ListServiceAccountTokensForbidden); ok {
					return t, pending, nil
				}
				return nil, pending, err
			}
			return t, target, nil
		},
		Timeout: timeout,
		Delay:   time.Second,
	}

	s, err := w.WaitForState()
	if err != nil {
		meta.log.Debugf("error while waiting for the tokens: %v", err)
		return nil, fmt.Errorf("error while waiting for the tokens: %v", err)
	}
	tokens := s.(*tokens.ListServiceAccountTokensOK)
	for _, v := range tokens.Payload {
		if v.ID == id {
			return v, nil
		}
	}
	return nil, nil
}

func metakubeResourceServiceAccountTokenUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	meta := m.(*metakubeProviderMeta)
	prj := d.Get("project_id").(string)
	svcacc := d.Get("service_account_id").(string)
	p := tokens.NewPatchServiceAccountTokenParams().
		WithProjectID(prj).
		WithServiceAccountID(svcacc).
		WithTokenID(d.Id()).
		WithBody(&models.PublicServiceAccountToken{
			Name: d.Get("name").(string),
		})
	// Only name is editable
	_, err := meta.client.Tokens.PatchServiceAccountToken(p, meta.auth)
	if err != nil {
		return diag.Errorf("resource service_account_token update: %s", stringifyResponseError(err))
	}
	return metakubeResourceServiceAccountTokenRead(ctx, d, m)
}

func metakubeResourceServiceAccountTokenDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	meta := m.(*metakubeProviderMeta)
	prj := d.Get("project_id").(string)
	svcacc := d.Get("service_account_id").(string)
	p := tokens.NewDeleteServiceAccountTokenParams().
		WithProjectID(prj).
		WithServiceAccountID(svcacc).
		WithTokenID(d.Id())
	_, err := meta.client.Tokens.DeleteServiceAccountToken(p, meta.auth)
	if err != nil {
		return diag.Errorf("failed to delete token: %v", stringifyResponseError(err))
	}
	return nil
}
