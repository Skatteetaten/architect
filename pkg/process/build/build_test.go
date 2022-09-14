package process_test

import (
	"context"
	"github.com/golang/mock/gomock"
	"github.com/skatteetaten/architect/v2/pkg/config"
	"github.com/skatteetaten/architect/v2/pkg/config/runtime"
	"github.com/skatteetaten/architect/v2/pkg/docker"
	docker_mock "github.com/skatteetaten/architect/v2/pkg/docker/mocks"
	"github.com/skatteetaten/architect/v2/pkg/nexus"
	nexus_mock "github.com/skatteetaten/architect/v2/pkg/nexus/mocks"
	"github.com/skatteetaten/architect/v2/pkg/process/build"
	build_mock "github.com/skatteetaten/architect/v2/pkg/process/build/mocks"
	"github.com/skatteetaten/architect/v2/pkg/trace"
	trace_mock "github.com/skatteetaten/architect/v2/pkg/trace/mocks"
	"testing"
)

func TestBuild(t *testing.T) {

	t.Run("Overwrite should NOT be allowed for semanticVersion", func(t *testing.T) {
		isSnapshot := false
		tagWith := ""
		tags := []string{"1.3.1"}
		semanticVersion := "1.3.1"
		completeVersion := ""
		err := process.CheckTagsForOverwrite(isSnapshot, tags, tagWith, semanticVersion, completeVersion)

		if err == nil {
			t.Fatal("Overwrite should not be allowed")
		}
	})

	t.Run("Overwrite should NOT be allowed for completeVersion", func(t *testing.T) {
		isSnapshot := false
		tagWith := ""
		tags := []string{"1.3.1"}
		semanticVersion := ""
		completeVersion := "1.3.1"
		err := process.CheckTagsForOverwrite(isSnapshot, tags, tagWith, semanticVersion, completeVersion)

		if err == nil {
			t.Fatal("Overwrite should not be allowed")
		}
	})

	t.Run("Overwrite should NOT be allowed for tagWith", func(t *testing.T) {
		isSnapshot := false
		tagWith := "1.3.1"
		tags := []string{"1.3.1"}
		semanticVersion := ""
		completeVersion := ""
		err := process.CheckTagsForOverwrite(isSnapshot, tags, tagWith, semanticVersion, completeVersion)

		if err == nil {
			t.Fatal("Overwrite should be allowed for tagWith")
		}
	})

	t.Run("Overwrite should be allowed for snapshot", func(t *testing.T) {
		isSnapshot := true
		tagWith := ""
		tags := []string{"1.3.1"}
		semanticVersion := "1.3.1"
		completeVersion := ""
		err := process.CheckTagsForOverwrite(isSnapshot, tags, tagWith, semanticVersion, completeVersion)

		if err != nil {
			t.Fatal("Overwrite should be allowed for snapshot")
		}
	})

	t.Run("Overwrite should be allowed for tagWith-snapshot", func(t *testing.T) {
		isSnapshot := false
		tagWith := "1.3.1-SNAPSHOT"
		tags := []string{"1.3.1"}
		semanticVersion := ""
		completeVersion := ""
		err := process.CheckTagsForOverwrite(isSnapshot, tags, tagWith, semanticVersion, completeVersion)

		if err != nil {
			t.Fatal("Overwrite should be allowed for tagWith-snapshot")
		}
	})

	t.Run("Test that build sends correct data to Sporingslogger ", func(t *testing.T) {

		ctx := context.Background()

		appSpec := config.ApplicationSpec{
			config.MavenGav{
				"ArtifactId",
				"GroupId",
				"Version111",
				"Classifier",
				"typeVersion",
			},
			config.DockerBaseImageSpec{
				"BaseImageName",
				"BaseVersion",
			},
		}

		dockerSpec := config.DockerSpec{
			"OutputRegistry",
			"OutputRepository",
			"InternalPullRegistry",
			config.PushExtraTags{
				false,
				true,
				false,
				false,
			},
			//This is the external docker registry where we check versions.
			"ExternalDockerRegistry",
			//The tag to push to. This is only used for ImageStreamTags (as for now) and RETAG functionality
			"TagWith",
			"RetagWith",
		}

		testConfig := config.Config{
			"ApplicationType",
			appSpec,
			dockerSpec,
			config.BuilderSpec{
				"BuildImageVersion123",
			},
			true,
			true,
			true,
			10,
			false,
			"Sporingstjeneste",
			"OwnerReferenceUUID",
			"BinaryBuildType",
			"NexusIQReportURL",
		}

		mockCtrl := gomock.NewController(t)
		registryClient := docker_mock.NewMockRegistry(mockCtrl)
		nexusDownloader := nexus_mock.NewMockDownloader(mockCtrl)
		layerBuilder := build_mock.NewMockBuilder(mockCtrl)
		traceMock := trace_mock.NewMockTrace(mockCtrl)

		registryClient.EXPECT().GetImageInfo(gomock.Any(), "BaseImageName", gomock.Any()).Return(&runtime.ImageInfo{
			CompleteBaseImageVersion: "CompleteBaseImageVersion",
			Labels:                   map[string]string{},
			Enviroment:               map[string]string{},
			Digest:                   "BaseImageDigest",
		}, nil).AnyTimes()

		registryClient.EXPECT().GetImageInfo(gomock.Any(), "ServiceNameTest", gomock.Any()).Return(&runtime.ImageInfo{
			CompleteBaseImageVersion: "CompleteBaseImageVersion",
			Labels:                   map[string]string{},
			Environment:              map[string]string{},
			Digest:                   "ImageDigest",
		}, nil).AnyTimes()

		imageConfig := make(map[string]interface{})
		imageConfig["image-config"] = "config123"

		registryClient.EXPECT().GetTags(gomock.Any(), gomock.Any()).Return(&docker.TagsAPIResponse{
			Name: "name",
			Tags: []string{"tag1", "tag2"},
		}, nil)

		nexusDownloader.EXPECT().DownloadArtifact(gomock.Any()).Return(nexus.Deliverable{
			Path: "PATH",
			SHA1: "SHA1",
		}, nil)

		layerBuilder.EXPECT().Pull(gomock.Any(), gomock.Any()).Return(nil, nil)
		layerBuilder.EXPECT().Push(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		layerBuilder.EXPECT().Build(gomock.Any(), gomock.Any()).Return(nil, nil)

		mockPrepper := func(
			cfg *config.Config,
			auroraVersion *runtime.AuroraVersion,
			deliverable nexus.Deliverable,
			baseImage runtime.BaseImage) (*docker.BuildConfig, error) {
			return &docker.BuildConfig{
				runtime.NewAuroraVersion("1.2.3", false, "giverVersioin", "completeVersion"), //AuroraVersion    *runtime.AuroraVersion
				"ServiceNameTest",
				"BuildFolder",
				runtime.DockerImage{
					"TAG-test",
					"repo-test",
					"registry-test",
				},
				"OutputRegistry",
				map[string]string{},
				map[string]string{},
				nil,
				nil,
			}, nil
		}

		tags := make(map[string]string)
		tags["TagWith"] = "OutputRegistry/ServiceNameTest:TagWith"

		traceMock.EXPECT().AddImageMetadata(gomock.Eq(
			trace.DeployableImage{
				Type:             "deployableImage",
				Digest:           "ImageDigest",
				Name:             "ServiceNameTest",
				AppVersion:       "Version111",
				BaseImageName:    "BaseImageName",
				BaseImageVersion: "CompleteBaseImageVersion",
				BaseImageDigest:  "BaseImageDigest",
				BuildVersion:     "BuildImageVersion123",
				Snapshot:         false,
			}))

		err := process.Build(ctx, registryClient, registryClient, &testConfig, nexusDownloader, mockPrepper, layerBuilder, traceMock)

		if err != nil {
			t.Fatal("Overwrite should be allowed for tagWith-snapshot")
		}
	})

}
