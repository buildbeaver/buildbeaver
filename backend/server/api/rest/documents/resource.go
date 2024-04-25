package documents

import "github.com/buildbeaver/buildbeaver/common/models"

type ResourceDocument interface {
	// GetLink returns a link that can be used to fetch the resource from the server.
	GetLink() string
	// GetID returns the globally unique ResourceID of the resource.
	GetID() models.ResourceID
	// GetKind returns the unique name/type of the resource e.g. "build" or "repo".
	GetKind() models.ResourceKind
	// GetCreatedAt returns the Time at which this resource was created.
	GetCreatedAt() models.Time
}

type baseResourceDocument struct {
	URL string `json:"url"`
}

func (d *baseResourceDocument) GetLink() string {
	return d.URL
}
