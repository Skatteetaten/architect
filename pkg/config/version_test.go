package config_test

import (
	"testing"
	"github.com/skatteetaten/architect/pkg/config"
	"fmt"
	"github.com/docker/distribution/manifest/schema1"
)

type RegistryMock struct {}

func TestGetCompleteSNAPSHOT(t *testing.T) {
	r := config.NewFileConfigReader("../../testdata/build-SNAPSHOT.json")
	c, err := r.ReadConfig()
	if err != nil {
		t.Fatalf("Error when reading config: %s", err)
	}
	buildInfo, err := config.NewBuildInfo(RegistryMock{}, *c,
		config.Deliverable{"/tmp/tmppackage2323/meldingsproduksjon-mva-omsetningsoppgave-omvendt-avgiftsplikt-leveransepakke-bugfix-UIMVA-493-20170407.085342-2-Leveransepakke.zip"})

	fmt.Println(buildInfo.OutputImage.Version)
	fmt.Println(c.MavenGav.Version)

	if buildInfo.OutputImage.Version != "SNAPSHOT-meldingsproduksjon-mva-omsetningsoppgave-omvendt-avgiftsplikt-leveransepakke-bugfix-UIMVA-493-20170407.085342-2" {
		t.Error("SNAPSHOT name not correct")
	}
}

func (registry RegistryMock) GetManifest(repository string, tag string) (*schema1.SignedManifest, error) {
	return nil, nil // Do not need this
}

func (registry RegistryMock) GetManifestEnv(repository string, tag string, name string) (string, error) {
	return "1.2.3", nil
}
