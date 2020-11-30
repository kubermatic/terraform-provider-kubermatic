package metakube

import (
	"fmt"
	"os"

	"go.uber.org/zap"
)

func sharedConfigForRegion(_ string) (*metakubeProviderMeta, error) {
	host := os.Getenv("METAKUBE_HOST")
	client, err := newClient(host)
	if err != nil {
		return nil, fmt.Errorf("create client %v", err)
	}
	token := os.Getenv("METAKUBE_TOKEN")
	auth, err := newAuth(token, "")
	if err != nil {
		return nil, fmt.Errorf("auth api %v", err)
	}
	log := zap.NewNop().Sugar()
	return &metakubeProviderMeta{
		client,
		auth,
		log,
	}, nil
}
