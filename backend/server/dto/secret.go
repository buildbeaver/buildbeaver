package dto

import "github.com/buildbeaver/buildbeaver/common/models"

type UpdateSecretPlaintext struct {
	KeyPlaintext   *string
	ValuePlaintext *string
	ETag           models.ETag
}
