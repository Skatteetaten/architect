package process

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/skatteetaten/architect/v2/pkg/config"
	"github.com/skatteetaten/architect/v2/pkg/config/runtime"
	"github.com/skatteetaten/architect/v2/pkg/docker"
	"github.com/skatteetaten/architect/v2/pkg/util"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

// LayerBuilder configuration
type LayerBuilder struct {
	config       *config.Config
	pushRegistry docker.Registry
	pullRegistry docker.Registry
}

// LayerProvider keep track of the image layers
type LayerProvider struct {
	Manifest        *docker.ManifestV2
	ContainerConfig *docker.ContainerConfig
	BaseImage       runtime.DockerImage
	Layers          []Layer
}

// Layer represent an image blob
type Layer struct {
	Digest  string
	Size    int
	Content func(cxt context.Context) (io.ReadCloser, error)
}

// NewLayerBuilder return Builder of type LayerBuilder
func NewLayerBuilder(config *config.Config, pushregistry docker.Registry, pullregistry docker.Registry) Builder {
	return &LayerBuilder{
		config:       config,
		pushRegistry: pushregistry,
		pullRegistry: pullregistry,
	}
}

// Pull layers
func (l *LayerBuilder) Pull(ctx context.Context, buildConfig docker.BuildConfig) (*LayerProvider, error) {
	baseImage := buildConfig.Image

	logrus.Infof("%s:%s", baseImage.Repository, baseImage.Tag)
	manifest, err := l.pullRegistry.GetManifest(ctx, baseImage.Repository, baseImage.Tag)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to fetch manifest")
	}
	logrus.Infof("Fetched manifest: %s/%s:%s", baseImage.Registry, baseImage.Repository, baseImage.Tag)

	containerConfig, err := l.pullRegistry.GetContainerConfig(ctx, baseImage.Repository, manifest.Config.Digest)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to fetch the container config")
	}

	// Include the container configuration blob
	blobs := make([]docker.Layer, len(manifest.Layers))
	copy(blobs, manifest.Layers)
	blobs = append(blobs, docker.Layer{
		MediaType: manifest.Config.MediaType,
		Size:      manifest.Config.Size,
		Digest:    manifest.Config.Digest,
	})

	var layers []Layer
	for _, layer := range blobs {
		ok, _ := l.pushRegistry.LayerExists(ctx, l.config.DockerSpec.OutputRepository, layer.Digest)
		if !ok {
			// Pull missing layers
			ok, err := l.pushRegistry.LayerExists(ctx, baseImage.Repository, layer.Digest)
			if err != nil || !ok {

				missingLayerPath, err := l.pullRegistry.PullLayer(ctx, baseImage.Repository, layer.Digest)
				if err != nil {
					return nil, errors.Wrapf(err, "Pull: Layer pull failed %s", layer.Digest)
				}
				layers = append(layers, Layer{
					Digest: layer.Digest,
					Size:   layer.Size,
					Content: func(cxt context.Context) (io.ReadCloser, error) {
						reader, err := os.Open(missingLayerPath)
						if err != nil {
							return nil, errors.Wrapf(err, "Unable to open layer %s file", missingLayerPath)
						}
						return reader, nil
					},
				})
			}
		}
	}

	return &LayerProvider{
		Manifest:        manifest,
		ContainerConfig: containerConfig,
		BaseImage:       baseImage,
		Layers:          layers,
	}, nil
}

// Build container image
func (l *LayerBuilder) Build(buildConfig docker.BuildConfig, baseImageLayerProvider *LayerProvider) (*LayerProvider, error) {
	buildFolder := buildConfig.BuildFolder
	layerFolder := filepath.Join(buildFolder, util.LayerFolder)

	files, err := ioutil.ReadDir(layerFolder)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to the read the layer folder")
	}

	manifest := baseImageLayerProvider.Manifest.CleanCopy()
	containerConfig := baseImageLayerProvider.ContainerConfig.CleanCopy()

	var layers []Layer
	for _, file := range files {
		if file.IsDir() {

			layerArchiveName, err := util.CompressLayerTarGz(layerFolder, file.Name(), buildFolder)
			if err != nil {
				return nil, errors.Wrapf(err, "Compression of layer %s failed", file.Name())
			}

			layerPath := filepath.Join(buildFolder, layerArchiveName)

			contentDigest, err := util.CalculateDigestFromArchive(layerPath)
			if err != nil {
				return nil, errors.Wrap(err, "Failed to calculate the content digest")
			}

			reader, err := os.Open(layerPath)
			if err != nil {
				return nil, errors.Wrapf(err, "Unable to open layer %s file", file)
			}
			defer reader.Close()

			stat, err := reader.Stat()
			if err != nil {
				return nil, errors.Wrapf(err, "Unable to calculate layer size of layer %s ", file)
			}

			size := int(stat.Size())
			digest, err := util.CalculateDigestFromFile(layerPath)
			if err != nil {
				return nil, errors.Wrapf(err, "Unable to calculate layer digest of layer %s", file)
			}

			//TODO: Disse kan ligge i minne. Det vil øke farten noe
			layers = append(layers, Layer{
				Digest: digest,
				Size:   size,
				Content: func(cxt context.Context) (io.ReadCloser, error) {
					file, err := os.Open(layerPath)
					if err != nil {
						return nil, errors.Wrapf(err, "Unable to open layer %s file", layerPath)
					}
					return file, nil
				},
			})

			// Add content digest to RootFS
			containerConfig = containerConfig.AddLayer(contentDigest)

			// Add to manifest
			manifest.Layers = append(manifest.Layers, docker.Layer{
				MediaType: "application/vnd.docker.image.rootfs.diff.tar.gzip",
				Size:      size,
				Digest:    digest,
			})
		}
	}

	cc, err := containerConfig.Create(buildConfig)
	if err != nil {
		return nil, err
	}

	// Create the container configuration layer
	containerConfigDigest := util.CalculateDigest(cc)
	layers = append(layers, Layer{
		Digest: containerConfigDigest,
		Size:   len(cc),
		Content: func(cxt context.Context) (io.ReadCloser, error) {
			return ioutil.NopCloser(bytes.NewBuffer(cc)), nil
		},
	})

	// Add container configuration to the manifest
	manifest.Config.Size = len(cc)
	manifest.Config.Digest = containerConfigDigest

	// Add base layers
	layers = append(layers, baseImageLayerProvider.Layers...)

	return &LayerProvider{
		Manifest:        manifest,
		ContainerConfig: containerConfig,
		Layers:          layers,
		BaseImage:       buildConfig.Image,
	}, nil
}

// Push layers and tags
func (l *LayerBuilder) Push(ctx context.Context, layers *LayerProvider, tag []string) error {

	// Push layers
	for _, layer := range layers.Layers {
		if layer.Content != nil {
			contentReader, err := layer.Content(ctx)
			if err != nil {
				return errors.Wrap(err, "Failed to read layer")
			}
			defer contentReader.Close()

			err = l.pushRegistry.PushLayer(ctx, contentReader, l.config.DockerSpec.OutputRepository, layer.Digest)
			if err != nil {
				return errors.Wrapf(err, "Failed to push layer %s", layer.Digest)
			}
		}
	}

	// Push tags
	for _, t := range tag {
		shortTag, err := util.FindOutputTagOrHash(t)
		if err != nil {
			return errors.Wrap(err, "Tag failed")
		}
		logrus.Infof("Push tag: %s", t)

		manifest, err := json.Marshal(layers.Manifest)
		if err != nil {
			return errors.Wrap(err, "Manfifest marshal failed")
		}
		err = l.pushRegistry.PushManifest(ctx, manifest, l.config.DockerSpec.OutputRepository, shortTag)
		if err != nil {
			return errors.Errorf("Failed to push manifest: %v", err)
		}
	}
	return nil
}
