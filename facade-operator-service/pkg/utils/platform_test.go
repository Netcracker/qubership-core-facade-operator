package utils

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSemVer(t *testing.T) {
	v1, err := NewSemVer("3.11")
	assert.Nil(t, err)
	assert.Equal(t, 3, v1.Major)
	assert.Equal(t, 11, v1.Minor)
	assert.Equal(t, 0, v1.Patch)

	// same version

	v2, err := NewSemVer("v3.11")
	assert.Nil(t, err)
	assert.True(t, v1 == v2)
	assert.True(t, v1.IsNewerThanOrEqual(v2))

	v2, err = NewSemVer("v3.11.0")
	assert.Nil(t, err)
	assert.True(t, v1 == v2)
	assert.True(t, v1.IsNewerThanOrEqual(v2))

	v2, err = NewSemVer("V3.11")
	assert.Nil(t, err)
	assert.True(t, v1 == v2)
	assert.True(t, v1.IsNewerThanOrEqual(v2))

	// different versions

	v2, err = NewSemVer("v1.11")
	assert.Nil(t, err)
	assert.False(t, v1 == v2)
	assert.True(t, v1.IsNewerThanOrEqual(v2))
	assert.True(t, v1.IsNewerThan(v2))
	assert.False(t, v2.IsNewerThanOrEqual(v1))
	assert.False(t, v2.IsNewerThan(v1))

	v2, err = NewSemVer("v1.18.2")
	assert.Nil(t, err)
	assert.False(t, v1 == v2)
	assert.True(t, v1.IsNewerThanOrEqual(v2))
	assert.True(t, v1.IsNewerThan(v2))
	assert.False(t, v2.IsNewerThanOrEqual(v1))
	assert.False(t, v2.IsNewerThan(v1))

	v2, err = NewSemVer("3.11.2")
	assert.Nil(t, err)
	assert.False(t, v1 == v2)
	assert.False(t, v1.IsNewerThanOrEqual(v2))
	assert.False(t, v1.IsNewerThan(v2))
	assert.True(t, v2.IsNewerThanOrEqual(v1))
	assert.True(t, v2.IsNewerThan(v1))

	v2, err = NewSemVer("3.11.2-SNAPSHOT")
	assert.Nil(t, err)
	assert.False(t, v1 == v2)
	assert.False(t, v1.IsNewerThanOrEqual(v2))
	assert.False(t, v1.IsNewerThan(v2))
	assert.True(t, v2.IsNewerThanOrEqual(v1))
	assert.True(t, v2.IsNewerThan(v1))
}
