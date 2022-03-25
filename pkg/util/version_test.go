package util_test

import (
	"github.com/sirupsen/logrus"
	"github.com/skatteetaten/architect/v2/pkg/util"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestVersionWithMinorAndPatch(t *testing.T) {
	assert.True(t, util.IsSemanticVersion("1"))
	assert.True(t, util.IsSemanticVersion("1.1"))
	assert.True(t, util.IsSemanticVersion("1.1.1"))
	assert.True(t, util.IsSemanticVersion("1.11.1"))
	assert.True(t, util.IsSemanticVersion("12.11.12"))
	assert.True(t, util.IsSemanticVersion("12.11.12+somemeta"))
	assert.False(t, util.IsSemanticVersion("12.11.12+some_meta"))
	assert.False(t, util.IsSemanticVersion("12.11.12++somemeta"))
	assert.False(t, util.IsSemanticVersion("12.11.12+some-meta"))
	assert.False(t, util.IsSemanticVersion("12.11.12+some?meta"))
	assert.False(t, util.IsSemanticVersion("12.11.12+some meta"))
	assert.True(t, util.IsSemanticVersion("12.11.12+1b2345"))
	assert.False(t, util.IsSemanticVersion("v12.11.12+1b2345"))
	assert.False(t, util.IsSemanticVersion("12.11.12.23"))
	assert.True(t, util.IsSemanticVersion("2019.03.08"))
	assert.True(t, util.IsSemanticVersion("2019.03.08+dailyrelease"))
	assert.True(t, util.IsSemanticVersion("2019.03+monthlyrelease"))
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
	assert.True(t, util.IsFullSemanticVersion("1.11.1+somemeta"))
	assert.False(t, util.IsFullSemanticVersion("1.11+somemeta"))
	assert.False(t, util.IsFullSemanticVersion("1.11-prerelase+andmeta"))
	assert.False(t, util.IsFullSemanticVersion("1.11.2.3+meta"))
	assert.True(t, util.IsFullSemanticVersion("1.11.2+123metaap"))
	assert.True(t, util.IsFullSemanticVersion("2019.03.08"))
	assert.True(t, util.IsFullSemanticVersion("2019.03.08+dailyrelease"))
	assert.False(t, util.IsFullSemanticVersion("2019.03+monthlyrelease"))
}

func TestFullSemanticVersionWithMeta(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	assert.True(t, util.IsSemanticVersionWithMeta("1.1.1+noemeta"))
	assert.True(t, util.IsSemanticVersionWithMeta("1.1+noemeta"))
	assert.False(t, util.IsSemanticVersionWithMeta("1.1-pre+noemeta"))
	assert.False(t, util.IsSemanticVersionWithMeta("1.1+noe_meta"))
	assert.False(t, util.IsSemanticVersionWithMeta("1.1+noe-meta"))
	assert.False(t, util.IsSemanticVersionWithMeta("1.1+noe meta"))
	assert.False(t, util.IsSemanticVersionWithMeta("1.1-pre"))
	assert.False(t, util.IsSemanticVersionWithMeta("1.1"))
	assert.False(t, util.IsSemanticVersionWithMeta("1.2.3"))
}

func TestGetVersionOnly(t *testing.T) {
	assert.Equal(t, "1.2.3", util.GetVersionWithoutMetadata("1.2.3+metdata"))
	assert.Equal(t, "1.2", util.GetVersionWithoutMetadata("1.2+metdata"))
	assert.Equal(t, "1.2", util.GetVersionWithoutMetadata("1.2"))
	assert.Equal(t, "2.2.2", util.GetVersionWithoutMetadata("2.2.2"))
	assert.Equal(t, "2.a.b", util.GetVersionWithoutMetadata("2.a.b"))
	assert.Equal(t, "2.a.b", util.GetVersionWithoutMetadata("2.a.b+metadata"))
}
