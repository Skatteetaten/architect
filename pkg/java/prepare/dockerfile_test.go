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

	if err := prepare.NewDockerfile(meta, buildinfo).Write(&buf); err != nil {
		t.Fatal(err)
	}

	dockerfile := buf.String()

	assertContainsElement(t, dockerfile, fmt.Sprintf("FROM %s:%s", buildinfo.BaseImage.Repository,
		buildinfo.BaseImage.Version))
	assertContainsElement(t, dockerfile, fmt.Sprintf("MAINTAINER %s",meta_maintainer))
	assertContainsElement(t, dockerfile, meta_k8sDescription)
	assertContainsElement(t, dockerfile, meta_openshiftTags)
	assertContainsElement(t, dockerfile, meta_readinessUrl)
}

func assertContainsElement(t *testing.T, target string, element string) {
	if strings.Contains(target, element) == false {
		t.Error("excpected", element, ", got", target)
	}
}
