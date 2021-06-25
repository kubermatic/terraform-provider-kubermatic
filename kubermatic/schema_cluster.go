package kubermatic

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
)

func clusterSpecFields() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"version": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "Cloud orchestrator version, either Kubernetes or OpenShift",
		},
		"cloud": {
			Type:        schema.TypeList,
			Required:    true,
			ForceNew:    true,
			MinItems:    1,
			MaxItems:    1,
			Description: "Cloud provider specification",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
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
					"azure": azureCloudSpecSchema(),
				},
			},
		},
		"enable_user_ssh_key_agent": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     true,
			Description: "Enable user ssh key Agent",
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
		"opa_integration": {
			Type:        schema.TypeList,
			Optional:    true,
			Description: "Enable OPA Integration",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"enabled": {
						Type:     schema.TypeBool,
						Optional: true,
						Default:  false,
					},
					"webhook_timeout_seconds": {
						Type:     schema.TypeInt,
						Optional: true,
						Default:  0,
					},
				},
			},
		},
		"mla": {
			Type:        schema.TypeList,
			Optional:    true,
			Description: "Enable MLA Feature",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"logging_enabled": {
						Type:     schema.TypeBool,
						Optional: true,
						Default:  false,
					},
					"monitoring_enabled": {
						Type:     schema.TypeBool,
						Optional: true,
						Default:  false,
					},
				},
			},
		},
		"audit_logging": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
			Description: "Whether to enable audit logging or not",
		},
		"use_pod_node_selector_admission_plugin": {
			Type:     schema.TypeBool,
			Optional: true,
			Default:  false,
		},
		"use_pod_security_policy_admission_plugin": {
			Type:     schema.TypeBool,
			Optional: true,
			Default:  false,
		},
	}
}

func azureCloudSpecSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Optional:    true,
		ForceNew:    true,
		Description: "Azire cluster specification",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"availability_set": {
					Type:     schema.TypeString,
					Optional: true,
				},
				"client_id": {
					Type:     schema.TypeString,
					Required: true,
				},
				"client_secret": {
					Type:      schema.TypeString,
					Required:  true,
					Sensitive: true,
				},
				"subscription_id": {
					Type:     schema.TypeString,
					Required: true,
				},
				"tenant_id": {
					Type:     schema.TypeString,
					Required: true,
				},
				"resource_group": {
					Type:     schema.TypeString,
					Computed: true,
					Optional: true,
				},
				"route_table": {
					Type:     schema.TypeString,
					Computed: true,
					Optional: true,
				},
				"security_group": {
					Type:     schema.TypeString,
					Computed: true,
					Optional: true,
				},
				"subnet": {
					Type:     schema.TypeString,
					Computed: true,
					Optional: true,
				},
				"vnet": {
					Type:     schema.TypeString,
					Computed: true,
					Optional: true,
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
		"floating_ip_pool": {
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
		},
		"network": {
			Type:     schema.TypeString,
			ForceNew: true,
			Optional: true,
		},
		"router_id": {
			Type:     schema.TypeString,
			ForceNew: true,
			Optional: true,
		},
		"security_groups": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"subnet_id": {
			Type:     schema.TypeString,
			ForceNew: true,
			Optional: true,
		},
		"tenant": {
			Type:         schema.TypeString,
			Required:     true,
			ForceNew:     true,
			ValidateFunc: validation.NoZeroValues,
		},
		"username": {
			Type:         schema.TypeString,
			Required:     true,
			Sensitive:    true,
			ValidateFunc: validation.NoZeroValues,
		},
		"password": {
			Type:         schema.TypeString,
			Required:     true,
			Sensitive:    true,
			ValidateFunc: validation.NoZeroValues,
		},
	}
}

func kubernetesConfigSchema() *schema.Schema {
	return &schema.Schema{
		Type:      schema.TypeList,
		MaxItems:  1,
		Computed:  true,
		Sensitive: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"raw_config": {
					Type:     schema.TypeString,
					Computed: true,
				},
			},
		},
	}
}
