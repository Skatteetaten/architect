package util

import "regexp"

//We limit to four digits... Git commits tend to be only nummeric as well
var versionWithOptionalMinorAndPatch = regexp.MustCompile(`^[0-9]{1,3}(\.[0-9]+(\.[0-9]+)?)?$`)
var versionWithMinorAndPatch = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)

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
