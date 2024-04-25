package documents

import "github.com/buildbeaver/buildbeaver/common/gerror"

// ErrorDocument is a standard error representation returned by the API
type ErrorDocument struct {
	Code           gerror.Code                      `json:"code"`
	HTTPStatusCode int                              `json:"http_status_code"`
	Message        string                           `json:"message"`
	Details        map[gerror.DetailKey]interface{} `json:"details"`
}
