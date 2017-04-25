package docker

import "testing"

func TestGetMajor(t *testing.T) {
	version_test := []string{"2.3.1", "2"}
	version, err := GetMajor(version_test[0])

	if err != nil {
		t.Error(err)
	}

	if version != version_test[1] {
		t.Errorf("Expexted %s, was %s", version_test[1], version)
	}
}

func TestExceptionGetMajor(t *testing.T) {
	_, err := GetMajor("b2.3.1")

	if err == nil {
		t.Error("Expected mailformed exception")
	}
}

func TestGetMinor(t *testing.T) {
	version_test := []string{"11.34.1", "11.34"}
	version, err := GetMinor(version_test[0])

	if err != nil {
		t.Error(err)
	}

	if version != version_test[1] {
		t.Errorf("Expexted %s, was %s", version_test[1], version)
	}
}

func TestExceptionGetMinor(t *testing.T) {
	_, err := GetMinor("b5.3.1")

	if err == nil {
		t.Error("Expected mailformed exception")
	}
}
