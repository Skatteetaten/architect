package config

import (
	extVersion "github.com/hashicorp/go-version"
	"fmt"
)

func GetMajor(version string) (string, error) {
	build_version, err := extVersion.NewVersion(version)

	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%d", build_version.Segments()[0]), nil
}

func GetMinor(version string) (string, error) {
	build_version, err := extVersion.NewVersion(version)

	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%d.%d", build_version.Segments()[0],build_version.Segments()[1]), nil
}

func GetCompleteVersion() () {

}

func GetVersionTags() () {

}