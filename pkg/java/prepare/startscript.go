package prepare

import (
	"github.com/pkg/errors"
	"github.com/skatteetaten/architect/pkg/java/config"
	"io"
	"text/template"
)

var startscriptTemplate string = `source $HOME/architect/run_tools.sh
    java {{.JvmOptions}} ` +
	`-cp "{{range $i, $value := .Classpath}}{{if $i}}:{{end}}{{$value}}{{end}}" ` +
	`$JAVA_OPTS {{.MainClass}} {{.ApplicationArgs}}`

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
		return errors.Wrap(err, "Failed to parse start script template")
	}

	if err = tmpl.Execute(writer, startscript); err != nil {
		return errors.Wrap(err, "Failed to execute start script template")
	}

	return nil
}
