package routes

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/store"
)

var resourceNameParamToKindMap = map[string]models.ResourceKind{
	"legal_entity_name": models.LegalEntityResourceKind,
	"repo_name":         models.RepoResourceKind,
	"runner_name":       models.RunnerResourceKind,
	"build_name":        models.BuildResourceKind,
	"secret_name":       models.SecretResourceKind,
	"job_name":          models.JobResourceKind,
	"step_name":         models.StepResourceKind,
	"artifact_name":     models.ArtifactResourceKind,
}

var resourceIDParamMap = map[string]struct{}{
	"legal_entity_id":   {},
	"repo_id":           {},
	"runner_id":         {},
	"build_id":          {},
	"secret_id":         {},
	"job_id":            {},
	"step_id":           {},
	"artifact_id":       {},
	"log_descriptor_id": {},
}

type ResourceLinker struct {
	linkStore store.ResourceLinkStore
	logger.Log
}

func NewResourceLinker(
	linkStore store.ResourceLinkStore,
	logFactory logger.LogFactory) *ResourceLinker {
	return &ResourceLinker{
		linkStore: linkStore,
		Log:       logFactory("ResourceLinker"),
	}
}

func (a *ResourceLinker) GetLeafResourceID(r *http.Request) (models.ResourceID, error) {
	var (
		rctx   = chi.RouteContext(r.Context())
		params = rctx.URLParams
		link   []models.ResourceLinkFragmentID
	)
	// Do a look ahead to determine if this is an id or name based route.
	var nameBased bool
	for i := len(params.Keys) - 1; i >= 0; i-- {
		key := params.Keys[i]
		_, ok := resourceIDParamMap[key]
		if ok {
			// ID based route - nothing more to do
			value := params.Values[i]
			// This is required to support clients that escape colon's in IDs
			escaped, err := url.PathUnescape(value)
			if err != nil {
				return models.ResourceID{}, fmt.Errorf("error unescaping path: %w", err)
			}
			return models.ParseResourceID(escaped)
		}
		_, ok = resourceNameParamToKindMap[key]
		if ok {
			// Name based route - we'll need to parse the full route below
			nameBased = true
			break
		}
	}
	if !nameBased {
		return models.ResourceID{}, fmt.Errorf("route does not contain a resource")
	}
	for i, key := range params.Keys {
		kind, ok := resourceNameParamToKindMap[key]
		if ok {
			link = append(link, models.ResourceLinkFragmentID{
				Name: models.ResourceName(params.Values[i]),
				Kind: kind,
			})
		}
	}
	leaf, err := a.linkStore.Resolve(r.Context(), nil, link)
	if err != nil {
		return models.ResourceID{}, fmt.Errorf("error resolving resource link %q: %w", link, err)
	}
	return leaf.ID, nil
}
