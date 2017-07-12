package prepare

import (
	"github.com/skatteetaten/architect/pkg/java/config"
	"github.com/skatteetaten/architect/pkg/util"
)

var startscriptTemplate string = `source $HOME/architect/run_tools.sh
java {{.JvmOptions}} -cp "{{range $i, $value := .Classpath}}{{if $i}}:{{end}}{{$value}}{{end}}" $JAVA_OPTS {{.MainClass}} {{.ApplicationArgs}}
`

type Startscript struct {
	Classpath       []string
	JvmOptions      string
	MainClass       string
	ApplicationArgs string
}

func newStartScript(classpath []string, meta config.DeliverableMetadata) util.WriterFunc {
	var jvmOptions string
	var mainClass string
	var applicationArgs string
	if meta.Java != nil {
		jvmOptions = meta.Java.JvmOpts
		mainClass = meta.Java.MainClass
		applicationArgs = meta.Java.ApplicationArgs
	}

	return util.NewTemplateWriter(
		&Startscript{classpath, jvmOptions, mainClass, applicationArgs},
		"generatedStartScript",
		startscriptTemplate)
}
