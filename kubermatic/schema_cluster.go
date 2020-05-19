package kubermatic

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func clusterSpecFields() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"version": {
			Type:        schema.TypeString,
			Required:    true,
			ForceNew:    true,
			Description: "Cluster version",
		},
		"cloud": {
			Type:        schema.TypeList,
			Required:    true,
			ForceNew:    true,
			MaxItems:    1,
			Description: "Cloud specification",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"dc": {
						Type:        schema.TypeString,
						Required:    true,
						Description: "Data center name",
					},
					"bringyourown": {
						Type:        schema.TypeList,
						Optional:    true,
						MaxItems:    1,
						Elem:        &schema.Resource{},
						Description: "Bring your own infrastructure",
					},
					"aws": {
						Type:        schema.TypeList,
						Optional:    true,
						MaxItems:    1,
						Description: "AWS cluster specification",
						Elem: &schema.Resource{
							Schema: awsCloudSpecFields(),
						},
					},
				},
			},
		},
		"machine_networks": {
			Type:        schema.TypeList,
			Optional:    true,
			ForceNew:    true,
			Description: "Machine networks optionally specifies the parameters for IPAM",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"cidr": {
						Type:        schema.TypeString,
						Optional:    true,
						Description: "Network CIDR",
					},
					"gateway": {
						Type:        schema.TypeString,
						Optional:    true,
						Description: "Network gateway",
					},
					"dns_servers": {
						Type:        schema.TypeSet,
						Optional:    true,
						Description: "DNS servers",
						Elem: &schema.Schema{
							Type: schema.TypeString,
						},
					},
				},
			},
		},
		"audit_logging": {
			Type:        schema.TypeList,
			Optional:    true,
			MaxItems:    1,
			Description: "Audit logging settings",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"enabled": {
						Type:        schema.TypeBool,
						Optional:    true,
						Default:     false,
						Description: "Enable audit logging",
					},
				},
			},
		},
	}
}

func awsCloudSpecFields() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"access_key_id": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "Access key identifier",
			DefaultFunc: schema.EnvDefaultFunc("AWS_ACCESS_KEY_ID", ""),
		},
		"secret_access_key": {
			Type:        schema.TypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Secret access key",
			DefaultFunc: schema.EnvDefaultFunc("AWS_SECRET_ACCESS_KEY", ""),
		},
		"vpc_id": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Virtual private cloud identifier",
		},
		"security_group_id": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Security group identifier",
		},
		"route_table_id": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Route table identifier",
		},
		"instance_profile_name": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Instance profile name",
		},
		"role_arn": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The IAM role the control plane will use over assume-role",
		},
	}
}
