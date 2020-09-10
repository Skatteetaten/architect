package prepare

//TODO: Refactor

type DockerfileData struct {
	Main             string
	Maintainer       string
	Baseimage        string
	PackageDirectory string
	Static           string
	Path             string
	Labels           map[string]string
	Env              map[string]string
}

//We copy this over the script in wrench if we don't have a nodejs app
const BLOCKING_RUN_NODEJS string = `#!/bin/sh
echo "Use of node.js was not configured in openshift.json. Blocking run script."
while true; do sleep 100d; done;
`

const READINESS_LIVENESS_SH = `#!/bin/sh
{{if .Include}}
wget --spider localhost:{{.Port}} > /dev/null 2>&1
{{end}}
`
