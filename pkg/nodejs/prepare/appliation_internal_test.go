package prepare

import (
	"bytes"
	"encoding/json"
	"github.com/skatteetaten/architect/pkg/config"
	"github.com/skatteetaten/architect/pkg/config/runtime"
	"github.com/skatteetaten/architect/pkg/util"
	"github.com/stretchr/testify/assert"
	"path"
	"testing"
)

const buildTime = "2016-09-12T14:30:10Z"
const expectedNodeJsDockerFile = `FROM aurora/wrench:latest

LABEL maintainer="Oyvind <oyvind@dagobah.wars>" version="1.2.3"

COPY ./architectscripts /u01/architect

RUN chmod 755 /u01/architect/*

COPY ./package /u01/application

COPY ./overrides /u01/bin/

COPY ./package/app /u01/static/

COPY nginx.conf /etc/nginx/nginx.conf

RUN chmod 666 /etc/nginx/nginx.conf && \
    chmod 777 /etc/nginx && \
    chmod 755 /u01/bin/*

ENV APP_VERSION="1.2.3" AURORA_VERSION="1.2.3-b--baseimageversion" IMAGE_BUILD_TIME="2016-09-12T14:30:10Z" MAIN_JAVASCRIPT_FILE="/u01/application/test.json" PROXY_PASS_HOST="localhost" PROXY_PASS_PORT="9090" PUSH_EXTRA_TAGS="major"

WORKDIR "/u01/"

CMD ["/u01/architect/run", "/u01/bin/run_nginx"]`

const expectedNodeJsDockerFileWithoutNodeApp = `FROM aurora/wrench:latest

LABEL maintainer="Oyvind <oyvind@dagobah.wars>" version="1.2.3"

COPY ./architectscripts /u01/architect

RUN chmod 755 /u01/architect/*

COPY ./package /u01/application

COPY ./overrides /u01/bin/

COPY ./package/app /u01/static/

COPY nginx.conf /etc/nginx/nginx.conf

RUN chmod 666 /etc/nginx/nginx.conf && \
    chmod 777 /etc/nginx && \
    chmod 755 /u01/bin/*

ENV APP_VERSION="1.2.3" AURORA_VERSION="1.2.3-b--baseimageversion" IMAGE_BUILD_TIME="2016-09-12T14:30:10Z" MAIN_JAVASCRIPT_FILE="/u01/application/" PROXY_PASS_HOST="localhost" PROXY_PASS_PORT="9090" PUSH_EXTRA_TAGS="major"

WORKDIR "/u01/"

CMD ["/u01/architect/run", "/u01/bin/run_nginx"]`

const expectedNodeJsDockerFileWithExtras = `FROM aurora/wrench:latest

LABEL io.k8s.description="Demo application with React on Openshift." io.openshift.tags="openshift,react,nodejs" maintainer="Aurora OpenShift Utvikling <utvpaas@skatteetaten.no>" version="1.2.3"

COPY ./architectscripts /u01/architect

RUN chmod 755 /u01/architect/*

COPY ./package /u01/application

COPY ./overrides /u01/bin/

COPY ./package/app /u01/static/

COPY nginx.conf /etc/nginx/nginx.conf

RUN chmod 666 /etc/nginx/nginx.conf && \
    chmod 777 /etc/nginx && \
    chmod 755 /u01/bin/*

ENV APP_VERSION="1.2.3" AURORA_VERSION="1.2.3-b--baseimageversion" IMAGE_BUILD_TIME="2016-09-12T14:30:10Z" MAIN_JAVASCRIPT_FILE="/u01/application/api/server.js" PROXY_PASS_HOST="localhost" PROXY_PASS_PORT="9090" PUSH_EXTRA_TAGS="major"

WORKDIR "/u01/"

CMD ["/u01/architect/run", "/u01/bin/run_nginx"]`

const nginxConfPrefix = `
worker_processes  1;
error_log stderr;

events {
    worker_connections  1024;
}


http {
    include       /etc/nginx/mime.types;
    default_type  application/octet-stream;

    log_format  main  '$remote_addr - $remote_user [$time_local] "$request" '
                      '$status $body_bytes_sent "$http_referer" '
                      '"$http_user_agent" "$http_x_forwarded_for"';

    access_log  /dev/stdout;

    sendfile        on;
    #tcp_nopush     on;

    keepalive_timeout  65;

    gzip on;
    gzip_min_length 10240;
    gzip_vary on;


    index index.html;
`

const expectedNginxConfFileNoNodejsPartial = `
    server {
        listen 8080;

        location /api {
            return 404;
        }

        location / {
            root /u01/static;
            try_files $uri /index.html;
        }

    }
}
`
const expectedNginxConfFilePartial = `
    server {
        listen 8080;

        location /api {
            proxy_pass http://${PROXY_PASS_HOST}:${PROXY_PASS_PORT};
        }

        location / {
            root /u01/static;
            try_files $uri /index.html;
        }

    }
}
`

const expectedNginxConfFileSpaAndCustomHeaders = `
    server {
        listen 8080;

        location /api {
            proxy_pass http://${PROXY_PASS_HOST}:${PROXY_PASS_PORT};
        }

        location / {
            root /u01/static;
            try_files $uri /index.html;
            add_header X-Test-Header "Tulleheader";
            add_header X-Test-Header2 "Tulleheader2";
        }

    }
}
`
const expectedNginxConfFileNoSpaAndCustomHeaders = `
    server {
        listen 8080;

        location /api {
            proxy_pass http://${PROXY_PASS_HOST}:${PROXY_PASS_PORT};
        }

        location / {
            root /u01/static;
            add_header X-Test-Header "Tulleheader";
            add_header X-Test-Header2 "Tulleheader2";
        }

    }
}
`

const expectedNginxConfigWithOverrides = `
    server {
        listen 8080;

        location /api {
            proxy_pass http://${PROXY_PASS_HOST}:${PROXY_PASS_PORT};
            client_max_body_size 5m;
        }

        location / {
            root /u01/static;
            try_files $uri /index.html;
        }

    }
}
`

const nginxConfPrefixLocations = `
worker_processes  1;
error_log stderr;

events {
    worker_connections  1024;
}


http {
    include       /etc/nginx/mime.types;
    default_type  application/octet-stream;

    log_format  main  '$remote_addr - $remote_user [$time_local] "$request" '
                      '$status $body_bytes_sent "$http_referer" '
                      '"$http_user_agent" "$http_x_forwarded_for"';

    access_log  /dev/stdout;

    sendfile        on;
    #tcp_nopush     on;

    keepalive_timeout  65;

    gzip on;
    gzip_min_length 2048;
    gzip_vary on;


    index index.html;
`

const expectedNginxConfigWithLocations = `
    server {
        listen 8080;

        location /api {
            proxy_pass http://${PROXY_PASS_HOST}:${PROXY_PASS_PORT};
        }

        location / {
            root /u01/static;
            try_files $uri /index.html;
            add_header X-Some-Header "Verdi";
        }
        location /index.html {
            root /u01/static;
            gzip on;
            gzip_min_length 1024;
            gzip_vary on;
            add_header Cache-Control "no-cache";
            add_header X-Frame-Options "DENY";
            add_header X-XSS-Protection "1";
            try_files $uri /index.html;
        }
        location /index/other.html {
            root /u01/static;
            add_header Cache-Control "no-store";
            add_header X-XSS-Protection "1; mode=block";
            try_files $uri /index.html;
        }
        location /index_other.html {
            root /u01/static;
            gzip off;
            add_header Cache-Control "max-age=60";
            add_header X-XSS-Protection "0";
            try_files $uri /index.html;
        }

    }
}
`

var osJson = openshiftJson{
	Aurora: auroraApplication{
		NodeJS: &nodeJSApplication{
			Main: "test.json",
		},
		SPA:    true,
		Static: "app",
	},
	DockerMetadata: dockerMetadata{
		Maintainer: "Oyvind <oyvind@dagobah.wars>",
	},
}

func TestGeneratedFiledWhenNodeJSIsEnabled(t *testing.T) {
	files := make(map[string]string)
	dockerSpec := config.DockerSpec{
		PushExtraTags: config.ParseExtraTags("major"),
	}
	auroraVersion := runtime.NewAuroraVersion("1.2.3", false, "1.2.3", runtime.CompleteVersion("1.2.3-b--baseimageversion"))
	err := prepareImage(dockerSpec, &osJson, runtime.DockerImage{
		Tag:        "latest",
		Repository: "aurora/wrench",
	}, auroraVersion, testFileWriter(files), buildTime)
	assert.NoError(t, err)
	assert.Equal(t, expectedNodeJsDockerFile, files["Dockerfile"])
	assert.Equal(t, nginxConfPrefix+expectedNginxConfFilePartial, files["nginx.conf"])
	assert.NotEmpty(t, files["architectscripts/run"])
	assert.NotEmpty(t, files["architectscripts/run_tools.sh"])
	assert.NotEmpty(t, files["overrides/readiness_nginx.sh"])
	assert.NotEmpty(t, files["overrides/readiness_node.sh"])
	assert.NotEmpty(t, files["overrides/liveness_node.sh"])
	assert.NotEmpty(t, files["overrides/liveness_nginx.sh"])
	assert.Equal(t, len(files), 8)
}

func TestGeneratedFilesWhenNodeJSIsDisabled(t *testing.T) {
	files := make(map[string]string)
	dockerSpec := config.DockerSpec{
		PushExtraTags: config.ParseExtraTags("major"),
	}
	auroraVersion := runtime.NewAuroraVersion("1.2.3", false, "1.2.3", runtime.CompleteVersion("1.2.3-b--baseimageversion"))
	var openshiftJsonNoNodeJs = osJson
	openshiftJsonNoNodeJs.Aurora.NodeJS = nil
	err := prepareImage(dockerSpec, &openshiftJsonNoNodeJs, runtime.DockerImage{
		Tag:        "latest",
		Repository: "aurora/wrench",
	}, auroraVersion, testFileWriter(files), buildTime)
	assert.NoError(t, err)
	assert.Equal(t, expectedNodeJsDockerFileWithoutNodeApp, files["Dockerfile"])
	assert.Equal(t, nginxConfPrefix+expectedNginxConfFileNoNodejsPartial, files["nginx.conf"])
	assert.NotEmpty(t, files["overrides/run_node"])
	assert.Equal(t, len(files), 9)
}

func TestThatCustomHeadersIsPresentInNginxConfig(t *testing.T) {
	files := make(map[string]string)
	dockerSpec := config.DockerSpec{
		PushExtraTags: config.ParseExtraTags("major"),
	}
	json := osJson
	auroraVersion := runtime.NewAuroraVersion("1.2.3", false, "1.2.3", runtime.CompleteVersion("1.2.3-b--baseimageversion"))
	webapp := webApplication{
		DisableTryfiles: false,
		Headers: map[string]string{
			"X-Test-Header":  "Tulleheader",
			"X-Test-Header2": "Tulleheader2",
		},
		StaticContent: "pathTilStatic",
	}
	json.Aurora.Webapp = &webapp
	err := prepareImage(dockerSpec, &json, runtime.DockerImage{
		Tag:        "latest",
		Repository: "aurora/wrench",
	}, auroraVersion, testFileWriter(files), buildTime)
	assert.NoError(t, err)
	assert.Equal(t, nginxConfPrefix+expectedNginxConfFileSpaAndCustomHeaders, files["nginx.conf"])

	json.Aurora.Webapp.DisableTryfiles = true
	err = prepareImage(dockerSpec, &json, runtime.DockerImage{
		Tag:        "latest",
		Repository: "aurora/wrench",
	}, auroraVersion, testFileWriter(files), buildTime)
	assert.NoError(t, err)
	assert.Equal(t, nginxConfPrefix+expectedNginxConfFileNoSpaAndCustomHeaders, files["nginx.conf"])
}

func TestThatOverrideInNginxIsSet(t *testing.T) {
	files := make(map[string]string)
	dockerSpec := config.DockerSpec{
		PushExtraTags: config.ParseExtraTags("major"),
	}
	json := osJson
	auroraVersion := runtime.NewAuroraVersion("1.2.3", false, "1.2.3", runtime.CompleteVersion("1.2.3-b--baseimageversion"))
	json.Aurora.NodeJS.Overrides = map[string]string{
		"client_max_body_size": "5m",
	}
	err := prepareImage(dockerSpec, &json, runtime.DockerImage{
		Tag:        "latest",
		Repository: "aurora/wrench",
	}, auroraVersion, testFileWriter(files), buildTime)
	assert.NoError(t, err)
	assert.Equal(t, nginxConfPrefix+expectedNginxConfigWithOverrides, files["nginx.conf"])

}

const openshiftJsonJSONWithLocations = `
{
	"docker": {
	  "maintainer": "Aurora OpenShift Utvikling <utvpaas@skatteetaten.no>",
	  "labels": {
		 "io.k8s.description": "Demo application with React on Openshift.",
		 "io.openshift.tags": "openshift,react,nodejs"
	  }
	},
	"web": {
	  "nodejs": {
		 "assets": "api",
		 "main": "api/server.js",
		 "waf": "aurora-standard",
		 "runtime": "nodeLTS"
	  },
	  "webapp": {
		 "content": "app",
		 "gzip": {
			"use": "on",
			"min_length": 2048,
			"vary": "on"
		 },
		 "headers": {
			"X-Some-Header": "Verdi"
		 },
		 "locations": {
			"index.html": {
			  "headers": {
				 "Cache-Control": "no-cache",
				 "X-XSS-Protection": "1",
				 "X-Frame-Options": "DENY"
			  },
			  "gzip": {
				 "use": "on",
				 "min_length": 1024,
				 "vary": "on"
			  }
			},
			"index_other.html": {
			  "headers": {
				 "Cache-Control": "max-age=60",
				 "X-XSS-Protection": "0"
			  },
			  "gzip": {
				 "use": "off"
			  }
			},
			"index/other.html": {
				"headers": {
				   "Cache-Control": "no-store",
				   "X-XSS-Protection": "1; mode=block"
				}
			}
		 },
		 "static": "app"
	  }
	}
 } 
`

func TestThatCustomLocationsIsPresentInNginx(t *testing.T) {
	b := []byte(openshiftJsonJSONWithLocations)
	var m openshiftJson
	err := json.Unmarshal(b, &m)

	if err != nil {
		assert.Error(t, err)
	}

	assert.Equal(t, m.Aurora.NodeJS.Main, "api/server.js")

	files := make(map[string]string)
	dockerSpec := config.DockerSpec{
		PushExtraTags: config.ParseExtraTags("major"),
	}
	auroraVersion := runtime.NewAuroraVersion("1.2.3", false, "1.2.3", runtime.CompleteVersion("1.2.3-b--baseimageversion"))
	err = prepareImage(dockerSpec, &m, runtime.DockerImage{
		Tag:        "latest",
		Repository: "aurora/wrench",
	}, auroraVersion, testFileWriter(files), buildTime)
	assert.NoError(t, err)

	assert.Equal(t, expectedNodeJsDockerFileWithExtras, files["Dockerfile"])
	assert.Equal(t, nginxConfPrefixLocations+expectedNginxConfigWithLocations, files["nginx.conf"])
}

func testFileWriter(files map[string]string) util.FileWriter {
	return func(writer util.WriterFunc, filename ...string) error {
		buffer := new(bytes.Buffer)
		err := writer(buffer)
		if err == nil {
			files[path.Join(filename...)] = buffer.String()
		}
		return err
	}
}
