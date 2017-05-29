package prepare_test

import (
	"bytes"
	"github.com/skatteetaten/architect/pkg/java/prepare"
	"strings"
	"testing"
	"fmt"
)

func TestBuild(t *testing.T) {
	var buf bytes.Buffer

	dockerfile, err := prepare.NewDockerfile(meta, buildinfo)

	if err != nil {
		t.Fatal(err)
	}

	if err := dockerfile.Write(&buf); err != nil {
		t.Fatal(err)
	}

	output := buf.String()

	assertContainsElement(t, output, fmt.Sprintf("FROM %s:%s", buildinfo.BaseImage.Repository,
		buildinfo.BaseImage.Version))
	assertContainsElement(t, output, fmt.Sprintf("MAINTAINER %s",meta_maintainer))
	assertContainsElement(t, output, meta_k8sDescription)
	assertContainsElement(t, output, meta_openshiftTags)
	assertContainsElement(t, output, meta_readinessUrl)
}

func assertContainsElement(t *testing.T, target string, element string) {
	if strings.Contains(target, element) == false {
		t.Error("excpected", element, ", got", target)
	}
}
