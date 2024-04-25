package search

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/buildbeaver/buildbeaver/common/models"
)

func TestQueryTokenStream(t *testing.T) {
	query := `deb foo in:name size:>=1048576 kind:artifact sort:size-asc`
	stream := &queryTokenStream{input: &queryInputStream{query: query}}

	token := stream.Next()
	require.Equal(t, Term("deb"), token)

	token = stream.Next()
	require.Equal(t, Term("foo"), token)

	token = stream.Next()
	require.Equal(t, FieldName("name"), token)

	token = stream.Next()
	require.Equal(t, NewFieldFilter("size", GreaterThanOrEqualTo, "1048576"), token)

	token = stream.Next()
	require.Equal(t, models.ArtifactResourceKind, token)

	token = stream.Next()
	require.Equal(t, NewSortField("size", Ascending), token)

	token = stream.Next()
	require.Nil(t, token)
}

func TestQueryTokenStreamQuotes(t *testing.T) {
	query := `"deb foo" in:name`
	stream := &queryTokenStream{input: &queryInputStream{query: query}}

	token := stream.Next()
	require.Equal(t, Term("deb foo"), token)

	token = stream.Next()
	require.Equal(t, FieldName("name"), token)

	token = stream.Next()
	require.Nil(t, token)
}

func TestQueryTokenStreamEscape(t *testing.T) {
	query := `"deb \"foo\"" in:name`
	stream := &queryTokenStream{input: &queryInputStream{query: query}}

	token := stream.Next()
	require.Equal(t, Term(`deb "foo"`), token)

	token = stream.Next()
	require.Equal(t, FieldName("name"), token)

	token = stream.Next()
	require.Nil(t, token)
}

func TestQueryTokenStreamTermPlacement(t *testing.T) {
	query := `deb in:name foo size:>=1 bar in:description baz`
	stream := &queryTokenStream{input: &queryInputStream{query: query}}

	token := stream.Next()
	require.Equal(t, Term("deb"), token)

	token = stream.Next()
	require.Equal(t, FieldName("name"), token)

	token = stream.Next()
	require.Equal(t, Term("foo"), token)

	token = stream.Next()
	require.Equal(t, NewFieldFilter("size", GreaterThanOrEqualTo, "1"), token)

	token = stream.Next()
	require.Equal(t, Term("bar"), token)

	token = stream.Next()
	require.Equal(t, FieldName("description"), token)

	token = stream.Next()
	require.Equal(t, Term("baz"), token)

	token = stream.Next()
	require.Nil(t, token)
}

func TestParseQuery(t *testing.T) {
	query := ParseQuery(`deb foo in:name size:>=1048576 kind:artifact sort:size-asc`)
	require.NotNil(t, query.Term)
	require.Equal(t, Term("deb foo"), *query.Term)
	require.Equal(t, true, query.IsInFieldSet("name"))
	require.Equal(t, NewFieldFilter("size", GreaterThanOrEqualTo, "1048576"), query.GetFilter("size"))
	require.NotNil(t, query.Kind)
	require.Equal(t, models.ArtifactResourceKind, *query.Kind)
	require.Equal(t, NewSortField("size", Ascending), query.Sort)
}

func TestOperators(t *testing.T) {
	// All operators
	for _, op := range operatorSet {
		query := ParseQuery(fmt.Sprintf("size:%s10", op))
		require.Equal(t, NewFieldFilter("size", op, "10"), query.GetFilter("size"))
	}
	// Default operator
	query := ParseQuery("size:10")
	require.Equal(t, NewFieldFilter("size", Equal, "10"), query.GetFilter("size"))
}

func TestSortDirection(t *testing.T) {
	// All directions
	for _, direction := range []SortDirection{Ascending, Descending} {
		query := ParseQuery(fmt.Sprintf("sort:size-%s", direction))
		require.Equal(t, NewSortField("size", direction), query.Sort)
	}
	// Default direction
	query := ParseQuery("sort:size")
	require.Equal(t, NewSortField("size", Ascending), query.Sort)
}
