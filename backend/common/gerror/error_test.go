package gerror

import (
	"errors"
	"fmt"
	"testing"

	"github.com/hashicorp/go-multierror"

	"github.com/stretchr/testify/require"
)

func TestError(t *testing.T) {
	err := NewErrAlreadyExists("foo already exists")
	err = err.Wrap(fmt.Errorf("i'm a scary internal error"))
	require.Equal(t, "foo already exists: i'm a scary internal error", err.Error())
	require.Equal(t, "foo already exists", err.Message())

	err = err.EDetail("foo", "bar")
	require.Equal(t, "foo already exists [foo=bar]: i'm a scary internal error", err.Error())
	require.Equal(t, "foo already exists", err.Message())

	err = err.Wrap(NewErrNotFound("foo does not exist").EDetail("bar", "baz").Wrap(fmt.Errorf("i'm a scary internal error")))
	require.Equal(t, "foo already exists [foo=bar]: foo does not exist [bar=baz]: i'm a scary internal error", err.Error())
	require.Equal(t, "foo already exists", err.Message())
}

func TestMultiError(t *testing.T) {
	// Compose a multierror with our tested error in the middle
	var results *multierror.Error

	results = multierror.Append(results, fmt.Errorf("error 1: %w", errors.New("1")))
	results = multierror.Append(results, NewErrArtifactUploadFailed("Failed uploading artifact", errors.New("2")))
	results = multierror.Append(results, fmt.Errorf("error 3: %w", errors.New("3")))

	// Assert that our Is chaining returns an error in the middle of the chain
	err := results.ErrorOrNil()
	require.True(t, IsArtifactUploadFailed(err))

	// Wrap up the above error with another multierror
	var outerResults *multierror.Error
	outerResults = multierror.Append(err, fmt.Errorf("outer error 1: %w", errors.New("11")))

	// And assert our Is chaining returns the error we are after.
	outerErr := outerResults.ErrorOrNil()
	require.True(t, IsArtifactUploadFailed(outerErr))
}
