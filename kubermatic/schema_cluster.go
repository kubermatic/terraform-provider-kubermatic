package kubermatic

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func clusterSpecFields() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"version": {
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
		},
		"cloud": {
			Type:     schema.TypeList,
			Required: true,
			ForceNew: true,
			MaxItems: 1,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"dc": {
						Type:     schema.TypeString,
						Required: true,
					},
					"bringyourown": {
						Type:     schema.TypeList,
						Optional: true,
						MaxItems: 1,
						Elem:     &schema.Resource{},
					},
					"aws": {
						Type:     schema.TypeList,
						Optional: true,
						MaxItems: 1,
						Elem: &schema.Resource{
							Schema: awsCloudSpecFields(),
						},
					},
				},
			},
		},
		"machine_networks": {
			Type:     schema.TypeList,
			Optional: true,
			ForceNew: true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"cidr": {
						Type:     schema.TypeString,
						Optional: true,
					},
					"gateway": {
						Type:     schema.TypeString,
						Optional: true,
					},
					"dns_servers": {
						Type:     schema.TypeSet,
						Optional: true,
						Elem: &schema.Schema{
							Type: schema.TypeString,
						},
					},
				},
			},
		},
		"audit_logging": {
			Type:     schema.TypeList,
			Optional: true,
			MaxItems: 1,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"enabled": {
						Type:     schema.TypeBool,
						Optional: true,
						Default:  false,
					},
				},
			},
		},
	}
}

func awsCloudSpecFields() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"access_key_id": {
			Type:      schema.TypeString,
			Required:  true,
			Sensitive: true,
		},
		"secret_access_key": {
			Type:      schema.TypeString,
			Required:  true,
			Sensitive: true,
		},
		"vpc_id": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"security_group_id": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"route_table_id": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"instance_profile_name": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"role_arn": {
			Type:     schema.TypeString,
			Optional: true,
		},
	}
}
