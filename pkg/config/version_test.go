package config_test

import (
	"testing"
	"github.com/skatteetaten/architect/pkg/config"
)

/*func TestGetMajor(t *testing.T) {
	version_test := []string{"2.3.1", "2"}
	version, err := getMajor(version_test[0])

	if err != nil {
		t.Error(err)
	}

	if version != version_test[1] {
		t.Errorf("Expexted %s, was %s", version_test[1], version)
	}
}

func TestExceptionGetMajor(t *testing.T) {
	_, err := getMajor("b2.3.1")

	if err == nil {
		t.Error("Expected mailformed exception")
	}
}

func TestGetMinor(t *testing.T) {
	version_test := []string{"11.34.1", "11.34"}
	version, err := getMinor(version_test[0])

	if err != nil {
		t.Error(err)
	}

	if version != version_test[1] {
		t.Errorf("Expexted %s, was %s", version_test[1], version)
	}
}

func TestExceptionGetMinor(t *testing.T) {
	_, err := getMinor("b5.3.1")

	if err == nil {
		t.Error("Expected mailformed exception")
	}
}
*/
func TestGetCompleteVersion(t *testing.T) {
	r := config.NewFileConfigReader("../../testdata/build.json")
	c, err := r.ReadConfig()
	if err != nil {
		t.Fatalf("Error when reading config: %s", err)
	}

	GetBaseImageVersion
}
