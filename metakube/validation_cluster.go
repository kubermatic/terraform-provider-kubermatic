package metakube

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/syseleven/terraform-provider-metakube/go-metakube/client/openstack"
	"github.com/syseleven/terraform-provider-metakube/go-metakube/client/versions"
	"github.com/syseleven/terraform-provider-metakube/go-metakube/models"
)

type openstackValidationData struct {
	dcName         *string
	credentials    *string
	domain         *string
	username       *string
	password       *string
	tenant         *string
	floatingIPPool *string
	securityGroup  *string
	network        *string
	subnetID       *string
}

type generalOpenstackReqParams interface {
	SetDatacenterName(*string)
	SetCredential(*string)
	SetDomain(*string)
	SetUsername(*string)
	SetPassword(*string)
	SetTenant(*string)
}

func (data *openstackValidationData) setParams(p generalOpenstackReqParams) {
	p.SetDatacenterName(data.dcName)
	p.SetCredential(data.credentials)
	p.SetDomain(data.domain)
	p.SetUsername(data.username)
	p.SetPassword(data.password)
	p.SetTenant(data.tenant)
}

func newOpenstackValidationData(d *schema.ResourceDiff) openstackValidationData {
	return openstackValidationData{
		dcName:         toStrPtrOrNil(d.Get("dc_name")),
		credentials:    toStrPtrOrNil(d.Get("credential")),
		domain:         strToPtr("Default"),
		username:       toStrPtrOrNil(d.Get("spec.0.cloud.0.openstack.0.username")),
		password:       toStrPtrOrNil(d.Get("spec.0.cloud.0.openstack.0.password")),
		tenant:         toStrPtrOrNil(d.Get("spec.0.cloud.0.openstack.0.tenant")),
		floatingIPPool: toStrPtrOrNil(d.Get("spec.0.cloud.0.openstack.0.floatingIPPool")),
		securityGroup:  toStrPtrOrNil(d.Get("spec.0.cloud.0.openstack.0.securityGroup")),
		network:        toStrPtrOrNil(d.Get("spec.0.cloud.0.openstack.0.network")),
		subnetID:       toStrPtrOrNil(d.Get("spec.0.cloud.0.openstack.0.subnetID")),
	}
}

func toStrPtrOrNil(v interface{}) *string {
	if v == nil {
		return nil
	}
	return strToPtr(v.(string))
}

func validateVersionExists() schema.CustomizeDiffFunc {
	return func(d *schema.ResourceDiff, meta interface{}) error {
		k := meta.(*metakubeProviderMeta)
		version := d.Get("spec.0.version").(string)
		p := versions.NewGetMasterVersionsParams()
		r, err := k.client.Versions.GetMasterVersions(p, k.auth)
		if err != nil {
			if e, ok := err.(*versions.GetMasterVersionsDefault); ok && errorMessage(e.Payload) != "" {
				return fmt.Errorf("get cluster upgrades: %s", errorMessage(e.Payload))
			}
			return err
		}

		for _, v := range r.Payload {
			if s, ok := v.Version.(string); ok && s == version {
				return nil
			}
		}

		return fmt.Errorf("unknown version %s", version)
	}
}

func validateOnlyOneCloudProviderSpecified() schema.CustomizeDiffFunc {
	return func(d *schema.ResourceDiff, meta interface{}) error {
		var existingProviders []string
		counter := 0
		for _, provider := range supportedProviders {
			if _, ok := d.GetOk(fmt.Sprintf("spec.0.cloud.0.%s.0", provider)); ok {
				existingProviders = append(existingProviders, provider)
				counter++
			}
		}
		if counter > 1 {
			return fmt.Errorf("only one cloud provider must be specified: %v", existingProviders)
		}
		return nil
	}
}

func validateOpenstackNetworkExistsIfSet(field string, external bool) schema.CustomizeDiffFunc {
	return func(d *schema.ResourceDiff, meta interface{}) error {
		value, ok := d.GetOk(field)
		if !ok {
			return nil
		}

		data := newOpenstackValidationData(d)
		k := meta.(*metakubeProviderMeta)
		_, err := getNetwork(k, data, value.(string), external)
		return err
	}
}

func validateOpenstackSubnetWithIDExistsIfSet() schema.CustomizeDiffFunc {
	return func(d *schema.ResourceDiff, meta interface{}) error {
		data := newOpenstackValidationData(d)
		if data.network == nil || data.subnetID == nil {
			return nil
		}
		k := meta.(*metakubeProviderMeta)
		network, err := getNetwork(k, data, *data.network, true)
		if err != nil {
			return err
		}

		_, err = getSubnet(k, data, network.ID)
		return err
	}
}

func getNetwork(k *metakubeProviderMeta, data openstackValidationData, name string, external bool) (*models.OpenstackNetwork, error) {
	p := openstack.NewListOpenstackNetworksParams()
	data.setParams(p)
	res, err := k.client.Openstack.ListOpenstackNetworks(p, k.auth)
	if err != nil {
		return nil, fmt.Errorf("find network instance %v", getErrorResponse(err))
	}
	ret := findNetwork(res.Payload, name, external)
	if ret == nil {
		return nil, fmt.Errorf("network `%s` not found", name)
	}
	return ret, nil
}

func findNetwork(list []*models.OpenstackNetwork, network string, external bool) *models.OpenstackNetwork {
	for _, item := range list {
		if item.Name == network && item.External == external {
			return item
		}
	}
	return nil
}

func getSubnet(k *metakubeProviderMeta, data openstackValidationData, networkID string) (bool, error) {
	p := openstack.NewListOpenstackSubnetsParams()
	data.setParams(p)
	p.SetNetworkID(&networkID)
	res, err := k.client.Openstack.ListOpenstackSubnets(p, k.auth)
	if err != nil {
		return false, fmt.Errorf("list network subnets: %v", getErrorResponse(err))
	}
	return findSubnet(res.Payload, *data.subnetID) != nil, nil
}

func findSubnet(list []*models.OpenstackSubnet, id string) *models.OpenstackSubnet {
	for _, item := range list {
		if item.ID == id {
			return item
		}
	}
	return nil
}
