package kubermatic

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
)

func clusterSpecFields() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"version": {
			Type:     schema.TypeString,
			Required: true,
			// TODO(furkhat): upgrade
			ForceNew:    true,
			Description: "Cluster version",
		},
		"cloud": {
			Type:        schema.TypeList,
			Required:    true,
			ForceNew:    true,
			MinItems:    1,
			MaxItems:    1,
			Description: "Cloud specification",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"dc": {
						Type:        schema.TypeString,
						ForceNew:    true,
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
					"openstack": {
						Type:        schema.TypeList,
						Optional:    true,
						MaxItems:    1,
						Description: "OpenStack cluster specification",
						Elem: &schema.Resource{
							Schema: openstackCloudSpecFields(),
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
						Elem:        schema.TypeString,
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
			Sensitive:   true,
			Description: "Access key identifier",
		},
		"secret_access_key": {
			Type:        schema.TypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Secret access key",
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

func openstackCloudSpecFields() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"tenant": {
			Type:         schema.TypeString,
			Required:     true,
			ForceNew:     true,
			ValidateFunc: validation.NoZeroValues,
		},
		"username": {
			Type:         schema.TypeString,
			Required:     true,
			ForceNew:     true,
			Sensitive:    true,
			ValidateFunc: validation.NoZeroValues,
		},
		"password": {
			Type:         schema.TypeString,
			Required:     true,
			ForceNew:     true,
			Sensitive:    true,
			ValidateFunc: validation.NoZeroValues,
		},
		"floating_ip_pool": {
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
		},
	}
}
