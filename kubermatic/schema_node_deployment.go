package kubermatic

import (
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func isReservedLabelOrTag(path string) bool {
	r := regexp.MustCompile(`(tags|labels)\.(system|kubernetes\.io)\/`)
	return r.MatchString(path)
}

func validateTagOrLabel(key string) error {
	r := regexp.MustCompile(`^(system|kubernetes\.io)\/`)
	if r.MatchString(key) {
		return fmt.Errorf("forbidden tag or label prefix %s", key)
	}
	return nil
}

func nodeDeploymentSpecFields() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"replicas": {
			Type:     schema.TypeInt,
			Optional: true,
			Default:  1,
		},
		"template": {
			Type:     schema.TypeList,
			MaxItems: 1,
			Required: true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"cloud": {
						Type:     schema.TypeList,
						MaxItems: 1,
						Required: true,
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"bringyourown": {
									Optional: true,
									Type:     schema.TypeMap,
									Elem: &schema.Schema{
										Type: schema.TypeString,
									},
								},
								"aws": {
									Type:     schema.TypeList,
									Optional: true,
									MaxItems: 1,
									Elem: &schema.Resource{
										Schema: awsNodeFields(),
									},
								},
							},
						},
					},
					"operating_system": {
						Type:     schema.TypeList,
						Optional: true,
						MaxItems: 1,
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								// TODO: add missing operating systems
								"ubuntu": {
									Type:     schema.TypeList,
									Optional: true,
									MaxItems: 1,
									Elem: &schema.Resource{
										Schema: map[string]*schema.Schema{
											"dist_upgrade_on_boot": {
												Type:     schema.TypeBool,
												Optional: true,
												Default:  false,
											},
										},
									},
								},
							},
						},
					},
					"versions": {
						Type:     schema.TypeList,
						Optional: true,
						Computed: true,
						MaxItems: 1,
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"kubelet": {
									Type:     schema.TypeString,
									Optional: true,
									Computed: true,
								},
							},
						},
					},
					"labels": {
						Type:     schema.TypeMap,
						Optional: true,
						Computed: true,
						Elem: &schema.Schema{
							Type: schema.TypeString,
						},
						DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
							return isReservedLabelOrTag(k)
						},
						ValidateFunc: func(v interface{}, k string) (strings []string, errors []error) {
							l := v.(map[string]interface{})
							for key := range l {
								if err := validateTagOrLabel(key); err != nil {
									errors = append(errors, err)
								}
							}
							return
						},
					},
					"taints": {
						Type:     schema.TypeList,
						Optional: true,
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"effect": {
									Type:     schema.TypeString,
									Optional: true,
								},
								"key": {
									Type:     schema.TypeString,
									Optional: true,
								},
								"value": {
									Type:     schema.TypeString,
									Optional: true,
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
			Type:     schema.TypeString,
			Required: true,
		},
		"disk_size": {
			Type:     schema.TypeInt,
			Required: true,
		},
		"volume_type": {
			Type:     schema.TypeString,
			Required: true,
		},
		"availability_zone": {
			Type:     schema.TypeString,
			Required: true,
		},
		"subnet_id": {
			Type:     schema.TypeString,
			Required: true,
		},
		"assign_public_ip": {
			Type:     schema.TypeBool,
			Optional: true,
			Default:  true,
		},
		"ami": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"tags": {
			Type:     schema.TypeMap,
			Optional: true,
			Computed: true,
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
			DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
				return isReservedLabelOrTag(k)
			},
			ValidateFunc: func(v interface{}, k string) (strings []string, errors []error) {
				l := v.(map[string]interface{})
				for key := range l {
					if err := validateTagOrLabel(key); err != nil {
						errors = append(errors, err)
					}
				}
				return
			},
		},
	}
}
