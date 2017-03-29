package prepare

import (
	"github.com/skatteetaten/architect/pkg/java/config"
	"io"
	"text/template"
)

var startscriptTemplate string = `exec java {{.JvmOptions}} $JAVA_PROPERTIES_ARGS ` +
	`-cp {{range $i, $value := .Classpath}}{{$value}}:{{end}} ` +
	`$JAVA_DEBUG_ARGS -javaagent:$JOLOKIA_PATH=host=0.0.0.0,port=8778,protocol=https $JAVA_OPTS {{.MainClass}} {{.ApplicationArgs}}`

type Startscript struct {
	Classpath       []string
	JvmOptions      string
	MainClass       string
	ApplicationArgs string
}

func NewStartscript(classpath []string, meta config.DeliverableMetadata) *Startscript {
	var jvmOptions string
	var mainClass string
	var applicationArgs string
	if meta.Java != nil {
		jvmOptions = meta.Java.JvmOpts
		mainClass = meta.Java.MainClass
		applicationArgs = meta.Java.ApplicationArgs
	}

	return &Startscript{classpath, jvmOptions, mainClass, applicationArgs}
}

func (startscript Startscript) Write(writer io.Writer) error {

	tmpl, err := template.New("startscript").Parse(startscriptTemplate)

	if err != nil {
		return err
	}

	return tmpl.Execute(writer, startscript)
}
