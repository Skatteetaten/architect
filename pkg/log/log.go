package log

import (
	golog "log"
	"os"
)

var Info *golog.Logger
var Error *golog.Logger
var Debug *golog.Logger

func init() {
	Debug = golog.New(os.Stdout,
		"TRACE: ", golog.Ldate|golog.Ltime|golog.Lshortfile)

	Info = golog.New(os.Stdout,
		"INFO: ", golog.Ldate|golog.Ltime|golog.Lshortfile)

	Error = golog.New(os.Stdout,
		"ERROR: ", golog.Ldate|golog.Ltime|golog.Lshortfile)
}
