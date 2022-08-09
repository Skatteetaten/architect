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
	"net/http"
	"net/http/httptest"
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

		//mockgen -destination=pkg/docker/mocks/mock_registry.go -package=mocks -source=pkg/docker/registry.go Registry
		//mockgen -destination=pkg/nexus/mocks/mock_nexus.go -package=mocks -source=pkg/nexus/nexus.go Downloader
		// mockgen -destination=pkg/process/build/mocks/mock_layer_builder.go -package=mocks -source=pkg/process/build/build.go  Builder
		//mockgen -destination=pkg/trace/mocks/mock_trace.go -package=mocks -source=pkg/trace/trace.go  Trace

		ctx := context.Background()

		appSpec := config.ApplicationSpec{
			config.MavenGav{
				"ArtifactId",
				"GroupId",
				"Version111",
				"Classifier",
				"Type",
			},
			config.DockerBaseImageSpec{
				"BaseImage",
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
				"BaseImageVersion123",
			},
			true,
			true,
			true,
			10,
			false,
			"Sporingstjeneste",
			"OwnerReferenceUid",
			"BinaryBuildType",
			"NexusIQReportUrl",
		}

		mockCtrl := gomock.NewController(t)
		registryClient := docker_mock.NewMockRegistry(mockCtrl)
		nexusDownloader := nexus_mock.NewMockDownloader(mockCtrl)
		layerBuilder := build_mock.NewMockBuilder(mockCtrl)
		traceMock := trace_mock.NewMockTrace(mockCtrl)

		registryClient.EXPECT().GetImageInfo(gomock.Any(), gomock.Any(), gomock.Any()).Return(&runtime.ImageInfo{
			CompleteBaseImageVersion: "CompleteBaseImageVersion",
			Labels:                   map[string]string{},
			Enviroment:               map[string]string{},
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
				"DockerRepository",
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
		tags["TagWith"] = "OutputRegistry/DockerRepository:TagWith"

		traceMock.EXPECT().AddImageMetadata(gomock.Eq(
			trace.DeployableImage{
				Type:          "deployableImage",
				Digest:        "ImageDigest",
				Name:          "DockerRepository",
				AppVersion:    "Version111",
				AuroraVersion: "Version111-bBaseImageVersion123-BaseImage-CompleteBaseImageVersion",
				Snapshot:      false,
				// TODO add GitCommit      "Missing",
				//Dependencies: dependencyMetadata, TODO -fix
				//public List<Dependency> dependencies;

			}))

		err := process.Build(ctx, registryClient, registryClient, &testConfig, nexusDownloader, mockPrepper, layerBuilder, traceMock)

		if err != nil {
			t.Fatal("Overwrite should be allowed for tagWith-snapshot")
		}
	})

}
func FakeEndpoint(t *testing.T, endpoint func(w http.ResponseWriter, r *http.Request)) *httptest.Server {

	srv := httptest.NewServer(http.HandlerFunc(endpoint))

	return srv
}
