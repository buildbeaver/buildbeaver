package dto

import "github.com/buildbeaver/buildbeaver/common/models"

type UpdateRepoEnabled struct {
	Enabled bool
	ETag    models.ETag
}
