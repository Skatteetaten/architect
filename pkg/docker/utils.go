package docker

import (
	"github.com/skatteetaten/architect/pkg/config/runtime"
	"time"
)

// CreateCompleteTagsFromSpecAndTags makes a target image to be used for push.
// The tag format in docker is somewhat confusing. For a description, see
// https://docs.docker.com/engine/reference/commandline/tag/
// Given format a list of tags consisting of "a", "b" and "c", a registry host:5000 and repository "aurora/test",
// this function will return:
// host:5000/aurora/test:a
// host:5000/aurora/test:b
// host:5000/aurora/test:c
func CreateImageNameFromSpecAndTags(tags []string, outputRegistry string, outputRepository string) []string {
	output := make([]string, len(tags))
	for i, t := range tags {
		name := &runtime.DockerImage{
			Registry:   outputRegistry,
			Repository: outputRepository,
			Tag:        t,
		}
		output[i] = name.GetCompleteDockerTagName()
	}
	return output
}

func GetUtcTimestamp() string {
	location, _ := time.LoadLocation("UTC")
	return time.Now().In(location).Format(time.RFC3339)

}
