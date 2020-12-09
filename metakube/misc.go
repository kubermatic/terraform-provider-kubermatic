package metakube

import (
	"fmt"
	"github.com/syseleven/terraform-provider-metakube/go-metakube/models"
)

func errorMessage(e *models.ErrorResponse) string {
	if e != nil && e.Error != nil && e.Error.Message != nil {
		if len(e.Error.Additional) > 0 {
			return fmt.Sprintf("%s %v", *e.Error.Message, e.Error.Additional)
		}
		return *e.Error.Message
	}
	return ""
}
