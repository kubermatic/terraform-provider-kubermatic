package kubermatic

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kubermatic/go-kubermatic/models"
)

func TestFlattenClusterSpec(t *testing.T) {
	cases := []struct {
		Input          *models.ClusterSpec
		ExpectedOutput []interface{}
	}{
		{
			&models.ClusterSpec{
				Version:                             "1.15.6",
				MachineNetworks:                     nil,
				AuditLogging:                        &models.AuditLoggingSettings{},
				EnableUserSSHKeyAgent:               false,
				UsePodSecurityPolicyAdmissionPlugin: true,
				UsePodNodeSelectorAdmissionPlugin:   true,
				OpaIntegration: &models.OPAIntegrationSettings{
					Enabled:               true,
					WebhookTimeoutSeconds: 0,
				},
				Mla: &models.MLASettings{
					LoggingEnabled:    true,
					MonitoringEnabled: true,
				},
				Cloud: &models.CloudSpec{
					DatacenterName: "eu-west-1",
					Bringyourown:   map[string]interface{}{},
				},
			},
			[]interface{}{
				map[string]interface{}{
					"version":                   "1.15.6",
					"audit_logging":             false,
					"enable_user_ssh_key_agent": false,
					"use_pod_security_policy_admission_plugin": true,
					"use_pod_node_selector_admission_plugin":   true,
					"opa_integration": []interface{}{
						map[string]interface{}{
							"enabled":                 true,
							"webhook_timeout_seconds": int32(0),
						},
					},
					"mla": []interface{}{
						map[string]interface{}{
							"logging_enabled":    true,
							"monitoring_enabled": true,
						},
					},
					"cloud": []interface{}{
						map[string]interface{}{
							"bringyourown": []interface{}{map[string]interface{}{}},
						},
					},
				},
			},
		},
		{
			&models.ClusterSpec{},
			[]interface{}{
				map[string]interface{}{
					"audit_logging":                            false,
					"use_pod_node_selector_admission_plugin":   false,
					"use_pod_security_policy_admission_plugin": false,
					"enable_user_ssh_key_agent":                false,
				},
			},
		},
		{
			nil,
			[]interface{}{},
		},
	}

	for _, tc := range cases {
		output := flattenClusterSpec(clusterPreserveValues{}, tc.Input)
		if diff := cmp.Diff(tc.ExpectedOutput, output); diff != "" {
			t.Fatalf("Unexpected output from expander: mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestFlattenClusterCloudSpec(t *testing.T) {
	cases := []struct {
		Input          *models.CloudSpec
		ExpectedOutput []interface{}
	}{
		{
			&models.CloudSpec{
				Aws: &models.AWSCloudSpec{},
			},
			[]interface{}{
				map[string]interface{}{
					"aws": []interface{}{},
				},
			},
		},
		{
			&models.CloudSpec{
				Bringyourown: map[string]interface{}{},
			},
			[]interface{}{
				map[string]interface{}{
					"bringyourown": []interface{}{
						map[string]interface{}{},
					},
				},
			},
		},
		{
			&models.CloudSpec{},
			[]interface{}{
				map[string]interface{}{},
			},
		},
		{
			nil,
			[]interface{}{},
		},
	}

	for _, tc := range cases {
		output := flattenClusterCloudSpec(clusterPreserveValues{}, tc.Input)
		if diff := cmp.Diff(tc.ExpectedOutput, output); diff != "" {
			t.Fatalf("Unexpected output from expander: mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestFlattenAWSCloudSpec(t *testing.T) {
	cases := []struct {
		Input          *models.AWSCloudSpec
		ExpectedOutput []interface{}
	}{
		{
			&models.AWSCloudSpec{
				AccessKeyID:         "AKIAIOSFODNN7EXAMPLE",
				ControlPlaneRoleARN: "default",
				InstanceProfileName: "default",
				RouteTableID:        "rtb-09ba434c1bEXAMPLE",
				SecretAccessKey:     "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
				SecurityGroupID:     "sg-51530134",
				VPCID:               "e5e4b2ef2fe",
			},
			[]interface{}{
				map[string]interface{}{
					"access_key_id":         "AKIAIOSFODNN7EXAMPLE",
					"role_arn":              "default",
					"instance_profile_name": "default",
					"route_table_id":        "rtb-09ba434c1bEXAMPLE",
					"secret_access_key":     "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
					"security_group_id":     "sg-51530134",
					"vpc_id":                "e5e4b2ef2fe",
				},
			},
		},
		{
			&models.AWSCloudSpec{},
			[]interface{}{
				map[string]interface{}{},
			},
		},
		{
			nil,
			[]interface{}{},
		},
	}

	for _, tc := range cases {
		output := flattenAWSCloudSpec(tc.Input)
		if diff := cmp.Diff(tc.ExpectedOutput, output); diff != "" {
			t.Fatalf("Unexpected output from expander: mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestFlattenOpenstackCloudSpec(t *testing.T) {
	cases := []struct {
		Input          *models.OpenstackCloudSpec
		PreserveValues clusterOpenstackPreservedValues
		ExpectedOutput []interface{}
	}{
		{
			&models.OpenstackCloudSpec{
				FloatingIPPool: "FloatingIPPool",
				Network:        "Network",
				Password:       "",
				RouterID:       "RouterID",
				SecurityGroups: "SecurityGroups",
				SubnetID:       "SubnetID",
				Tenant:         "",
				TenantID:       "TenantID",
				Username:       "",
			},
			clusterOpenstackPreservedValues{
				openstackUsername: "Username",
				openstackPassword: "Password",
				openstackTenant:   "Tenant",
			},
			[]interface{}{
				map[string]interface{}{
					"username":         "Username",
					"password":         "Password",
					"tenant":           "Tenant",
					"floating_ip_pool": "FloatingIPPool",
					"network":          "Network",
					"subnet_id":        "SubnetID",
					"router_id":        "RouterID",
					"security_groups":  "SecurityGroups",
				},
			},
		},
		{
			&models.OpenstackCloudSpec{},
			clusterOpenstackPreservedValues{},
			[]interface{}{
				map[string]interface{}{},
			},
		},
		{
			nil,
			clusterOpenstackPreservedValues{},
			[]interface{}{},
		},
	}

	for _, tc := range cases {
		output := flattenOpenstackSpec(&tc.PreserveValues, tc.Input)
		if diff := cmp.Diff(tc.ExpectedOutput, output); diff != "" {
			t.Fatalf("Unexpected output from expander: mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestFlattenAzureCloudSpec(t *testing.T) {
	cases := []struct {
		Input          *models.AzureCloudSpec
		ExpectedOutput []interface{}
	}{
		{
			&models.AzureCloudSpec{
				ClientID:       "ClientID",
				ClientSecret:   "ClientSecret",
				SubscriptionID: "SubscriptionID",
				TenantID:       "TenantID",
				ResourceGroup:  "ResourceGroup",
				RouteTableName: "RouteTableName",
				SecurityGroup:  "SecurityGroup",
				SubnetName:     "SubnetName",
				VNetName:       "VNetName",
			},
			[]interface{}{
				map[string]interface{}{
					"client_id":       "ClientID",
					"client_secret":   "ClientSecret",
					"tenant_id":       "TenantID",
					"subscription_id": "SubscriptionID",
					"resource_group":  "ResourceGroup",
					"route_table":     "RouteTableName",
					"security_group":  "SecurityGroup",
					"subnet":          "SubnetName",
					"vnet":            "VNetName",
				},
			},
		},
		{
			&models.AzureCloudSpec{},
			[]interface{}{
				map[string]interface{}{},
			},
		},
		{
			nil,
			[]interface{}{},
		},
	}

	for _, tc := range cases {
		output := flattenAzureSpec(tc.Input)
		if diff := cmp.Diff(tc.ExpectedOutput, output); diff != "" {
			t.Fatalf("Unexpected output from expander: mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestFlattenOPAIntegration(t *testing.T) {
	cases := []struct {
		Input          *models.OPAIntegrationSettings
		ExpectedOutput []interface{}
	}{
		{
			&models.OPAIntegrationSettings{
				Enabled:               false,
				WebhookTimeoutSeconds: 40,
			},
			[]interface{}{
				map[string]interface{}{
					"enabled":                 false,
					"webhook_timeout_seconds": int32(40),
				},
			},
		},
	}

	for _, tc := range cases {
		output := flattenOPAIntegration(tc.Input)
		if diff := cmp.Diff(tc.ExpectedOutput, output); diff != "" {
			t.Fatalf("Unexpected output from expander: mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestFlattenMLA(t *testing.T) {
	cases := []struct {
		Input          *models.MLASettings
		ExpectedOutput []interface{}
	}{
		{
			&models.MLASettings{
				LoggingEnabled:    true,
				MonitoringEnabled: true,
			},
			[]interface{}{
				map[string]interface{}{
					"logging_enabled":    true,
					"monitoring_enabled": true,
				},
			},
		},
	}

	for _, tc := range cases {
		output := flattenMLA(tc.Input)
		if diff := cmp.Diff(tc.ExpectedOutput, output); diff != "" {
			t.Fatalf("Unexpected output from expander: mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestFlattenMachineNetwork(t *testing.T) {
	cases := []struct {
		Input          []*models.MachineNetworkingConfig
		ExpectedOutput []interface{}
	}{
		{
			[]*models.MachineNetworkingConfig{
				{
					CIDR:    "192.168.0.0/24",
					Gateway: "192.168.1.1",
					DNSServers: []string{
						"192.200.200.1",
						"192.200.200.201",
					},
				},
			},
			[]interface{}{
				map[string]interface{}{
					"cidr":    "192.168.0.0/24",
					"gateway": "192.168.1.1",
					"dns_servers": []interface{}{
						"192.200.200.1",
						"192.200.200.201",
					},
				},
			},
		},
		{
			[]*models.MachineNetworkingConfig{},
			[]interface{}{},
		},
		{
			nil,
			[]interface{}{},
		},
	}

	for _, tc := range cases {
		output := flattenMachineNetworks(tc.Input)
		if diff := cmp.Diff(tc.ExpectedOutput, output); diff != "" {
			t.Fatalf("Unexpected output from expander: mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestExpandClusterSpec(t *testing.T) {
	cases := []struct {
		Input          []interface{}
		ExpectedOutput *models.ClusterSpec
		DCName         string
	}{
		{
			[]interface{}{
				map[string]interface{}{
					"version":          "1.15.6",
					"machine_networks": []interface{}{},
					"audit_logging":    false,
					"use_pod_security_policy_admission_plugin": true,
					"use_pod_node_selector_admission_plugin":   true,
					"opa_integration": []interface{}{
						map[string]interface{}{
							"enabled":                 false,
							"webhook_timeout_seconds": 10,
						},
					},
					"mla": []interface{}{
						map[string]interface{}{
							"logging_enabled":    true,
							"monitoring_enabled": true,
						},
					},
					"cloud": []interface{}{
						map[string]interface{}{
							"bringyourown": []interface{}{
								map[string]interface{}{},
							},
						},
					},
				},
			},
			&models.ClusterSpec{
				Version:                             "1.15.6",
				MachineNetworks:                     nil,
				AuditLogging:                        &models.AuditLoggingSettings{},
				UsePodSecurityPolicyAdmissionPlugin: true,
				UsePodNodeSelectorAdmissionPlugin:   true,
				OpaIntegration: &models.OPAIntegrationSettings{
					Enabled:               false,
					WebhookTimeoutSeconds: int32(10),
				},
				Mla: &models.MLASettings{
					LoggingEnabled:    true,
					MonitoringEnabled: true,
				},
				Cloud: &models.CloudSpec{
					DatacenterName: "eu-west-1",
					Bringyourown:   map[string]interface{}{},
				},
			},
			"eu-west-1",
		},
		{
			[]interface{}{
				map[string]interface{}{},
			},
			&models.ClusterSpec{},
			"",
		},
		{
			[]interface{}{},
			nil,
			"",
		},
	}

	for _, tc := range cases {
		output := expandClusterSpec(tc.Input, tc.DCName)
		if diff := cmp.Diff(tc.ExpectedOutput, output); diff != "" {
			t.Fatalf("Unexpected output from expander: mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestExpandClusterCloudSpec(t *testing.T) {
	cases := []struct {
		Input          []interface{}
		ExpectedOutput *models.CloudSpec
		DCName         string
	}{
		{
			[]interface{}{
				map[string]interface{}{
					"bringyourown": []interface{}{
						map[string]interface{}{},
					},
				},
			},
			&models.CloudSpec{
				DatacenterName: "eu-west-1",
				Bringyourown:   map[string]interface{}{},
			},
			"eu-west-1",
		},
		{
			[]interface{}{
				map[string]interface{}{
					"aws": []interface{}{
						map[string]interface{}{},
					},
				},
			},
			&models.CloudSpec{
				DatacenterName: "eu-west-1",
				Aws:            &models.AWSCloudSpec{},
			},
			"eu-west-1",
		},
		{
			[]interface{}{
				map[string]interface{}{},
			},
			&models.CloudSpec{
				DatacenterName: "eu-west-1",
			},
			"eu-west-1",
		},
		{
			[]interface{}{},
			nil,
			"eu-west-1",
		},
	}

	for _, tc := range cases {
		output := expandClusterCloudSpec(tc.Input, tc.DCName)
		if diff := cmp.Diff(tc.ExpectedOutput, output); diff != "" {
			t.Fatalf("Unexpected output from expander: mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestExpandBringYourOwnCloud(t *testing.T) {
	cases := []struct {
		Input          []interface{}
		ExpectedOutput models.BringYourOwnCloudSpec
	}{
		{
			[]interface{}{
				map[string]interface{}{},
			},
			map[string]interface{}{},
		},
		{
			[]interface{}{},
			nil,
		},
	}

	for _, tc := range cases {
		output := expandBringYourOwnCloudSpec(tc.Input)
		if diff := cmp.Diff(tc.ExpectedOutput, output); diff != "" {
			t.Fatalf("Unexpected output from expander: mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestExpandAWSCloudSpec(t *testing.T) {
	cases := []struct {
		Input          []interface{}
		ExpectedOutput *models.AWSCloudSpec
	}{
		{
			[]interface{}{
				map[string]interface{}{
					"access_key_id":         "AKIAIOSFODNN7EXAMPLE",
					"role_arn":              "default",
					"instance_profile_name": "default",
					"route_table_id":        "rtb-09ba434c1bEXAMPLE",
					"secret_access_key":     "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
					"security_group_id":     "sg-51530134",
					"vpc_id":                "e5e4b2ef2fe",
				},
			},
			&models.AWSCloudSpec{
				AccessKeyID:         "AKIAIOSFODNN7EXAMPLE",
				ControlPlaneRoleARN: "default",
				InstanceProfileName: "default",
				RouteTableID:        "rtb-09ba434c1bEXAMPLE",
				SecretAccessKey:     "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
				SecurityGroupID:     "sg-51530134",
				VPCID:               "e5e4b2ef2fe",
			},
		},
		{
			[]interface{}{
				map[string]interface{}{},
			},
			&models.AWSCloudSpec{},
		},
		{
			[]interface{}{},
			nil,
		},
	}

	for _, tc := range cases {
		output := expandAWSCloudSpec(tc.Input)
		if diff := cmp.Diff(tc.ExpectedOutput, output); diff != "" {
			t.Fatalf("Unexpected output from expander: mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestExpandOpenstackCloudSpec(t *testing.T) {
	cases := []struct {
		Input          []interface{}
		ExpectedOutput *models.OpenstackCloudSpec
	}{
		{
			[]interface{}{
				map[string]interface{}{
					"tenant":           "Tenant",
					"floating_ip_pool": "FloatingIPPool",
					"username":         "Username",
					"password":         "Password",
					"network":          "Network",
					"subnet_id":        "SubnetID",
					"router_id":        "RouterID",
					"security_groups":  "SecurityGroups",
				},
			},
			&models.OpenstackCloudSpec{
				Domain:         "Default",
				FloatingIPPool: "FloatingIPPool",
				Password:       "Password",
				Tenant:         "Tenant",
				Username:       "Username",
				Network:        "Network",
				SubnetID:       "SubnetID",
				RouterID:       "RouterID",
				SecurityGroups: "SecurityGroups",
			},
		},
		{
			[]interface{}{
				map[string]interface{}{},
			},
			&models.OpenstackCloudSpec{
				Domain: "Default",
			},
		},
		{
			[]interface{}{},
			nil,
		},
	}

	for _, tc := range cases {
		output := expandOpenstackCloudSpec(tc.Input)
		if diff := cmp.Diff(tc.ExpectedOutput, output); diff != "" {
			t.Fatalf("Unexpected output from expander: mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestExpandAzureCloudSpec(t *testing.T) {
	cases := []struct {
		Input          []interface{}
		ExpectedOutput *models.AzureCloudSpec
	}{
		{

			[]interface{}{
				map[string]interface{}{
					"client_id":       "ClientID",
					"client_secret":   "ClientSecret",
					"tenant_id":       "TenantID",
					"subscription_id": "SubscriptionID",
					"resource_group":  "ResourceGroup",
					"route_table":     "RouteTableName",
					"security_group":  "SecurityGroup",
					"subnet":          "SubnetName",
					"vnet":            "VNetName",
				},
			},
			&models.AzureCloudSpec{
				ClientID:       "ClientID",
				ClientSecret:   "ClientSecret",
				SubscriptionID: "SubscriptionID",
				TenantID:       "TenantID",
				ResourceGroup:  "ResourceGroup",
				RouteTableName: "RouteTableName",
				SecurityGroup:  "SecurityGroup",
				SubnetName:     "SubnetName",
				VNetName:       "VNetName",
			},
		},
		{
			[]interface{}{
				map[string]interface{}{},
			},
			&models.AzureCloudSpec{},
		},
		{
			[]interface{}{},
			nil,
		},
	}

	for _, tc := range cases {
		output := expandAzureCloudSpec(tc.Input)
		if diff := cmp.Diff(tc.ExpectedOutput, output); diff != "" {
			t.Fatalf("Unexpected output from expander: mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestExpandOPAIntegration(t *testing.T) {
	cases := []struct {
		Input          []interface{}
		ExpectedOutput *models.OPAIntegrationSettings
	}{
		{
			[]interface{}{
				map[string]interface{}{
					"enabled":                 true,
					"webhook_timeout_seconds": 20,
				},
			},
			&models.OPAIntegrationSettings{
				Enabled:               true,
				WebhookTimeoutSeconds: 20,
			},
		},
	}

	for _, tc := range cases {
		output := expandOPAIntegration(tc.Input)
		if diff := cmp.Diff(tc.ExpectedOutput, output); diff != "" {
			t.Fatalf("Unexpected output from expander: mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestExpandMLA(t *testing.T) {
	cases := []struct {
		Input          []interface{}
		ExpectedOutput *models.MLASettings
	}{
		{
			[]interface{}{
				map[string]interface{}{
					"logging_enabled":    true,
					"monitoring_enabled": true,
				},
			},
			&models.MLASettings{
				LoggingEnabled:    true,
				MonitoringEnabled: true,
			},
		},
	}

	for _, tc := range cases {
		output := expandMLA(tc.Input)
		if diff := cmp.Diff(tc.ExpectedOutput, output); diff != "" {
			t.Fatalf("Unexpected output from expander: mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestExpandMachineNetwork(t *testing.T) {
	cases := []struct {
		Input          []interface{}
		ExpectedOutput []*models.MachineNetworkingConfig
	}{
		{
			[]interface{}{
				map[string]interface{}{
					"cidr":    "192.168.0.0/24",
					"gateway": "192.168.1.1",
					"dns_servers": []interface{}{
						"192.200.200.1",
						"192.200.200.201",
					},
				},
			},
			[]*models.MachineNetworkingConfig{
				{
					CIDR:    "192.168.0.0/24",
					Gateway: "192.168.1.1",
					DNSServers: []string{
						"192.200.200.1",
						"192.200.200.201",
					},
				},
			},
		},
		{
			[]interface{}{
				map[string]interface{}{},
			},
			[]*models.MachineNetworkingConfig{{}},
		},
		{
			[]interface{}{},
			nil,
		},
	}

	for _, tc := range cases {
		output := expandMachineNetworks(tc.Input)
		if diff := cmp.Diff(tc.ExpectedOutput, output); diff != "" {
			t.Fatalf("Unexpected output from expander: mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestExpandAuditLogging(t *testing.T) {
	want := &models.AuditLoggingSettings{
		Enabled: true,
	}
	got := expandAuditLogging(true)
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("want %+v, got %+v", want, got)
	}
}
