package prepare

import (
	"bytes"
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

ENV MAIN_JAVASCRIPT_FILE="/u01/application/test.json" \
    IMAGE_BUILD_TIME="2016-09-12T14:30:10Z" \
    PROXY_PASS_HOST="localhost" \
    PROXY_PASS_PORT="9090"

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

ENV MAIN_JAVASCRIPT_FILE="/u01/application/" \
    IMAGE_BUILD_TIME="2016-09-12T14:30:10Z" \
    PROXY_PASS_HOST="localhost" \
    PROXY_PASS_PORT="9090"

WORKDIR "/u01/"

CMD ["/u01/architect/run", "/u01/bin/run_nginx"]`

const expectedNginxConfFileNoNodejs = `
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

    #gzip  on;

    index index.html;

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
const expectedNginxConfFile = `
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

    #gzip  on;

    index index.html;

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

var openshiftJson = OpenshiftJson{
	Aurora: AuroraApplication{
		NodeJS: &NodeJSApplication{
			Main: "test.json",
		},
		SPA:    true,
		Static: "app",
	},
	DockerMetadata: DockerMetadata{
		Maintainer: "Oyvind <oyvind@dagobah.wars>",
	},
}

var openshiftJsonNoNodeJs = openshiftJson

func init() {
	openshiftJsonNoNodeJs.Aurora.NodeJS = nil
}

func TestNodeJsDockerFiles(t *testing.T) {
	files := make(map[string]string)
	err := prepareImage(&openshiftJson, runtime.DockerImage{
		Tag:        "latest",
		Repository: "aurora/wrench",
	}, "1.2.3", testFileWriter(files), buildTime)
	assert.NoError(t, err)
	assert.Equal(t, expectedNodeJsDockerFile, files["Dockerfile"])
	assert.Equal(t, expectedNginxConfFile, files["nginx.conf"])
	assert.NotEmpty(t, files["architectscripts/run"])
	assert.NotEmpty(t, files["architectscripts/run_tools.sh"])
	assert.NotEmpty(t, files["overrides/readiness_nginx.sh"])
	assert.NotEmpty(t, files["overrides/readiness_node.sh"])
	assert.NotEmpty(t, files["overrides/liveness_node.sh"])
	assert.NotEmpty(t, files["overrides/liveness_nginx.sh"])
	assert.Equal(t, len(files), 8)
}

func TestDockerfileWithoutNodeJSApp(t *testing.T) {
	files := make(map[string]string)

	err := prepareImage(&openshiftJsonNoNodeJs, runtime.DockerImage{
		Tag:        "latest",
		Repository: "aurora/wrench",
	}, "1.2.3", testFileWriter(files), buildTime)
	assert.NoError(t, err)
	assert.Equal(t, expectedNodeJsDockerFileWithoutNodeApp, files["Dockerfile"])
	assert.NotEmpty(t, files["overrides/run_node"])
	assert.Equal(t, len(files), 9)
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
