package util

import (
	extVersion "github.com/hashicorp/go-version"
	"regexp"
	"strings"
)

//We limit to four digits... Git commits tend to be only nummeric as well
var versionWithOptionalMinorAndPatch = regexp.MustCompile(`^[0-9]{1,3}(\.[0-9]+(\.[0-9]+)?)?$`)
var versionWithMinorAndPatch = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)

func IsFullSemanticVersion(versionString string) bool {
	versionStringWithoutMeta := GetVersionWithoutMetadata(versionString)
	if versionWithMinorAndPatch.MatchString(versionStringWithoutMeta) {
		return true
	}
	return false
}

func IsSemanticVersion(versionString string) bool {
	versionStringWithoutMeta := GetVersionWithoutMetadata(versionString)
	if versionWithOptionalMinorAndPatch.MatchString(versionStringWithoutMeta) {
		return true
	}
	return false
}

func GetVersionWithoutMetadata(versionString string) string {
	c, err := extVersion.NewSemver(versionString)
	if err == nil {
		if len(c.Metadata()) > 0 {
			return strings.Replace(c.Original(), "+"+c.Metadata(), "", -1)
		}
	}
	return versionString
}

func GetVersionMetadata(versionString string) string {
	c, err := extVersion.NewSemver(versionString)
	if err != nil {
		return ""
	}
	return c.Metadata()
}

func IsSemanticVersionWithMeta(versionString string) bool {
	c, err := extVersion.NewSemver(versionString)

	if err != nil {
		return false
	}

	if len(c.Metadata()) > 0 {
		return true
	}
	return false
}
