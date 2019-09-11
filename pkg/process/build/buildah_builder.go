package process

import (
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/skatteetaten/architect/pkg/config/runtime"
	"github.com/skatteetaten/architect/pkg/docker"
	"os"
	"os/exec"
)

type BuildahCmd struct {

}

func (BuildahCmd) Build(buildFolder string) (string, error){
	context := buildFolder + "/Dockerfile"
	ruuid, err := uuid.NewUUID()
	if err != nil {
		return "", errors.Wrap(err, "UUID generation failed")
	}

	build := exec.Command("buildah", "--storage-driver", "vfs", "bud", "--isolation", "chroot", "-t", ruuid.String(), "-f", context, buildFolder)
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	return ruuid.String(), build.Run()
}

func (BuildahCmd) Pull(image runtime.DockerImage) error {
	//Buildah dont require this method
	return nil
}

func (BuildahCmd) Push(ruuid string , tags []string, credentials *docker.RegistryCredentials) error {
	var err error
	for _, tag := range tags {
		cmd := exec.Command("buildah", "--storage-driver", "vfs", "push", ruuid, tag)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		err = errors.Wrapf(err, "Push of tag %s failed", tag)
	}
	return err

}

func (BuildahCmd) Tag(ruuid string, tag string) error {
	cmd := exec.Command("buildah", "--storage-driver", "vfs", "tag", ruuid, tag)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
