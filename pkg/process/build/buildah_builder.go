package process

import (
	"context"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/skatteetaten/architect/pkg/docker"
	"os"
	"os/exec"
	"strconv"
)

type BuildahCmd struct {
	TlsVerify bool
}

func (b *BuildahCmd) Build(ctx context.Context, buildConfig docker.DockerBuildConfig) (*BuildOutput, error) {

	type responseAndError struct {
		uuid string
		err  error
	}
	result := make(chan responseAndError, 1)

	go func() {
		buildContext := buildConfig.BuildFolder + "/Dockerfile"
		ruuid, err := uuid.NewUUID()
		if err != nil {
			result <- responseAndError{"", errors.Wrap(err, "UUID generation failed")}
		}
		build := exec.Command("buildah", "--storage-driver", "vfs", "bud", "--quiet",
			"--tls-verify="+strconv.FormatBool(b.TlsVerify), "--isolation", "chroot", "-t", ruuid.String(),
			"-f", buildContext, buildConfig.BuildFolder)

		build.Stdout = os.Stdout
		build.Stderr = os.Stderr
		result <- responseAndError{ruuid.String(), build.Run()}
	}()

	select {
	case <-ctx.Done():
		<-result //Wait for function
		return nil, errors.Wrap(ctx.Err(), "Buildah push operation timed out")
	case r := <-result:
		return &BuildOutput{ImageId: r.uuid}, r.err
	}
}

func (b *BuildahCmd) Pull(ctx context.Context, buildConfig docker.DockerBuildConfig) error {
	//Buildah dont require this method as long as we don't cache
	return nil
}

func (b *BuildahCmd) Push(ctx context.Context, buildResult *BuildOutput, tags []string, credentials *docker.RegistryCredentials) error {
	imageid := buildResult.ImageId
	c := make(chan error)
	go func() {
		var err error
		var creds = ""
		if credentials != nil {
			creds = "--creds=" + credentials.Username + ":" + credentials.Password
		}
		for _, tag := range tags {
			var cmd *exec.Cmd
			if credentials != nil {
				cmd = exec.Command("buildah", "--storage-driver", "vfs", "push", "--quiet",
					"--tls-verify="+strconv.FormatBool(b.TlsVerify), creds, imageid, tag)
			} else {
				cmd = exec.Command("buildah", "--storage-driver", "vfs", "push", "--quiet",
					"--tls-verify="+strconv.FormatBool(b.TlsVerify), imageid, tag)
			}
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err = cmd.Run()
			err = errors.Wrapf(err, "Push of tag %s failed", tag)
		}
		c <- err
	}()
	select {
	case <-ctx.Done():
		<-c //Wait for function
		return errors.Wrap(ctx.Err(), "Buildah push operation timed out")
	case err := <-c:
		return err
	}
}

func (b *BuildahCmd) Tag(ctx context.Context, buildResult *BuildOutput, tag string) error {
	c := make(chan error, 1)
	imageid := buildResult.ImageId
	go func() {
		cmd := exec.Command("buildah", "--storage-driver", "vfs", "tag", imageid, tag)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		c <- cmd.Run()
	}()

	select {
	case <-ctx.Done():
		<-c //Wait for function
		return errors.Wrap(ctx.Err(), "Buildah tag operation timed out")
	case err := <-c:
		return err
	}
}
