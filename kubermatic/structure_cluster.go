package kubermatic

import (
	"github.com/kubermatic/go-kubermatic/models"
)

// flatteners

func flattenClusterSpec(in *models.ClusterSpec) []interface{} {
	if in == nil {
		return []interface{}{}
	}

	att := make(map[string]interface{})

	if in.Version != nil {
		att["version"] = in.Version
	}

	if len(in.MachineNetworks) > 0 {
		att["machine_networks"] = flattenMachineNetworks(in.MachineNetworks)
	}

	if in.AuditLogging != nil {
		att["audit_logging"] = []interface{}{
			map[string]interface{}{
				"enabled": in.AuditLogging.Enabled,
			},
		}
	}

	if in.Cloud != nil {
		att["cloud"] = flattenClusterCloudSpec(in.Cloud)
	}

	return []interface{}{att}
}

func flattenMachineNetworks(in []*models.MachineNetworkingConfig) []interface{} {
	if len(in) < 1 {
		return []interface{}{}
	}

	att := make([]interface{}, len(in))

	for i, v := range in {
		m := make(map[string]interface{})

		if v.CIDR != "" {
			m["cidr"] = v.CIDR
		}
		if v.Gateway != "" {
			m["gateway"] = v.Gateway
		}
		if l := len(v.DNSServers); l > 0 {
			ds := make([]interface{}, l)
			for i, s := range v.DNSServers {
				ds[i] = s
			}
			m["dns_servers"] = ds
		}
		att[i] = m
	}

	return att
}

func flattenClusterCloudSpec(in *models.CloudSpec) []interface{} {
	if in == nil {
		return []interface{}{}
	}

	att := make(map[string]interface{})

	if in.DatacenterName != "" {
		att["dc"] = in.DatacenterName
	}

	if in.Bringyourown != nil {
		att["bringyourown"] = []interface{}{in.Bringyourown}
	}

	if in.Aws != nil {
		att["aws"] = flattenAWSCloudSpec(in.Aws)
	}

	return []interface{}{att}
}

func flattenAWSCloudSpec(in *models.AWSCloudSpec) []interface{} {
	if in == nil {
		return []interface{}{}
	}

	att := make(map[string]interface{})

	if in.AccessKeyID != "" {
		att["access_key_id"] = in.AccessKeyID
	}

	if in.SecretAccessKey != "" {
		att["secret_access_key"] = in.SecretAccessKey
	}

	if in.VPCID != "" {
		att["vpc_id"] = in.VPCID
	}

	if in.SecurityGroupID != "" {
		att["security_group_id"] = in.SecurityGroupID
	}

	if in.InstanceProfileName != "" {
		att["instance_profile_name"] = in.InstanceProfileName
	}

	if in.ControlPlaneRoleARN != "" {
		att["role_arn"] = in.ControlPlaneRoleARN
	}

	if in.RouteTableID != "" {
		att["route_table_id"] = in.RouteTableID
	}

	return []interface{}{att}
}

// expanders

func expandClusterSpec(p []interface{}) *models.ClusterSpec {
	if len(p) < 1 {
		return nil
	}
	obj := &models.ClusterSpec{}
	if p[0] == nil {
		return obj
	}
	in := p[0].(map[string]interface{})

	if v, ok := in["version"]; ok {
		obj.Version = v.(string)
	}

	if v, ok := in["machine_networks"]; ok {
		obj.MachineNetworks = expandMachineNetworks(v.([]interface{}))
	}

	if v, ok := in["audit_logging"]; ok {
		obj.AuditLogging = expandAuditLogging(v.([]interface{}))
	}

	if v, ok := in["cloud"]; ok {
		obj.Cloud = expandClusterCloudSpec(v.([]interface{}))
	}

	return obj
}

func expandMachineNetworks(p []interface{}) []*models.MachineNetworkingConfig {
	if len(p) < 1 {
		return nil
	}
	var machines []*models.MachineNetworkingConfig
	for _, elem := range p {
		in := elem.(map[string]interface{})
		obj := &models.MachineNetworkingConfig{}

		if v, ok := in["cidr"]; ok {
			obj.CIDR = v.(string)
		}

		if v, ok := in["gateway"]; ok {
			obj.Gateway = v.(string)
		}

		if v, ok := in["dns_servers"]; ok {
			for _, s := range v.([]interface{}) {
				obj.DNSServers = append(obj.DNSServers, s.(string))
			}
		}

		machines = append(machines, obj)
	}

	return machines
}

func expandAuditLogging(p []interface{}) *models.AuditLoggingSettings {
	if len(p) < 1 {
		return nil
	}
	obj := &models.AuditLoggingSettings{}
	if p[0] == nil {
		return obj
	}
	in := p[0].(map[string]interface{})

	if v, ok := in["enabled"]; ok {
		obj.Enabled = v.(bool)
	}

	return obj
}

func expandClusterCloudSpec(p []interface{}) *models.CloudSpec {
	if len(p) < 1 {
		return nil
	}
	obj := &models.CloudSpec{}
	if p[0] == nil {
		return obj
	}
	in := p[0].(map[string]interface{})

	if v, ok := in["dc"]; ok {
		obj.DatacenterName = v.(string)
	}

	if v, ok := in["bringyourown"]; ok {
		obj.Bringyourown = expandBringYourOwnCloudSpec(v.([]interface{}))
	}

	if v, ok := in["aws"]; ok {
		obj.Aws = expandAWSCloudSpec(v.([]interface{}))
	}

	return obj
}

func expandBringYourOwnCloudSpec(p []interface{}) models.BringYourOwnCloudSpec {
	if len(p) < 1 {
		return nil
	}
	// just to return json object {}
	return map[string]interface{}{}
}

func expandAWSCloudSpec(p []interface{}) *models.AWSCloudSpec {
	if len(p) < 1 {
		return nil
	}
	obj := &models.AWSCloudSpec{}
	if p[0] == nil {
		return obj
	}
	in := p[0].(map[string]interface{})

	if v, ok := in["access_key_id"]; ok {
		obj.AccessKeyID = v.(string)
	}

	if v, ok := in["secret_access_key"]; ok {
		obj.SecretAccessKey = v.(string)
	}

	if v, ok := in["vpc_id"]; ok {
		obj.VPCID = v.(string)
	}

	if v, ok := in["security_group_id"]; ok {
		obj.SecurityGroupID = v.(string)
	}

	if v, ok := in["instance_profile_name"]; ok {
		obj.InstanceProfileName = v.(string)
	}

	if v, ok := in["role_arn"]; ok {
		obj.ControlPlaneRoleARN = v.(string)
	}

	if v, ok := in["route_table_id"]; ok {
		obj.RouteTableID = v.(string)
	}

	return obj
}
