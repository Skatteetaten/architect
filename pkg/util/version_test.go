package util_test

import (
	"github.com/skatteetaten/architect/pkg/util"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestVersionWithMinorAndPatch(t *testing.T) {
	assert.True(t, util.IsSemanticVersion("1"))
	assert.True(t, util.IsSemanticVersion("1.1"))
	assert.True(t, util.IsSemanticVersion("1.1.1"))
	assert.True(t, util.IsSemanticVersion("1.11.1"))
	assert.True(t, util.IsSemanticVersion("12.11.12"))
	assert.False(t, util.IsSemanticVersion("12.11.F"))
	assert.False(t, util.IsSemanticVersion("12.11.12-23"))
	assert.False(t, util.IsSemanticVersion("12.11.12.2"))
	assert.False(t, util.IsSemanticVersion("1b.11.12"))
	assert.False(t, util.IsSemanticVersion("sussebass"))
}

func TestFullSemanticVersion(t *testing.T) {
	assert.False(t, util.IsFullSemanticVersion("1"))
	assert.False(t, util.IsFullSemanticVersion("1.1"))
	assert.True(t, util.IsFullSemanticVersion("1.1.1"))
	assert.True(t, util.IsFullSemanticVersion("1.11.1"))
	assert.True(t, util.IsFullSemanticVersion("12.11.12"))
	assert.False(t, util.IsFullSemanticVersion("12.11.F"))
	assert.False(t, util.IsFullSemanticVersion("12.11.12-23"))
	assert.False(t, util.IsFullSemanticVersion("12.11.12.2"))
	assert.False(t, util.IsFullSemanticVersion("1b.11.12"))
	assert.False(t, util.IsFullSemanticVersion("sussebass"))
}
