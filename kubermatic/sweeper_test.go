package kubermatic

import (
	"fmt"
	"os"

	"go.uber.org/zap"
)

func sharedConfigForRegion(_ string) (*kubermaticProviderMeta, error) {
	host := os.Getenv("KUBERMATIC_HOST")
	client, err := newClient(host)
	if err != nil {
		return nil, fmt.Errorf("create client %w", err)
	}
	token := os.Getenv("KUBERMATIC_TOKEN")
	auth, err := newAuth(token, "")
	if err != nil {
		return nil, fmt.Errorf("auth api %w", err)
	}
	log := zap.NewNop().Sugar()
	return &kubermaticProviderMeta{
		client,
		auth,
		log,
	}, nil
}
