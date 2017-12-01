package prepare

import (
	"bytes"
	"github.com/skatteetaten/architect/pkg/java/config"
	"github.com/stretchr/testify/assert"
	"testing"
)

const mainClass string = "foo.bar.Main"
const jvmOpts string = "-Dfoo=bar"
const applicationArgs string = "--logging.config=logback.xml"

var classpath []string = []string{"/app/lib/metrics.jar", "/app/lib/rt.jar", "/app/lib/spring.jar"}

var expectedStartScript = `source $HOME/architect/run_tools.sh
java -Dfoo=bar -cp "/app/lib/metrics.jar:/app/lib/rt.jar:/app/lib/spring.jar" $JAVA_OPTS foo.bar.Main --logging.config=logback.xml
`

var testMeta = &config.DeliverableMetadata{
	Docker: &config.MetadataDocker{},
	Java: &config.MetadataJava{
		MainClass:       mainClass,
		JvmOpts:         jvmOpts,
		ApplicationArgs: applicationArgs,
	},
	Openshift: &config.MetadataOpenShift{},
}

func TestWriteStartscript(t *testing.T) {

	writer := newStartScript(classpath, testMeta)
	buffer := new(bytes.Buffer)
	err := writer(buffer)
	assert.NoError(t, err)

	startscript := buffer.String()
	assert.Equal(t, startscript, expectedStartScript)
}
