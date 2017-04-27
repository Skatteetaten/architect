package config_test

import (
	"testing"
	"github.com/skatteetaten/architect/pkg/config"
	"fmt"
)

func TestGetCompleteVersion(t *testing.T) {
	r := config.NewFileConfigReader("../../testdata/build-SNAPSHOT.json")
	c, err := r.ReadConfig()
	if err != nil {
		t.Fatalf("Error when reading config: %s", err)
	}
	buildInfo, err := config.NewBuildInfo(*c, "/tmp/tmppackage2323/meldingsproduksjon-mva-omsetningsoppgave-omvendt-avgiftsplikt-leveransepakke-bugfix-UIMVA-493-20170407.085342-2-Leveransepakke.zip")

	fmt.Println(buildInfo.OutputImage.Version)
	/*for s := range config.GetVersionTags(*buildInfo) {
		fmt.Println(s)
	}*/
	// the tests

}
