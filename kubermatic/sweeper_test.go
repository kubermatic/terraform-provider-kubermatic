package kubermatic

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"go.uber.org/zap"
)

func TestMain(m *testing.M) {
	resource.TestMain(m)
}

func sharedConfigForRegion(_ string) (*kubermaticProviderMeta, error) {
	host := os.Getenv("KUBERMATIC_HOST")
	client, err := newClient(host)
	if err != nil {
		return nil, fmt.Errorf("create client %v", err)
	}
	token := os.Getenv("KUBERMATIC_TOKEN")
	auth, err := newAuth(token, "")
	if err != nil {
		return nil, fmt.Errorf("auth api %v", err)
	}
	log := zap.NewNop().Sugar()
	return &kubermaticProviderMeta{
		client,
		auth,
		log,
	}, nil
}
