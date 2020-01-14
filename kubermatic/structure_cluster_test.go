package kubermatic

import (
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
				Version:         "1.15.6",
				MachineNetworks: nil,
				AuditLogging:    &models.AuditLoggingSettings{},
				Cloud: &models.CloudSpec{
					DatacenterName: "eu-west-1",
					Bringyourown:   map[string]interface{}{},
				},
			},
			[]interface{}{
				map[string]interface{}{
					"version": "1.15.6",
					"audit_logging": []interface{}{
						map[string]interface{}{
							"enabled": false,
						},
					},
					"cloud": []interface{}{
						map[string]interface{}{
							"dc":           "eu-west-1",
							"bringyourown": []interface{}{map[string]interface{}{}},
						},
					},
				},
			},
		},
		{
			&models.ClusterSpec{},
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
		output := flattenClusterSpec(tc.Input)
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
				DatacenterName: "eu-west-1",
				Aws:            &models.AWSCloudSpec{},
			},
			[]interface{}{
				map[string]interface{}{
					"dc": "eu-west-1",
					"aws": []interface{}{
						map[string]interface{}{},
					},
				},
			},
		},
		{
			&models.CloudSpec{
				DatacenterName: "eu-west-1",
				Bringyourown:   map[string]interface{}{},
			},
			[]interface{}{
				map[string]interface{}{
					"dc": "eu-west-1",
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
		output := flattenClusterCloudSpec(tc.Input)
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
	}{
		{
			[]interface{}{
				map[string]interface{}{
					"version":          "1.15.6",
					"machine_networks": []interface{}{},
					"audit_logging": []interface{}{
						map[string]interface{}{},
					},
					"cloud": []interface{}{
						map[string]interface{}{
							"dc": "eu-west-1",
							"bringyourown": []interface{}{
								map[string]interface{}{},
							},
						},
					},
				},
			},
			&models.ClusterSpec{
				Version:         "1.15.6",
				MachineNetworks: nil,
				AuditLogging:    &models.AuditLoggingSettings{},
				Cloud: &models.CloudSpec{
					DatacenterName: "eu-west-1",
					Bringyourown:   map[string]interface{}{},
				},
			},
		},
		{
			[]interface{}{
				map[string]interface{}{},
			},
			&models.ClusterSpec{},
		},
		{
			[]interface{}{},
			nil,
		},
	}

	for _, tc := range cases {
		output := expandClusterSpec(tc.Input)
		if diff := cmp.Diff(tc.ExpectedOutput, output); diff != "" {
			t.Fatalf("Unexpected output from expander: mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestExpandClusterCloudSpec(t *testing.T) {
	cases := []struct {
		Input          []interface{}
		ExpectedOutput *models.CloudSpec
	}{
		{
			[]interface{}{
				map[string]interface{}{
					"dc": "eu-west-1",
					"bringyourown": []interface{}{
						map[string]interface{}{},
					},
				},
			},
			&models.CloudSpec{
				DatacenterName: "eu-west-1",
				Bringyourown:   map[string]interface{}{},
			},
		},
		{
			[]interface{}{
				map[string]interface{}{
					"dc": "eu-west-1",
					"aws": []interface{}{
						map[string]interface{}{},
					},
				},
			},
			&models.CloudSpec{
				DatacenterName: "eu-west-1",
				Aws:            &models.AWSCloudSpec{},
			},
		},
		{
			[]interface{}{
				map[string]interface{}{},
			},
			&models.CloudSpec{},
		},
		{
			[]interface{}{},
			nil,
		},
	}

	for _, tc := range cases {
		output := expandClusterCloudSpec(tc.Input)
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
	cases := []struct {
		Input          []interface{}
		ExpectedOutput *models.AuditLoggingSettings
	}{
		{
			[]interface{}{
				map[string]interface{}{
					"enabled": true,
				},
			},
			&models.AuditLoggingSettings{
				Enabled: true,
			},
		},
		{
			[]interface{}{
				map[string]interface{}{},
			},
			&models.AuditLoggingSettings{
				Enabled: false,
			},
		},
		{
			[]interface{}{},
			nil,
		},
	}

	for _, tc := range cases {
		output := expandAuditLogging(tc.Input)
		if diff := cmp.Diff(tc.ExpectedOutput, output); diff != "" {
			t.Fatalf("Unexpected output from expander: mismatch (-want +got):\n%s", diff)
		}
	}
}
