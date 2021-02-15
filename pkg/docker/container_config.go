package docker

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"os"
)

//
type ContainerConfig struct {
	Architecture    string                `json:"architecture"`
	Config          DockerContainerConfig `json:"config"`
	Container       string                `json:"container"`
	ContainerConfig OCIContainerConfig    `json:"container_config"`
	Created         string                `json:"created"`
	History         []History             `json:"history"`
	Os              string                `json:"os"`
	RootFs          RootFs                `json:"rootfs"`
}

type DockerContainerConfig struct {
	Hostname     string
	DomainName   string
	User         string
	AttachStdin  bool
	AttachStdout bool
	AttachStderr bool
	Tty          bool
	OpenStdin    bool
	StdinOnce    bool
	Env          []string
	Cmd          []string
	ArgsEscaped  bool
	Image        string
	Volumes      interface{}
	WorkingDir   string
	EntryPoint   []string
	OnBuild      []interface{}
	Labels       map[string]string
}

type OCIContainerConfig struct {
	Hostname     string
	Domainname   string
	User         string
	AttachStdin  bool
	AttachStdout bool
	AttachStderr bool
	Tty          bool
	OpenStdin    bool
	StdinOnce    bool
	Env          []string
	Cmd          []string
	ArgsEscaped  bool
	Image        string
	Volumes      interface{}
	WorkingDir   string
	Entrypoint   []string
	OnBuild      []interface{}
	Labels       map[string]string
}

type RootFs struct {
	Type    string   `json:"type,omitempty"`
	DiffIds []string `json:"diff_ids,omitempty"`
}

type History struct {
	Created    string `json:"created,omitempty"`
	Comment    string `json:"comment,omitempty"`
	EmptyLayer bool   `json:"empty_layer,omitempty"`
}

//TODO: Implement me!
func (c *ContainerConfig) CleanCopy() *ContainerConfig {
	c.ContainerConfig = OCIContainerConfig{}
	c.History = nil

	return c
}

func (c *ContainerConfig) AddLayer(digest string) *ContainerConfig {
	c.RootFs.DiffIds = append(c.RootFs.DiffIds, digest)
	return c
}

func (c *ContainerConfig) Save(dstFolder string, filename string) error {
	containerConfigFile, err := os.Create(dstFolder + "/" + filename)
	if err != nil {
		return errors.Wrap(err, "Failed when creating container config tmp file")
	}
	defer containerConfigFile.Close()

	data, err := json.Marshal(c)
	if err != nil {
		return errors.Wrap(err, "Container config: Marshal operation failed")
	}

	_, err = containerConfigFile.Write(data)
	if err != nil {
		return errors.Wrap(err, "Unable to write container config")
	}
	return nil
}

func (c *ContainerConfig) addEnv(env map[string]string) {
	var envList []string
	for k, v := range env {
		envList = append(envList, fmt.Sprintf("%s=%s", k, v))
	}
	c.Config.Env = append(c.Config.Env, envList...)
}

func (c *ContainerConfig) addLabels(labels map[string]string) {
	for k, v := range labels {
		c.Config.Labels[k] = v
	}
}

func (c *ContainerConfig) setCmd(cmd []string) {
	c.Config.Cmd = cmd
}

func (c *ContainerConfig) Create(buildConfig BuildConfig) ([]byte, error) {
	//Set env, labels, and cmd
	c.addEnv(buildConfig.Env)
	c.addLabels(buildConfig.Labels)

	//Ensure that we dont override
	if buildConfig.Cmd != nil {
		c.setCmd(buildConfig.Cmd)
	}
	rawContainerConfig, err := json.Marshal(c)
	if err != nil {
		return nil, errors.Wrapf(err, "Container config marshal failed")
	}
	return rawContainerConfig, nil
}
