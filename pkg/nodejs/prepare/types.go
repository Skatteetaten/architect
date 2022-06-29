package prepare

import "github.com/skatteetaten/architect/v2/pkg/config/runtime"

// We copy this over the script in wrench if we don't have a nodejs app
const blockingRunNodeJS string = `#!/bin/sh
echo "Use of node.js was not configured in openshift.json. Blocking run script."
while true; do sleep 100d; done;
`

const readinessLivenessSH = `#!/bin/sh
{{if .Include}}
wget --spider localhost:{{.Port}} > /dev/null 2>&1
{{end}}
`

type auroraApplication struct {
	NodeJS *nodeJSApplication `json:"nodejs"`
	//Deprecated
	Static            string                 `json:"static"`
	Webapp            *webApplication        `json:"webapp"`
	ConfigurableProxy bool                   `json:"configurableProxy"`
	Gzip              nginxGzip              `json:"gzip"`
	Locations         map[string]interface{} `json:"locations"`
	Exclude           []string               `json:"exclude"`
	//Deprecated
	Path string `json:"path"`
	//Deprecated
	SPA bool `json:"spa"`
}

type webApplication struct {
	StaticContent   string            `json:"content"`
	Path            string            `json:"path"`
	Headers         map[string]string `json:"headers"`
	DisableTryfiles bool              `json:"disableTryfiles"`
}

type nodeJSApplication struct {
	Main      string            `json:"main"`
	Overrides map[string]string `json:"overrides"`
}

type openshiftJSON struct {
	Aurora         auroraApplication `json:"web"`
	DockerMetadata dockerMetadata    `json:"docker"`
}

type dockerMetadata struct {
	Maintainer string            `json:"maintainer"`
	Labels     map[string]string `json:"labels"`
}

type PreparedImage struct {
	baseImage runtime.DockerImage
	Path      string
}

type probe struct {
	Include bool
	Port    int
}

type nginxLocations map[string]*nginxLocation

type nginxLocation struct {
	Headers map[string]string `json:"headers"`
	Gzip    nginxGzip         `json:"gzip"`
}

type nginxGzip struct {
	UseStatic string `json:"use_static"`
}

// ImageMetadata utility struct containing image metadata
type ImageMetadata struct {
	Main             string
	Maintainer       string
	Baseimage        string
	PackageDirectory string
	Static           string
	Path             string
	Labels           map[string]string
	Env              map[string]string
}
