package kubermatic

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/kubermatic/go-kubermatic/client/project"
	"github.com/kubermatic/go-kubermatic/models"
)

func resourceSSHKey() *schema.Resource {
	return &schema.Resource{
		Create: resourceSSHKeyCreate,
		Read:   resourceSSHKeyRead,
		Delete: resourceSSHKeyDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.NoZeroValues,
				ForceNew:     true,
			},
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.NoZeroValues,
				ForceNew:     true,
			},
			"public_key": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.NoZeroValues,
				DiffSuppressFunc: func(_, old, new string, _ *schema.ResourceData) bool {
					return strings.TrimSpace(old) == strings.TrimSpace(new)
				},
				ForceNew: true,
			},
		},
	}
}

func resourceSSHKeyCreate(d *schema.ResourceData, m interface{}) error {
	k := m.(*kubermaticProviderMeta)
	p := project.NewCreateSSHKeyParams()
	p.SetProjectID(d.Get("project_id").(string))
	p.Key = &models.SSHKey{
		Name: d.Get("name").(string),
		Spec: &models.SSHKeySpec{
			PublicKey: d.Get("public_key").(string),
		},
	}
	created, err := k.client.Project.CreateSSHKey(p, k.auth)
	if err != nil {
		return fmt.Errorf("unable to create SSH key: %s", getErrorResponse(err))
	}
	d.SetId(created.Payload.ID)
	return resourceSSHKeyRead(d, m)
}

func resourceSSHKeyRead(d *schema.ResourceData, m interface{}) error {
	k := m.(*kubermaticProviderMeta)
	p := project.NewListSSHKeysParams()
	p.SetProjectID(d.Get("project_id").(string))
	ret, err := k.client.Project.ListSSHKeys(p, k.auth)
	if err != nil {
		return fmt.Errorf("unable to list SSH keys: %s", getErrorResponse(err))
	}
	var sshkey *models.SSHKey
	for _, r := range ret.Payload {
		if r.ID == d.Id() {
			sshkey = r
			break
		}
	}
	if sshkey == nil {
		d.SetId("")
		return nil
	}
	d.Set("name", sshkey.Name)
	d.Set("public_key", sshkey.Spec.PublicKey)
	return nil
}

func resourceSSHKeyDelete(d *schema.ResourceData, m interface{}) error {
	k := m.(*kubermaticProviderMeta)
	p := project.NewDeleteSSHKeyParams()
	p.SetProjectID(d.Get("project_id").(string))
	p.SetSSHKeyID(d.Id())
	_, err := k.client.Project.DeleteSSHKey(p, k.auth)
	return fmt.Errorf("unable to delete SSH key: %s", getErrorResponse(err))
}
