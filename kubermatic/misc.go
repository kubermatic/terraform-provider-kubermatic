package kubermatic

import "github.com/kubermatic/go-kubermatic/models"

func errorMessage(e *models.ErrorResponse) string {
	if e != nil && e.Error != nil && e.Error.Message != nil {
		return *e.Error.Message
	}
	return ""
}
