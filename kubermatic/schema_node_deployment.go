package kubermatic

import (
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func isLabelOrTagReserved(path string) bool {
	r := regexp.MustCompile(`(tags|labels)\.(system|kubernetes\.io)\/`)
	return r.MatchString(path)
}

func validateLabelOrTag(key string) error {
	r := regexp.MustCompile(`^(system|kubernetes\.io)\/`)
	if r.MatchString(key) {
		return fmt.Errorf("forbidden tag or label prefix %s", key)
	}
	return nil
}

func nodeDeploymentSpecFields() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"replicas": {
			Type:        schema.TypeInt,
			Optional:    true,
			Default:     1,
			Description: "Number of replicas",
		},
		"template": {
			Type:        schema.TypeList,
			MaxItems:    1,
			Required:    true,
			Description: "Template specification",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"cloud": {
						Type:        schema.TypeList,
						MaxItems:    1,
						Required:    true,
						Description: "Cloud specification",
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"bringyourown": {
									Optional:    true,
									Type:        schema.TypeMap,
									Description: "Bring your own infrastructure",
									Elem:        schema.TypeString,
								},
								"aws": {
									Type:        schema.TypeList,
									Optional:    true,
									MaxItems:    1,
									Description: "AWS node deployment specification",
									Elem: &schema.Resource{
										Schema: awsNodeFields(),
									},
								},
							},
						},
					},
					"operating_system": {
						Type:        schema.TypeList,
						Optional:    true,
						MaxItems:    1,
						Description: "Operating system",
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								// TODO: add missing operating systems
								"ubuntu": {
									Type:        schema.TypeList,
									Optional:    true,
									MaxItems:    1,
									Description: "Ubuntu operating system",
									Elem: &schema.Resource{
										Schema: map[string]*schema.Schema{
											"dist_upgrade_on_boot": {
												Type:        schema.TypeBool,
												Optional:    true,
												Default:     false,
												Description: "Upgrade operating system on boot",
											},
										},
									},
								},
							},
						},
					},
					"versions": {
						Type:        schema.TypeList,
						Optional:    true,
						Computed:    true,
						MaxItems:    1,
						Description: "Cloud components versions",
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"kubelet": {
									Type:        schema.TypeString,
									Optional:    true,
									Computed:    true,
									Description: "Kubelet version",
								},
							},
						},
					},
					"labels": {
						Type:     schema.TypeMap,
						Optional: true,
						Computed: true,
						Description: "Map of string keys and values that can be used to organize and categorize (scope and select) objects. " +
							"It will be applied to Nodes allowing users run their apps on specific Node using labelSelector.",
						Elem: schema.TypeString,
						DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
							return isLabelOrTagReserved(k)
						},
						ValidateFunc: func(v interface{}, k string) (strings []string, errors []error) {
							l := v.(map[string]interface{})
							for key := range l {
								if err := validateLabelOrTag(key); err != nil {
									errors = append(errors, err)
								}
							}
							return
						},
					},
					"taints": {
						Type:        schema.TypeList,
						Optional:    true,
						Description: "List of taints to set on new nodes",
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"effect": {
									Type:        schema.TypeString,
									Optional:    true,
									Description: "Taint effect",
								},
								"key": {
									Type:        schema.TypeString,
									Optional:    true,
									Description: "Taint key",
								},
								"value": {
									Type:        schema.TypeString,
									Optional:    true,
									Description: "Taint value",
								},
							},
						},
					},
				},
			},
		},
	}
}

func awsNodeFields() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"instance_type": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "EC2 instance type",
		},
		"disk_size": {
			Type:        schema.TypeInt,
			Required:    true,
			Description: "Size of the volume in GBs. Only one volume will be created",
		},
		"volume_type": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "EBS volume type",
		},
		"availability_zone": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "Availability zone in which to place the node. It is coupled with the subnet to which the node will belong",
		},
		"subnet_id": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "The VPC subnet to which the node shall be connected",
		},
		"assign_public_ip": {
			Type:     schema.TypeBool,
			Optional: true,
			Default:  true,
			Description: "Flag which controls a property of the AWS instance. When set the AWS instance will get a public IP address " +
				"assigned during launch overriding a possible setting in the used AWS subnet.",
		},
		"ami": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Amazon Machine Image to use. Will be defaulted to an AMI of your selected operating system and region",
		},
		"tags": {
			Type:        schema.TypeMap,
			Optional:    true,
			Computed:    true,
			Description: "Additional instance tags",
			Elem:        schema.TypeString,
			DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
				return isLabelOrTagReserved(k)
			},
			ValidateFunc: func(v interface{}, k string) (strings []string, errors []error) {
				l := v.(map[string]interface{})
				for key := range l {
					if err := validateLabelOrTag(key); err != nil {
						errors = append(errors, err)
					}
				}
				return
			},
		},
	}
}
