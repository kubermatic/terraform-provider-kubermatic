package metakube

import (
	"github.com/syseleven/terraform-provider-metakube/go-metakube/models"
)

// flatteners

func flattenClusterSpec(values clusterPreserveValues, in *models.ClusterSpec) []interface{} {
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

	att["audit_logging"] = false
	if in.AuditLogging != nil {
		att["audit_logging"] = in.AuditLogging.Enabled
	}

	att["pod_security_policy"] = in.UsePodSecurityPolicyAdmissionPlugin

	if in.Cloud != nil {
		att["cloud"] = flattenClusterCloudSpec(values, in.Cloud)
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

func flattenClusterCloudSpec(values clusterPreserveValues, in *models.CloudSpec) []interface{} {
	if in == nil {
		return []interface{}{}
	}

	att := make(map[string]interface{})

	if in.Bringyourown != nil {
		att["bringyourown"] = []interface{}{in.Bringyourown}
	}

	if in.Aws != nil {
		att["aws"] = flattenAWSCloudSpec(values.aws)
	}

	if in.Openstack != nil {
		att["openstack"] = flattenOpenstackSpec(values.openstack, in.Openstack)
	}

	if in.Azure != nil {
		att["azure"] = flattenAzureSpec(values.azure)
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

func flattenOpenstackSpec(values *clusterOpenstackPreservedValues, in *models.OpenstackCloudSpec) []interface{} {
	if in == nil {
		return []interface{}{}
	}

	att := make(map[string]interface{})

	if in.FloatingIPPool != "" {
		att["floating_ip_pool"] = in.FloatingIPPool
	}

	if values.openstackTenant != nil {
		att["tenant"] = values.openstackTenant
	}
	if values.openstackUsername != nil {
		att["username"] = values.openstackUsername
	}
	if values.openstackPassword != nil {
		att["password"] = values.openstackPassword
	}

	return []interface{}{att}
}

func flattenAzureSpec(in *models.AzureCloudSpec) []interface{} {
	if in == nil {
		return []interface{}{}
	}

	// API returns empty spec for Azure clusters, so we just preserve values used for creation

	att := make(map[string]interface{})

	if in.AvailabilitySet != "" {
		att["availability_set"] = in.AvailabilitySet
	}

	if in.ClientID != "" {
		att["client_id"] = in.ClientID
	}

	if in.ClientSecret != "" {
		att["client_secret"] = in.ClientSecret
	}

	if in.SubscriptionID != "" {
		att["subscription_id"] = in.SubscriptionID
	}

	if in.TenantID != "" {
		att["tenant_id"] = in.TenantID
	}

	if in.ResourceGroup != "" {
		att["resource_group"] = in.ResourceGroup
	}

	if in.RouteTableName != "" {
		att["route_table"] = in.RouteTableName
	}

	if in.SecurityGroup != "" {
		att["security_group"] = in.SecurityGroup
	}

	if in.SubnetName != "" {
		att["subnet"] = in.SubnetName
	}

	if in.VNetName != "" {
		att["vnet"] = in.VNetName
	}

	return []interface{}{att}
}

// expanders

func expandClusterSpec(p []interface{}, dcName string) *models.ClusterSpec {
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
		obj.AuditLogging = expandAuditLogging(v.(bool))
	}

	if v, ok := in["pod_security_policy"]; ok {
		obj.UsePodSecurityPolicyAdmissionPlugin = v.(bool)
	}

	if v, ok := in["cloud"]; ok {
		obj.Cloud = expandClusterCloudSpec(v.([]interface{}), dcName)
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

func expandAuditLogging(enabled bool) *models.AuditLoggingSettings {
	return &models.AuditLoggingSettings{
		Enabled: enabled,
	}
}

func expandClusterCloudSpec(p []interface{}, dcName string) *models.CloudSpec {
	if len(p) < 1 {
		return nil
	}
	obj := &models.CloudSpec{}
	if p[0] == nil {
		return obj
	}
	in := p[0].(map[string]interface{})

	obj.DatacenterName = dcName

	if v, ok := in["bringyourown"]; ok {
		obj.Bringyourown = expandBringYourOwnCloudSpec(v.([]interface{}))
	}

	if v, ok := in["aws"]; ok {
		obj.Aws = expandAWSCloudSpec(v.([]interface{}))
	}

	if v, ok := in["openstack"]; ok {
		obj.Openstack = expandOpenstackCloudSpec(v.([]interface{}))
	}

	if v, ok := in["azure"]; ok {
		obj.Azure = expandAzureCloudSpec(v.([]interface{}))
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

func expandOpenstackCloudSpec(p []interface{}) *models.OpenstackCloudSpec {
	if len(p) < 1 {
		return nil
	}

	obj := &models.OpenstackCloudSpec{}
	if p[0] == nil {
		return obj
	}
	in := p[0].(map[string]interface{})

	if v, ok := in["tenant"]; ok {
		obj.Tenant = v.(string)
	}

	if v, ok := in["floating_ip_pool"]; ok {
		obj.FloatingIPPool = v.(string)
	}

	if v, ok := in["username"]; ok {
		obj.Username = v.(string)
	}

	if v, ok := in["password"]; ok {
		obj.Password = v.(string)
	}

	// HACK(furkhat): API doesn't return domain for cluster. Use 'Default' all the time.
	obj.Domain = "Default"

	return obj
}

func expandAzureCloudSpec(p []interface{}) *models.AzureCloudSpec {
	if len(p) < 1 {
		return nil
	}

	obj := &models.AzureCloudSpec{}

	if p[0] == nil {
		return obj
	}

	in := p[0].(map[string]interface{})

	if v, ok := in["availability_set"]; ok {
		obj.AvailabilitySet = v.(string)
	}

	if v, ok := in["client_id"]; ok {
		obj.ClientID = v.(string)
	}

	if v, ok := in["client_secret"]; ok {
		obj.ClientSecret = v.(string)
	}

	if v, ok := in["subscription_id"]; ok {
		obj.SubscriptionID = v.(string)
	}

	if v, ok := in["tenant_id"]; ok {
		obj.TenantID = v.(string)
	}

	if v, ok := in["resource_group"]; ok {
		obj.ResourceGroup = v.(string)
	}

	if v, ok := in["route_table"]; ok {
		obj.RouteTableName = v.(string)
	}

	if v, ok := in["security_group"]; ok {
		obj.SecurityGroup = v.(string)
	}

	if v, ok := in["subnet"]; ok {
		obj.SubnetName = v.(string)
	}

	if v, ok := in["vnet"]; ok {
		obj.VNetName = v.(string)
	}

	return obj
}
