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
	"errors"
	"github.com/sirupsen/logrus"
	"github.com/skatteetaten/architect/v2/cmd"
	"github.com/skatteetaten/architect/v2/cmd/architect"
	"github.com/skatteetaten/architect/v2/pkg/config"
	"github.com/skatteetaten/architect/v2/pkg/docker"
	"github.com/skatteetaten/architect/v2/pkg/nexus"
	"github.com/skatteetaten/architect/v2/pkg/util"
	"os"
	"strings"
)

func main() {
	// We are called main. Assume we run in a container
	if strings.HasSuffix(os.Args[:1][0], "architect") && len(os.Args) == 1 {
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
	for _, env := range os.Environ() {
		logrus.Debugf("Environment %s", env)
	}

	// Read build config
	configReader := config.NewInClusterConfigReader()
	c, err := configReader.ReadConfig()
	if err != nil {
		logrus.Fatalf("Could not read configuration: %s", err)
	}

	if err != nil {
		logrus.Fatalf("Error reading config: %s", err)
	}

	var nexusDownloader nexus.Downloader
	if c.BinaryBuild {
		binaryInput, err := util.ExtractBinaryFromStdIn()
		if err != nil {
			logrus.Fatalf("Could not read binary input: %s", err)
		}
		nexusDownloader = nexus.NewBinaryDownloader(binaryInput)
	} else {
		nexusAccess, err := config.ReadNexusConfigFromFileSystem()
		if err != nil {
			logrus.Fatalf("Error reading NexusAccess, and build is not binary: %s", errors.Unwrap(err))
		}
		logrus.Debugf("Using Maven repo on %s", nexusAccess.NexusURL)
		nexusDownloader = nexus.NewMavenDownloader(nexusAccess.NexusURL, nexusAccess.Username, nexusAccess.Password)
	}
	runConfig := architect.RunConfiguration{
		Config:                  c,
		NexusDownloader:         nexusDownloader,
		RegistryCredentialsFunc: docker.ClusterRegistryCredentials(),
	}
	architect.RunArchitect(runConfig)
}
