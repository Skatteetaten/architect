package util

import (
	"github.com/docker/distribution/reference"
	"github.com/pkg/errors"
	"strings"
)

//FindOutputTagOrHash get tag or hash from docker name
func FindOutputTagOrHash(dockerName string) (string, error) {
	//In case when working with insecure registries
	dockerName = strings.Replace(dockerName, "http://", "", -1)
	name, err := reference.ParseNamed(dockerName)
	if err != nil {
		return "", errors.Wrap(err, "Error parsing docker registry reference")
	}

	if tagged, isTagged := name.(reference.NamedTagged); isTagged {
		return tagged.Tag(), nil
	}

	if digested, isDigested := name.(reference.Digested); isDigested {
		return digested.Digest().String(), nil
	}

	return "", errors.Errorf("Could not parse tag from %s", dockerName)
}
