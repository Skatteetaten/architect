// Copyright © 2017 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"github.com/Sirupsen/logrus"
	"github.com/skatteetaten/architect/cmd"
	"github.com/skatteetaten/architect/cmd/architect"
	"github.com/skatteetaten/architect/pkg/config"
	"github.com/skatteetaten/architect/pkg/java/nexus"
	"github.com/skatteetaten/architect/pkg/nodejs/npm"
	"os"
	"strings"
	"github.com/skatteetaten/architect/pkg/docker"
	"github.com/skatteetaten/architect/pkg/util"
)

func main() {
	// We are called main. Assume we run in a container
	if strings.HasSuffix(os.Args[:1][0], "main") {
		initializeAndRunOnOpenShift()
	} else {
		cmd.Execute()
	}
}
func initializeAndRunOnOpenShift() {
	if len(os.Getenv("DEBUG")) > 0 {
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.InfoLevel)
	}
	mavenRepo := "http://aurora/nexus/service/local/artifact/maven/content"
	logrus.Debugf("Using Maven repo on %s", mavenRepo)
	// Read build config
	configReader := config.NewInClusterConfigReader()
	c, err := configReader.ReadConfig()
	if err != nil {
		logrus.Fatalf("Could not read configuration: %s", err)
	}

	cfg, err := configReader.ReadConfig()

	if err != nil {
		logrus.Fatalf("Error reading config: %s", err)
	}
	var npmDownloader npm.Downloader
	if c.BinaryBuild  && c.ApplicationType == config.NodeJsLeveransepakke {
		binaryInput, err := util.ExtractBinaryFromStdIn()
		if err != nil {
			logrus.Fatalf("Could not read binary input", err)
		}
		npmDownloader = npm.NewBinaryBuildRegistry(binaryInput, c.NodeJsApplication.Version)
	} else {
		npmDownloader = npm.NewRemoteRegistry("http://aurora/npm/repository/npm-internal")
	}
	runConfig := architect.RunConfiguration{
		Config:    cfg,
		NexusDownloader: nexus.NewNexusDownloader(mavenRepo),
		NpmDownloader:  npmDownloader,
		RegistryCredentialsFunc: docker.CusterRegistryCredentials(),
	}
	architect.RunArchitect(runConfig)
}
