package kubermatic

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/kubermatic/go-kubermatic/client/tokens"
	"github.com/kubermatic/go-kubermatic/models"
)

func resourceServiceAccountToken() *schema.Resource {
	return &schema.Resource{
		Create: resourceServiceAccountTokenCreate,
		Read:   resourceServiceAccountTokenRead,
		Update: resourceServiceAccountTokenUpdate,
		Delete: resourceServiceAccountTokenDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
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

func kubermaticServiceAccountTokenMakeID(p, s, t string) string {
	return fmt.Sprint(p, ":", s, ":", t)
}

func kubermaticServiceAccountTokenParseID(id string) (string, string, string, error) {
	parts := strings.SplitN(id, ":", 3)
	if parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return "", "", "", fmt.Errorf("unexpected format of ID (%s), expected project_id:service_account_id", id)
	}
	return parts[0], parts[1], parts[2], nil
}

func resourceServiceAccountTokenCreate(d *schema.ResourceData, m interface{}) error {
	k := m.(*kubermaticProviderMeta)

	projectID, serviceAccountID, err := kubermaticServiceAccountParseID(d.Get("service_account_id").(string))
	if err != nil {
		return err
	}

	p := tokens.NewAddTokenToServiceAccountParams()
	p.SetProjectID(projectID)
	p.SetServiceAccountID(serviceAccountID)
	p.SetBody(&models.ServiceAccountToken{
		Name: d.Get("name").(string),
	})
	r, err := k.client.Tokens.AddTokenToServiceAccount(p, k.auth)
	if err != nil {
		if e, ok := err.(*tokens.AddTokenToServiceAccountDefault); ok && errorMessage(e.Payload) != "" {
			return fmt.Errorf("unable to create token: %s", errorMessage(e.Payload))
		}
		return fmt.Errorf("unable to create token: %v", err)
	}
	d.Set("token", r.Payload.Token)
	d.SetId(kubermaticServiceAccountTokenMakeID(projectID, serviceAccountID, r.Payload.ID))
	return resourceServiceAccountTokenRead(d, m)
}

func resourceServiceAccountTokenRead(d *schema.ResourceData, m interface{}) error {
	k := m.(*kubermaticProviderMeta)

	projectID, serviceAccountID, tokenID, err := kubermaticServiceAccountTokenParseID(d.Id())
	if err != nil {
		return err
	}

	p := tokens.NewListServiceAccountTokensParams()
	p.SetProjectID(projectID)
	p.SetServiceAccountID(serviceAccountID)
	r, err := k.client.Tokens.ListServiceAccountTokens(p, k.auth)
	if err != nil {
		if e, ok := err.(*tokens.ListServiceAccountTokensDefault); ok && errorMessage(e.Payload) != "" {
			return fmt.Errorf("unable to get token: %s", errorMessage(e.Payload))
		}
		return fmt.Errorf("unable to get token: %v", err)
	}

	var token *models.PublicServiceAccountToken
	for _, v := range r.Payload {
		if v.ID == tokenID {
			token = v
			break
		}
	}
	if token == nil {
		d.SetId("")
		return nil
	}

	d.Set("name", token.Name)
	d.Set("creation_timestamp", token.CreationTimestamp.String())
	d.Set("expiry", token.Expiry.String())
	return nil
}

func resourceServiceAccountTokenUpdate(d *schema.ResourceData, m interface{}) error {
	// TODO(furkhat): Fix go-kubermatic client PatchServiceAccountTokenParams structure
	// k := m.(*kubermaticProviderMeta)

	// projectID, serviceAccountID, tokenID, err := kubermaticServiceAccountTokenParseID(d.Id())
	// if err != nil {
	// 	return err
	// }
	// p := tokens.NewPatchServiceAccountTokenParams()
	// p.SetProjectID(projectID)
	// p.SetServiceAccountID(serviceAccountID)
	// p.SetTokenID(tokenID)
	// // Only name is editable
	// p.SetBody([]uint8(fmt.Sprintf(`{name:"%s"}`, d.Get("name").(string))))
	// _, err = k.client.Tokens.PatchServiceAccountToken(p, k.auth)
	// if err != nil {
	// 	if e, ok := err.(*tokens.PatchServiceAccountTokenDefault); ok && errorMessage(e.Payload) != "" {
	// 		return fmt.Errorf("failed to update token: %s", errorMessage(e.Payload))
	// 	}
	// 	return fmt.Errorf("failed to update token: %v", err)
	// }
	return resourceServiceAccountTokenRead(d, m)
}

func resourceServiceAccountTokenDelete(d *schema.ResourceData, m interface{}) error {
	k := m.(*kubermaticProviderMeta)

	projectID, serviceAccountID, tokenID, err := kubermaticServiceAccountTokenParseID(d.Id())
	if err != nil {
		return err
	}

	p := tokens.NewDeleteServiceAccountTokenParams()
	p.SetProjectID(projectID)
	p.SetServiceAccountID(serviceAccountID)
	p.SetTokenID(tokenID)
	_, err = k.client.Tokens.DeleteServiceAccountToken(p, k.auth)
	if err != nil {
		if e, ok := err.(*tokens.DeleteServiceAccountTokenDefault); ok && errorMessage(e.Payload) != "" {
			return fmt.Errorf("failed to delete token: %s", errorMessage(e.Payload))
		}
		return fmt.Errorf("failed to delete token: %v", err)
	}
	return nil
}
