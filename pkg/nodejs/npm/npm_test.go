package npm_test

import (
	"github.com/Sirupsen/logrus"
	"github.com/skatteetaten/architect/pkg/nodejs/npm"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRegistryClient_DownloadPackageJson(t *testing.T) {
	d := npm.RegistryClient{
		RegistryUrl: "http://aurora/npm/repository/npm-internal",
	}
	p, err := d.DownloadPackageJson("openshift-referanse-react")

	logrus.Infof("P %v", p)
	s, err := d.DownloadTarball(p.Versions["0.1.2"].Dist.Tarball)

	logrus.Info(p.Versions["0.1.2"].Aurora)
	if err != nil {
		logrus.Errorf("%v", err)
	}
	logrus.Infof("%s", s)
}

func TestLocalClient(t *testing.T) {
	c := npm.NewLocalRegistry("testfiles")
	p, err := c.DownloadPackageJson("openshift-referanse-react")
	assert.NoError(t, err)

	version := p.Versions["0.1.2"]
	_, err = c.DownloadTarball(version.Dist.Tarball)
	assert.NoError(t, err)

	aurora := version.Aurora
	assert.Equal(t, aurora.NodeJS.Main, "api/server.js")
}
