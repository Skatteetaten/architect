package process

import (
	"github.com/Sirupsen/logrus"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/skatteetaten/architect/pkg/config"
	"github.com/skatteetaten/architect/pkg/config/runtime"
	"github.com/skatteetaten/architect/pkg/docker"
	"github.com/skatteetaten/architect/pkg/nexus"
	"github.com/skatteetaten/architect/pkg/process/tagger"
	"log"
	"os"
	"os/exec"
)


type Builder interface {
	Build(ruuid string, context string, buildFolder string) error
	Push(ruuid string, tag string) error
	Tag(ruuid string, tag string) error
}

type BuildahCmd struct {

}

func (BuildahCmd) Build(ruuid string, context string, buildFolder string) error{
	build := exec.Command("buildah", "--storage-driver", "vfs", "bud", "--isolation", "chroot", "-t", ruuid, "-f", context, buildFolder)
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	return build.Run()
}


func (BuildahCmd) Push(ruuid string , tag string) error {
	cmd := exec.Command("buildah", "--storage-driver", "vfs", "push", ruuid, tag)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (BuildahCmd) Tag(ruuid string, tag string) error {
	cmd := exec.Command("buildah", "--storage-driver", "vfs", "tag", ruuid, tag)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}


//TODO: Implement credentials support (Vault ?)
func Build(credentials *docker.RegistryCredentials, provider docker.ImageInfoProvider, cfg *config.Config, downloader nexus.Downloader, prepper Prepper, builder Builder) error {

	logrus.Debugf("Download deliverable for GAV %-v", cfg.ApplicationSpec)
	deliverable, err := downloader.DownloadArtifact(&cfg.ApplicationSpec.MavenGav)
	if err != nil {
		return errors.Wrapf(err, "Could not download deliverable %-v", cfg.ApplicationSpec)
	}
	application := cfg.ApplicationSpec
	logrus.Debug("Extract build info")

	imageInfo, err := provider.GetImageInfo(application.BaseImageSpec.BaseImage,
		application.BaseImageSpec.BaseVersion)
	if err != nil {
		return errors.Wrap(err, "Unable to get the complete build version")
	}

	completeBaseImageVersion := imageInfo.CompleteBaseImageVersion

	baseImage := runtime.BaseImage{
		DockerImage: runtime.DockerImage{
			Tag:        completeBaseImageVersion,
			Repository: application.BaseImageSpec.BaseImage,
			Registry:   cfg.DockerSpec.GetExternalRegistryWithoutProtocol(),
		},
		ImageInfo: imageInfo,
	}

	buildImage := &runtime.ArchitectImage{
		Tag: cfg.BuilderSpec.Version,
	}
	snapshot := application.MavenGav.IsSnapshot()
	appVersion := nexus.GetSnapshotTimestampVersion(application.MavenGav, deliverable)
	auroraVersion := runtime.NewAuroraVersionFromBuilderAndBase(appVersion, snapshot,
		application.MavenGav.Version, buildImage, baseImage.DockerImage)

	dockerBuildConfig, err := prepper(cfg, auroraVersion, deliverable, baseImage)
	if err != nil {
		return errors.Wrap(err, "Error preparing image")
	}

	if !cfg.DockerSpec.TagOverwrite {
		for _, buildConfig := range dockerBuildConfig {
			if !buildConfig.AuroraVersion.Snapshot {
				tags, err := provider.GetTags(cfg.DockerSpec.OutputRepository)
				if err != nil {
					return err
				}
				completeVersion := buildConfig.AuroraVersion.GetCompleteVersion()
				for _, tag := range tags.Tags {
					if tag == completeVersion {
						return errors.Errorf("There are already a build with tag %s, consider TAG_OVERWRITE", completeVersion)
					}
				}
			}
		}
	}

	for _, buildConfig := range dockerBuildConfig {

		ruuid, err := uuid.NewUUID()

		if err != nil {
			log.Fatalf("UUID generation failed: %s ", err)
		}

		context := buildConfig.BuildFolder + "/Dockerfile"

		logrus.Info("Docker context ", buildConfig.BuildFolder)

		err = builder.Build(ruuid.String(), context, buildConfig.BuildFolder)

		if err != nil {
			return errors.Wrap(err, "There was an error with the Docker build operation.")
		} else {
			logrus.Infof("Done building. Imageid: %s", ruuid)
		}

		var tagResolver tagger.TagResolver
		if cfg.DockerSpec.TagWith == "" {
			tagResolver = &tagger.NormalTagResolver{
				Overwrite:  cfg.DockerSpec.TagOverwrite,
				Provider:   provider,
				Registry:   cfg.DockerSpec.OutputRegistry,
				Repository: buildConfig.DockerRepository,
			}
		} else {
			tagResolver = &tagger.SingleTagTagResolver{
				Tag:        cfg.DockerSpec.TagWith,
				Registry:   cfg.DockerSpec.OutputRegistry,
				Repository: buildConfig.DockerRepository,
			}
		}

		tags, err := tagResolver.ResolveTags(buildConfig.AuroraVersion, cfg.DockerSpec.PushExtraTags)
		logrus.Debugf("Tag image %s with %s", ruuid, tags)

		for _, tag := range tags {
			err = builder.Tag(ruuid.String(), tag)
			if err != nil {
				return errors.Wrapf(err, "Image tag with buildah failed")
			}
		}

		for _, tag := range tags {
			err = builder.Push(ruuid.String(), tag)
			if err != nil {
				return errors.Wrapf(err, "Image push with buildah failed")
			}
		}
	}
	return nil
}


//TODO: Remove legacy later
//TODO: Write some test for this..
func LegacyBuild(credentials *docker.RegistryCredentials, provider docker.ImageInfoProvider, cfg *config.Config, downloader nexus.Downloader, prepper Prepper) error {

	logrus.Debugf("Download deliverable for GAV %-v", cfg.ApplicationSpec)
	deliverable, err := downloader.DownloadArtifact(&cfg.ApplicationSpec.MavenGav)
	if err != nil {
		return errors.Wrapf(err, "Could not download deliverable %-v", cfg.ApplicationSpec)
	}
	application := cfg.ApplicationSpec
	logrus.Debug("Extract build info")

	imageInfo, err := provider.GetImageInfo(application.BaseImageSpec.BaseImage,
		application.BaseImageSpec.BaseVersion)
	if err != nil {
		return errors.Wrap(err, "Unable to get the complete build version")
	}

	completeBaseImageVersion := imageInfo.CompleteBaseImageVersion

	baseImage := runtime.BaseImage{
		DockerImage: runtime.DockerImage{
			Tag:        completeBaseImageVersion,
			Repository: application.BaseImageSpec.BaseImage,
			Registry:   cfg.DockerSpec.GetExternalRegistryWithoutProtocol(),
		},
		ImageInfo: imageInfo,
	}

	buildImage := &runtime.ArchitectImage{
		Tag: cfg.BuilderSpec.Version,
	}
	snapshot := application.MavenGav.IsSnapshot()
	appVersion := nexus.GetSnapshotTimestampVersion(application.MavenGav, deliverable)
	auroraVersion := runtime.NewAuroraVersionFromBuilderAndBase(appVersion, snapshot,
		application.MavenGav.Version, buildImage, baseImage.DockerImage)
	if err != nil {
		return errors.Wrap(err, "Error creating version information")
	}

	dockerBuildConfig, err := prepper(cfg, auroraVersion, deliverable, baseImage)
	if err != nil {
		return errors.Wrap(err, "Error preparing image")
	}

	if !cfg.DockerSpec.TagOverwrite {
		for _, buildConfig := range dockerBuildConfig {
			if !buildConfig.AuroraVersion.Snapshot {
				tags, err := provider.GetTags(cfg.DockerSpec.OutputRepository)
				if err != nil {
					return err
				}
				completeVersion := buildConfig.AuroraVersion.GetCompleteVersion()
				for _, tag := range tags.Tags {
					if tag == completeVersion {
						return errors.Errorf("There are already a build with tag %s, consider TAG_OVERWRITE", completeVersion)
					}
				}
			}
		}
	}

	client, err := docker.NewDockerClient()
	if err != nil {
		return errors.Wrap(err, "Error initializing Docker")
	}

	for _, buildConfig := range dockerBuildConfig {
		client.PullImage(buildConfig.Baseimage)
		imageid, err := client.BuildImage(buildConfig.BuildFolder)

		if err != nil {
			return errors.Wrap(err, "There was an error with the Docker build operation.")
		} else {
			logrus.Infof("Done building. Imageid: %s", imageid)
		}

		var tagResolver tagger.TagResolver
		if cfg.DockerSpec.TagWith == "" {
			tagResolver = &tagger.NormalTagResolver{
				Overwrite:  cfg.DockerSpec.TagOverwrite,
				Provider:   provider,
				Registry:   cfg.DockerSpec.OutputRegistry,
				Repository: buildConfig.DockerRepository,
			}
		} else {
			tagResolver = &tagger.SingleTagTagResolver{
				Tag:        cfg.DockerSpec.TagWith,
				Registry:   cfg.DockerSpec.OutputRegistry,
				Repository: buildConfig.DockerRepository,
			}
		}

		tags, err := tagResolver.ResolveTags(buildConfig.AuroraVersion, cfg.DockerSpec.PushExtraTags)
		logrus.Debugf("Tag image %s with %s", imageid, tags)
		for _, tag := range tags {
			err = client.TagImage(imageid, tag)
			if err != nil {
				return err
			}
		}
		err = client.PushImages(tags, credentials)
		if err != nil {
			return errors.Wrap(err, "Error pushing images")
		}
	}
	return nil
}
