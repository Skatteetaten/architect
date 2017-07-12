package docker

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
		name := &ImageName{outputRegistry,
			outputRepository, t}
		output[i] = name.String()
	}
	return output
}
