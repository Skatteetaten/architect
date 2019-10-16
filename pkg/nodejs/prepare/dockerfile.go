package prepare

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

const WRENCH_DOCKER_FILE string = `FROM {{.Baseimage}}

LABEL{{range $key, $value := .Labels}} {{$key}}="{{$value}}"{{end}}

COPY ./{{.PackageDirectory}} /u01/application

COPY ./overrides /u01/bin/

COPY ./{{.PackageDirectory}}/{{.Static}} /u01/static{{.Path}}

COPY nginx.conf /etc/nginx/nginx.conf

RUN chmod 666 /etc/nginx/nginx.conf && \
    chmod 777 /etc/nginx && \
    chmod 755 /u01/bin/*

ENV{{range $key, $value := .Env}} {{$key}}="{{$value}}"{{end}}

WORKDIR "/u01/"`

const WRENCH_RADISH_DOCKER_FILE string = `FROM {{.Baseimage}}

LABEL{{range $key, $value := .Labels}} {{$key}}="{{$value}}"{{end}}

COPY ./{{.PackageDirectory}} /u01/application

COPY ./overrides /u01/bin/

COPY nginx-radish.json $HOME/

COPY ./{{.PackageDirectory}}/{{.Static}} /u01/static{{.Path}}

RUN chmod 666 /etc/nginx/nginx.conf && \
    chmod 777 /etc/nginx && \
    chmod 755 /u01/bin/*

ENV{{range $key, $value := .Env}} {{$key}}="{{$value}}"{{end}}

WORKDIR "/u01/"`

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
