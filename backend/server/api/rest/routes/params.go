package routes

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/pkg/errors"
)

// IntParam extracts an int from the url parameters on the supplied request.
func IntParam(r *http.Request, key string) (int64, error) {
	idStr := chi.URLParam(r, key)
	if idStr == "" {
		return 0, fmt.Errorf("error %q param does not exist", key)
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return -1, errors.Wrap(err, "error parsing int param")
	}
	return id, nil
}
