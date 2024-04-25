package models_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/buildbeaver/buildbeaver/common/models"
)

func Test_buildID(t *testing.T) {
	id := models.NewBuildID()
	json, err := id.MarshalJSON()
	require.Nil(t, err)
	id2 := models.BuildID{}
	err = id2.UnmarshalJSON(json)
	require.Nil(t, err)
	require.Equal(t, id, id2)
}
