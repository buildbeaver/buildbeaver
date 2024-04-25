package models

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResourceNameTruncation(t *testing.T) {
	tooLong := "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz"
	normalized := NormalizeResourceName(tooLong)
	require.Equal(t, "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijkl", normalized[prefixLen:])
}
