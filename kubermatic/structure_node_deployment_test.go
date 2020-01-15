package kubermatic

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kubermatic/go-kubermatic/models"
)

func TestFlattenNodeDeploymentSpec(t *testing.T) {
	cases := []struct {
		Input          *models.NodeDeploymentSpec
		ExpectedOutput []interface{}
	}{
		{
			&models.NodeDeploymentSpec{
				Replicas: int32ToPtr(1),
				Template: &models.NodeSpec{},
			},
			[]interface{}{
				map[string]interface{}{
					"replicas": int32(1),
					"template": []interface{}{map[string]interface{}{}},
				},
			},
		},
		{
			&models.NodeDeploymentSpec{},
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
		output := flattenNodeDeploymentSpec(tc.Input)
		if diff := cmp.Diff(tc.ExpectedOutput, output); diff != "" {
			t.Fatalf("Unexpected output from expander: mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestFlattenNodeSpec(t *testing.T) {
	cases := []struct {
		Input          *models.NodeSpec
		ExpectedOutput []interface{}
	}{
		{
			&models.NodeSpec{
				OperatingSystem: &models.OperatingSystemSpec{
					Ubuntu: &models.UbuntuSpec{},
				},
				Taints: []*models.TaintSpec{
					{
						Key:    "key1",
						Value:  "value1",
						Effect: "NoSchedule",
					},
					{
						Key:    "key2",
						Value:  "value2",
						Effect: "NoSchedule",
					},
				},
				Cloud: &models.NodeCloudSpec{
					Aws: &models.AWSNodeSpec{},
				},
				Labels: map[string]string{
					"foo": "bar",
				},
				Versions: &models.NodeVersionInfo{
					Kubelet: "1.15.6",
				},
			},
			[]interface{}{
				map[string]interface{}{
					"operating_system": []interface{}{
						map[string]interface{}{
							"ubuntu": []interface{}{
								map[string]interface{}{
									"dist_upgrade_on_boot": false,
								},
							},
						},
					},
					"taints": []interface{}{
						map[string]interface{}{
							"key":    "key1",
							"value":  "value1",
							"effect": "NoSchedule",
						},
						map[string]interface{}{
							"key":    "key2",
							"value":  "value2",
							"effect": "NoSchedule",
						},
					},
					"cloud": []interface{}{
						map[string]interface{}{
							"aws": []interface{}{
								map[string]interface{}{
									"assign_public_ip": false,
								},
							},
						},
					},
					"labels": map[string]string{
						"foo": "bar",
					},
					"versions": []interface{}{
						map[string]interface{}{
							"kubelet": "1.15.6",
						},
					},
				},
			},
		},
		{
			&models.NodeSpec{},
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
		output := flattenNodeSpec(tc.Input)
		if diff := cmp.Diff(tc.ExpectedOutput, output); diff != "" {
			t.Fatalf("Unexpected output from expander: mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestFlattenOperatingSystem(t *testing.T) {
	cases := []struct {
		Input          *models.OperatingSystemSpec
		ExpectedOutput []interface{}
	}{
		{
			&models.OperatingSystemSpec{
				Centos: &models.CentOSSpec{
					DistUpgradeOnBoot: true,
				},
			},
			[]interface{}{
				map[string]interface{}{
					"centos": []interface{}{
						map[string]interface{}{
							"dist_upgrade_on_boot": true,
						},
					},
				},
			},
		},
		{
			&models.OperatingSystemSpec{
				Ubuntu: &models.UbuntuSpec{
					DistUpgradeOnBoot: true,
				},
			},
			[]interface{}{
				map[string]interface{}{
					"ubuntu": []interface{}{
						map[string]interface{}{
							"dist_upgrade_on_boot": true,
						},
					},
				},
			},
		},
		{
			&models.OperatingSystemSpec{
				ContainerLinux: &models.ContainerLinuxSpec{
					DisableAutoUpdate: true,
				},
			},
			[]interface{}{
				map[string]interface{}{
					"container_linux": []interface{}{
						map[string]interface{}{
							"disable_auto_update": true,
						},
					},
				},
			},
		},
		{
			&models.OperatingSystemSpec{},
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
		output := flattenOperatingSystem(tc.Input)
		if diff := cmp.Diff(tc.ExpectedOutput, output); diff != "" {
			t.Fatalf("Unexpected output from expander: mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestFlattenAWSNodeSpec(t *testing.T) {
	cases := []struct {
		Input          *models.AWSNodeSpec
		ExpectedOutput []interface{}
	}{
		{
			&models.AWSNodeSpec{
				AMI:              "ami-5731123e",
				AssignPublicIP:   true,
				AvailabilityZone: "eu-west-1",
				InstanceType:     strToPtr("t3.small"),
				SubnetID:         "subnet-53485",
				Tags: map[string]string{
					"foo": "bar",
				},
				VolumeSize: int64ToPtr(25),
				VolumeType: strToPtr("standard"),
			},
			[]interface{}{
				map[string]interface{}{
					"ami":               "ami-5731123e",
					"assign_public_ip":  true,
					"availability_zone": "eu-west-1",
					"instance_type":     "t3.small",
					"subnet_id":         "subnet-53485",
					"tags": map[string]string{
						"foo": "bar",
					},
					"disk_size":   int64(25),
					"volume_type": "standard",
				},
			},
		},
		{
			&models.AWSNodeSpec{},
			[]interface{}{
				map[string]interface{}{
					"assign_public_ip": false,
				},
			},
		},
		{
			nil,
			[]interface{}{},
		},
	}

	for _, tc := range cases {
		output := flattenAWSNodeSpec(tc.Input)
		if diff := cmp.Diff(tc.ExpectedOutput, output); diff != "" {
			t.Fatalf("Unexpected output from expander: mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestExpandNodeDeploymentSpec(t *testing.T) {
	cases := []struct {
		Input          []interface{}
		ExpectedOutput *models.NodeDeploymentSpec
	}{
		{
			[]interface{}{
				map[string]interface{}{
					"replicas": 1,
					"template": []interface{}{map[string]interface{}{}},
				},
			},
			&models.NodeDeploymentSpec{
				Replicas: int32ToPtr(1),
				Template: &models.NodeSpec{},
			},
		},
		{

			[]interface{}{
				map[string]interface{}{},
			},
			&models.NodeDeploymentSpec{},
		},
		{
			[]interface{}{},
			nil,
		},
	}

	for _, tc := range cases {
		output := expandNodeDeploymentSpec(tc.Input)
		if diff := cmp.Diff(tc.ExpectedOutput, output); diff != "" {
			t.Fatalf("Unexpected output from expander: mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestExpandNodeSpec(t *testing.T) {
	cases := []struct {
		Input          []interface{}
		ExpectedOutput *models.NodeSpec
	}{
		{
			[]interface{}{
				map[string]interface{}{
					"operating_system": []interface{}{
						map[string]interface{}{
							"ubuntu": []interface{}{
								map[string]interface{}{
									"dist_upgrade_on_boot": false,
								},
							},
						},
					},
					"taints": []interface{}{
						map[string]interface{}{
							"key":    "key1",
							"value":  "value1",
							"effect": "NoSchedule",
						},
						map[string]interface{}{
							"key":    "key2",
							"value":  "value2",
							"effect": "NoSchedule",
						},
					},
					"cloud": []interface{}{
						map[string]interface{}{
							"aws": []interface{}{
								map[string]interface{}{
									"assign_public_ip": false,
								},
							},
						},
					},
					"labels": map[string]interface{}{
						"foo": "bar",
					},
					"versions": []interface{}{
						map[string]interface{}{
							"kubelet": "1.15.6",
						},
					},
				},
			},
			&models.NodeSpec{
				OperatingSystem: &models.OperatingSystemSpec{
					Ubuntu: &models.UbuntuSpec{},
				},
				Taints: []*models.TaintSpec{
					{
						Key:    "key1",
						Value:  "value1",
						Effect: "NoSchedule",
					},
					{
						Key:    "key2",
						Value:  "value2",
						Effect: "NoSchedule",
					},
				},
				Cloud: &models.NodeCloudSpec{
					Aws: &models.AWSNodeSpec{},
				},
				Labels: map[string]string{
					"foo": "bar",
				},
				Versions: &models.NodeVersionInfo{
					Kubelet: "1.15.6",
				},
			},
		},
		{

			[]interface{}{
				map[string]interface{}{},
			},
			&models.NodeSpec{},
		},
		{
			[]interface{}{},
			nil,
		},
	}

	for _, tc := range cases {
		output := expandNodeSpec(tc.Input)
		if diff := cmp.Diff(tc.ExpectedOutput, output); diff != "" {
			t.Fatalf("Unexpected output from expander: mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestExpandOperatingSystem(t *testing.T) {
	cases := []struct {
		Input          []interface{}
		ExpectedOutput *models.OperatingSystemSpec
	}{
		{
			[]interface{}{
				map[string]interface{}{
					"centos": []interface{}{
						map[string]interface{}{
							"dist_upgrade_on_boot": true,
						},
					},
				},
			},
			&models.OperatingSystemSpec{
				Centos: &models.CentOSSpec{
					DistUpgradeOnBoot: true,
				},
			},
		},
		{
			[]interface{}{
				map[string]interface{}{
					"ubuntu": []interface{}{
						map[string]interface{}{
							"dist_upgrade_on_boot": true,
						},
					},
				},
			},
			&models.OperatingSystemSpec{
				Ubuntu: &models.UbuntuSpec{
					DistUpgradeOnBoot: true,
				},
			},
		},
		{
			[]interface{}{
				map[string]interface{}{
					"container_linux": []interface{}{
						map[string]interface{}{
							"disable_auto_update": true,
						},
					},
				},
			},
			&models.OperatingSystemSpec{
				ContainerLinux: &models.ContainerLinuxSpec{
					DisableAutoUpdate: true,
				},
			},
		},
		{

			[]interface{}{
				map[string]interface{}{},
			},
			&models.OperatingSystemSpec{},
		},
		{
			[]interface{}{},
			nil,
		},
	}

	for _, tc := range cases {
		output := expandOperatingSystem(tc.Input)
		if diff := cmp.Diff(tc.ExpectedOutput, output); diff != "" {
			t.Fatalf("Unexpected output from expander: mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestExpandAWSNodeSpec(t *testing.T) {
	cases := []struct {
		Input          []interface{}
		ExpectedOutput *models.AWSNodeSpec
	}{
		{
			[]interface{}{
				map[string]interface{}{
					"ami":               "ami-5731123e",
					"assign_public_ip":  true,
					"availability_zone": "eu-west-1",
					"instance_type":     "t3.small",
					"subnet_id":         "subnet-53485",
					"tags": map[string]interface{}{
						"foo": "bar",
					},
					"disk_size":   25,
					"volume_type": "standard",
				},
			},
			&models.AWSNodeSpec{
				AMI:              "ami-5731123e",
				AssignPublicIP:   true,
				AvailabilityZone: "eu-west-1",
				InstanceType:     strToPtr("t3.small"),
				SubnetID:         "subnet-53485",
				Tags: map[string]string{
					"foo": "bar",
				},
				VolumeSize: int64ToPtr(25),
				VolumeType: strToPtr("standard"),
			},
		},
		{

			[]interface{}{
				map[string]interface{}{},
			},
			&models.AWSNodeSpec{},
		},
		{
			[]interface{}{},
			nil,
		},
	}

	for _, tc := range cases {
		output := expandAWSNodeSpec(tc.Input)
		if diff := cmp.Diff(tc.ExpectedOutput, output); diff != "" {
			t.Fatalf("Unexpected output from expander: mismatch (-want +got):\n%s", diff)
		}
	}
}
