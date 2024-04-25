package documents

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/api/rest/routes"
)

type LogEntry struct {
	// TODO: Work out how to represent this in OpenAPI
	*models.LogEntry
}

type LogDescriptor struct {
	baseResourceDocument

	ID        models.LogDescriptorID `json:"id"`
	CreatedAt models.Time            `json:"created_at"`
	UpdatedAt models.Time            `json:"updated_at"`
	ETag      models.ETag            `json:"etag" hash:"ignore"`

	// ParentLogID is the ID of the log descriptor that this one is nested within.
	ParentLogID models.LogDescriptorID `json:"parent_log_id"`
	// ResourceID is the ID of the resource that this log belongs to
	ResourceID models.ResourceID `json:"resource_id"`
	// Sealed is set to true when the log is completed and has become immutable
	Sealed bool `json:"sealed"`
	// SizeBytes is calculated and set at the time the log is sealed
	SizeBytes int64 `json:"size_bytes"`

	DataURL string `json:"data_url"`
}

func MakeLog(rctx routes.RequestContext, log *models.LogDescriptor) *LogDescriptor {
	return &LogDescriptor{
		baseResourceDocument: baseResourceDocument{
			URL: routes.MakeLogLink(rctx, log.ID),
		},

		ID:          log.ID,
		ParentLogID: log.ParentLogID,
		CreatedAt:   log.CreatedAt,
		UpdatedAt:   log.UpdatedAt,
		ETag:        log.ETag,

		ResourceID: log.ResourceID,
		Sealed:     log.Sealed,
		SizeBytes:  log.SizeBytes,

		DataURL: routes.MakeLogDataLink(rctx, log.ID),
	}
}

func MakeLogs(rctx routes.RequestContext, logs []*models.LogDescriptor) []*LogDescriptor {
	var docs []*LogDescriptor
	for _, model := range logs {
		docs = append(docs, MakeLog(rctx, model))
	}
	return docs
}

func (m *LogDescriptor) GetID() models.ResourceID {
	return m.ID.ResourceID
}

func (m *LogDescriptor) GetKind() models.ResourceKind {
	return models.LogDescriptorResourceKind
}

func (m *LogDescriptor) GetCreatedAt() models.Time {
	return m.CreatedAt
}

type LogSearchRequest struct {
	// TODO: Include all model fields directly in this object
	*models.LogSearch
}

func NewLogSearchRequest() *LogSearchRequest {
	return &LogSearchRequest{LogSearch: models.NewLogSearch()}
}

func (d *LogSearchRequest) Bind(r *http.Request) error {
	return d.Validate()
}

func (d *LogSearchRequest) GetQuery() url.Values {
	values := make(url.Values)
	if d.StartSeqNo != nil && *d.StartSeqNo > 0 {
		values.Set("start", url.QueryEscape(strconv.Itoa(*d.StartSeqNo)))
	}
	if d.Plaintext != nil {
		values.Set("plaintext", url.QueryEscape(strconv.FormatBool(*d.Plaintext)))
	}
	if d.Expand != nil {
		values.Set("expand", url.QueryEscape(strconv.FormatBool(*d.Expand)))
	}
	return values
}

func (d *LogSearchRequest) FromQuery(values url.Values) error {
	vals, ok := values["start"]
	if ok && len(vals) > 0 {
		val, err := url.QueryUnescape(vals[0])
		if err != nil {
			return fmt.Errorf("error unescaping start: %w", err)
		}
		parsed, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return fmt.Errorf("error parsing start: %w", err)
		}
		startSeqNo := int(parsed)
		d.StartSeqNo = &startSeqNo
	}
	vals, ok = values["plaintext"]
	if ok && len(vals) > 0 {
		val, err := url.QueryUnescape(vals[0])
		if err != nil {
			return fmt.Errorf("error unescaping plaintext: %w", err)
		}
		plaintext, err := strconv.ParseBool(val)
		if err != nil {
			return fmt.Errorf("error parsing plaintext: %w", err)
		}
		d.Plaintext = &plaintext
	}
	vals, ok = values["expand"]
	if ok && len(vals) > 0 {
		val, err := url.QueryUnescape(vals[0])
		if err != nil {
			return fmt.Errorf("error unescaping expand: %w", err)
		}
		expand, err := strconv.ParseBool(val)
		if err != nil {
			return fmt.Errorf("error parsing expand: %w", err)
		}
		d.Expand = &expand
	}
	return d.Validate()
}
