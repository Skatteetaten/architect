package process

import (
	"context"
	"encoding/json"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/skatteetaten/architect/pkg/config"
	"github.com/skatteetaten/architect/pkg/docker"
	"github.com/skatteetaten/architect/pkg/util"
	"io/ioutil"
)

type LayerBuilder struct {
	config   *config.Config
	provider docker.Registry
}

const (
	manifestFileName        = "manifest.json"
	containerConfigFileName = "config.json"
)

func NewLayerBuilder(config *config.Config, provider docker.Registry) Builder {
	return &LayerBuilder{
		config:   config,
		provider: provider,
	}
}

func (l *LayerBuilder) Pull(ctx context.Context, buildConfig docker.DockerBuildConfig) error {
	baseImage := buildConfig.Baseimage

	logrus.Infof("%s:%s", baseImage.Repository, baseImage.Tag)
	manifest, err := l.provider.GetManifest(baseImage.Repository, baseImage.Tag)
	if err != nil {
		return errors.Wrap(err, "Failed to fetch manifest")
	}
	logrus.Infof("Fetched manifest: %s/%s:%s", baseImage.Registry, baseImage.Repository, baseImage.Tag)

	for _, layer := range manifest.Layers {
		ok, _ := l.provider.LayerExists(l.config.DockerSpec.OutputRepository, layer.Digest)
		if !ok {
			err := l.provider.MountLayer(baseImage.Repository, l.config.DockerSpec.OutputRepository, layer.Digest)
			if err != nil {
				return errors.Wrap(err, "Layer mount failed")
			}
			logrus.Infof("Mounted layer %s", layer.Digest)
		}
	}

	containerConfig, err := l.provider.GetContainerConfig(baseImage.Repository, manifest.Config.Digest)
	if err != nil {
		return errors.Wrap(err, "Failed to fetch the container config")
	}

	logrus.Infof("Fetched container config: %s", manifest.Config.Digest)

	err = manifest.Save(buildConfig.BuildFolder, manifestFileName)
	if err != nil {
		return errors.Wrap(err, "Manifest save: failed")
	}

	err = containerConfig.Save(buildConfig.BuildFolder, containerConfigFileName)
	if err != nil {
		return errors.Wrap(err, "Container config save: failed")
	}
	return nil
}

func (l *LayerBuilder) Build(ctx context.Context, buildConfig docker.DockerBuildConfig) (*BuildOutput, error) {
	buildFolder := buildConfig.BuildFolder

	files, err := ioutil.ReadDir(buildConfig.BuildFolder + "/" + util.LayerFolder)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to the read the layer folder")
	}

	var layers []Layer
	for _, file := range files {
		if file.IsDir() {
			logrus.Infof("Handle layer: %s", file.Name())
			layer, err := util.CompressTarGz(buildFolder+"/"+util.LayerFolder, file.Name(), buildFolder)
			if err != nil {
				return nil, errors.Wrapf(err, "Compress of layer %s failed", file.Name())
			}

			contentDigest, err := util.CalculateDigestFromArchive(buildFolder + "/" + layer)
			if err != nil {
				return nil, errors.Wrap(err, "Failed to calculate the content digest")
			}

			digest, err := util.CalculateDigest(buildFolder + "/" + layer)
			if err != nil {
				return nil, errors.Wrapf(err, "Manifest generation: Digest calculation failed on layer %s", layer)
			}

			size, err := util.CalculateSize(buildFolder + "/" + layer)
			if err != nil {
				return nil, errors.Wrapf(err, "Manifest generation: Unable to stat layer %s", layer)
			}

			layers = append(layers, Layer{
				Name:          layer,
				Digest:        digest,
				Size:          size,
				ContentDigest: contentDigest,
			})
		}
	}

	//Update the manifest
	manifest, err := getManifest(buildConfig.BuildFolder)
	if err != nil {
		return nil, err
	}

	containerConfig, err := getContainerConfig(buildConfig.BuildFolder)
	if err != nil {
		return nil, err
	}

	//Removes unnecessary data
	manifest = manifest.CleanCopy()

	containerConfig = containerConfig.CleanCopy()

	//Add layers to the container configuration
	for _, layer := range layers {
		//Add content digest to RootFS
		containerConfig = containerConfig.AddLayer(layer.ContentDigest)

		//Modify the manifest
		manifest.Layers = append(manifest.Layers, docker.Layer{
			MediaType: "application/vnd.docker.image.rootfs.diff.tar.gzip",
			Size:      layer.Size,
			Digest:    layer.Digest,
		})

	}
	containerConfig.AddEnv(buildConfig.Env)
	containerConfig.AddLabels(buildConfig.Labels)

	//TODO: Handle Cmd or entrypoint

	err = containerConfig.Save(buildFolder, containerConfigFileName)
	if err != nil {
		return nil, errors.Wrap(err, "Save operation failed")
	}

	size, err := util.CalculateSize(buildFolder + "/" + containerConfigFileName)
	if err != nil {
		return nil, errors.Wrap(err, "ContainerConfig: File stat failed")
	}

	digest, err := util.CalculateDigest(buildFolder + "/" + containerConfigFileName)
	if err != nil {
		return nil, errors.Wrap(err, "ContainerConfig: Digest calculation failed")
	}

	manifest.Config.Size = size
	manifest.Config.Digest = digest

	err = manifest.Save(buildFolder, manifestFileName)
	if err != nil {
		return nil, err
	}

	return &BuildOutput{
		ImageId:               "",
		BuildFolder:           buildFolder,
		ContainerConfigDigest: digest,
		Layers:                layers,
	}, nil
}

func (l *LayerBuilder) Push(ctx context.Context, buildOutput *BuildOutput, tag []string, credentials *docker.RegistryCredentials) error {

	buildFolder := buildOutput.BuildFolder
	for _, layer := range buildOutput.Layers {
		logrus.Infof("Push: %s", layer.Digest)
		layerLocation := buildOutput.BuildFolder + "/" + layer.Name
		//Push
		err := l.provider.PushLayer(layerLocation, l.config.DockerSpec.OutputRepository, layer.Digest)
		if err != nil {
			return errors.Errorf("Failed to push layer %v", err)
		}
	}

	logrus.Infof("Push config: %s", buildOutput.ContainerConfigDigest)
	err := l.provider.PushLayer(buildFolder+"/"+containerConfigFileName, l.config.DockerSpec.OutputRepository, buildOutput.ContainerConfigDigest)
	if err != nil {
		return errors.New("Failed to push the container configuration")
	}

	for _, t := range tag {
		shortTag, err := util.FindOutputTagOrHash(t)
		if err != nil {
			return errors.Wrap(err, "Tag failed")
		}

		logrus.Infof("Push tag: %s", t)
		err = l.provider.PushManifest(buildFolder+"/"+manifestFileName, l.config.DockerSpec.OutputRepository, shortTag)
		if err != nil {
			return errors.Errorf("Failed to push manifest: %v", err)
		}
	}
	return nil
}

//Not needed
func (l *LayerBuilder) Tag(ctx context.Context, buildOutput *BuildOutput, tag string) error {
	return nil
}

func getManifest(folder string) (*docker.ManifestV2, error) {
	data, err := ioutil.ReadFile(folder + "/" + manifestFileName)
	if err != nil {
		return nil, err
	}

	var manifest docker.ManifestV2
	err = json.Unmarshal(data, &manifest)
	if err != nil {
		return nil, errors.Wrapf(err, "getContainerConfig: Unmarshal operation failed")
	}

	return &manifest, nil
}

func getContainerConfig(folder string) (*docker.ContainerConfig, error) {
	data, err := ioutil.ReadFile(folder + "/" + containerConfigFileName)
	if err != nil {
		return nil, err
	}

	var containerConfig docker.ContainerConfig
	err = json.Unmarshal(data, &containerConfig)
	if err != nil {
		return nil, errors.Wrapf(err, "getContainerConfig: Unmarshal operation failed")
	}

	return &containerConfig, nil
}
