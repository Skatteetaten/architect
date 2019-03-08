package util

import (
	"regexp"
	"strings"
)

//We limit to four digits... Git commits tend to be only nummeric as well
var versionWithOptionalMinorAndPatch = regexp.MustCompile(`^[0-9]{1,5}(\.[0-9]+(\.[0-9]+)?)?(\+([0-9A-Za-z]+))?$`)
var versionWithMinorAndPatch = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$|^[0-9]+\.[0-9]+\.[0-9]+\+([0-9A-Za-z]+)$`)
var versionMeta = regexp.MustCompile(`\+([0-9A-Za-z]+)$`)

func IsFullSemanticVersion(versionString string) bool {
	if versionWithMinorAndPatch.MatchString(versionString) {
		return true
	}
	return false
}

func IsSemanticVersion(versionString string) bool {
	if versionWithOptionalMinorAndPatch.MatchString(versionString) {
		return true
	}
	return false
}

func IsSemanticVersionWithMeta(versionString string) bool {
	if IsSemanticVersion(versionString) && versionMeta.MatchString(versionString) {
		return true
	}
	return false
}

func GetVersionWithoutMetadata(versionString string) string {
	matches := versionMeta.FindStringSubmatch(versionString)
	if matches == nil {
		return versionString
	}
	return strings.Replace(versionString, "+"+matches[1], "", -1)
}

func GetVersionMetadata(versionString string) string {
	matches := versionMeta.FindStringSubmatch(versionString)
	if matches == nil {
		return ""
	}
	return matches[1]
}
