package prepare

import (
	global "github.com/skatteetaten/architect/pkg/config"
	"github.com/skatteetaten/architect/pkg/config/runtime"
	"github.com/skatteetaten/architect/pkg/docker"
	"github.com/skatteetaten/architect/pkg/java/config"
)

//ImageMetadata
type ImageMetadata struct {
	BaseImage  string
	Maintainer string
	Labels     map[string]string
	Env        map[string]string
}

func createEnv(auroraVersion runtime.AuroraVersion, pushextratags global.PushExtraTags, imageBuildTime string) map[string]string {
	env := make(map[string]string)
	env[docker.EnvAppVersion] = string(auroraVersion.GetAppVersion())
	env[docker.EnvAuroraVersion] = auroraVersion.GetCompleteVersion()
	env[docker.EnvPushExtraTags] = pushextratags.ToStringValue()
	env[docker.TZ] = "Europe/Oslo"
	env[docker.ImageBuildTime] = imageBuildTime

	if auroraVersion.Snapshot {
		env[docker.EnvSnapshotVersion] = auroraVersion.GetGivenVersion()
	}

	return env
}

func createLabels(meta config.DeliverableMetadata) map[string]string {
	var labels = make(map[string]string)

	for k, v := range meta.Docker.Labels {
		labels[k] = v
	}

	return labels
}
