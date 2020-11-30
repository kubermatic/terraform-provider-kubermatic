package metakube

import "github.com/syseleven/terraform-provider-metakube/go-metakube/models"

func errorMessage(e *models.ErrorResponse) string {
	if e != nil && e.Error != nil && e.Error.Message != nil {
		return *e.Error.Message
	}
	return ""
}
