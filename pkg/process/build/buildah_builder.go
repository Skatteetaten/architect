package process

import (
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/skatteetaten/architect/pkg/config/runtime"
	"github.com/skatteetaten/architect/pkg/docker"
	"os"
	"os/exec"
	"strconv"
)

type BuildahCmd struct {
	TlsVerify bool
}

func (b *BuildahCmd) Build(buildFolder string) (string, error) {
	context := buildFolder + "/Dockerfile"
	ruuid, err := uuid.NewUUID()
	if err != nil {
		return "", errors.Wrap(err, "UUID generation failed")
	}
	build := exec.Command("buildah", "--storage-driver", "vfs", "bud", "--quiet",
		"--tls-verify="+strconv.FormatBool(b.TlsVerify), "--isolation", "chroot", "-t", ruuid.String(),
		"-f", context, buildFolder)

	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	return ruuid.String(), build.Run()
}

func (b *BuildahCmd) Pull(image runtime.DockerImage) error {
	//Buildah dont require this method as long as we don't cache
	return nil
}

func (b *BuildahCmd) Push(ruuid string, tags []string, credentials *docker.RegistryCredentials) error {
	var err error
	var creds = ""
	if credentials != nil {
		creds = "--creds=" + credentials.Username + ":" + credentials.Password
	}
	for _, tag := range tags {
		var cmd *exec.Cmd
		if credentials != nil {
			cmd = exec.Command("buildah", "--storage-driver", "vfs", "push", "--quiet",
				"--tls-verify="+strconv.FormatBool(b.TlsVerify), creds, ruuid, tag)
		} else {
			cmd = exec.Command("buildah", "--storage-driver", "vfs", "push", "--quiet",
				"--tls-verify="+strconv.FormatBool(b.TlsVerify), ruuid, tag)
		}
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		err = errors.Wrapf(err, "Push of tag %s failed", tag)
	}
	return err
}

func (b *BuildahCmd) Tag(ruuid string, tag string) error {
	cmd := exec.Command("buildah", "--storage-driver", "vfs", "tag", ruuid, tag)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
