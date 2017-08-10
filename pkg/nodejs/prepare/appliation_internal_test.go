package prepare

import (
	"bytes"
	"github.com/skatteetaten/architect/pkg/config/runtime"
	"github.com/skatteetaten/architect/pkg/nodejs/npm"
	"github.com/skatteetaten/architect/pkg/util"
	"github.com/stretchr/testify/assert"
	"testing"
)

const expectedNodeJsDockerFile = `FROM aurora/wrench:latest

LABEL maintainer="Oyvind <oyvind@dagobah.wars>" version="1.2.3"

COPY ./package /u01/app

ENV MAIN_JAVASCRIPT_FILE="/u01/app/test.json"

WORKDIR "/u01/app"

CMD ["/u01/bin/run"]`

const expectedNginxDockerFile = `FROM aurora/modem:latest

LABEL maintainer="Oyvind <oyvind@dagobah.wars>" version="1.2.3"

COPY ./package/app /u01/app/static

COPY nginx.conf /etc/nginx/nginx.conf
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
       root /u01/app/static;

       location /api {
          proxy_pass http://localhost:9090;
       }

    }
}
`

var testVersion = npm.VersionedPackageJson{
	Aurora: npm.AuroraApplication{
		NodeJS: npm.NodeJSApplication{
			Main: "test.json",
		},
		Static: "app",
	},
	Maintainers: []npm.Maintainer{
		{
			Name:  "Oyvind",
			Email: "oyvind@dagobah.wars",
		},
	},
}

func TestNodeJsDockerFiles(t *testing.T) {
	files := make(map[string]string)
	err := prepareNodeJsImage(&testVersion, runtime.BaseImage{
		Tag:        "latest",
		Repository: "aurora/wrench",
	}, "1.2.3", testFileWriter(files))
	assert.NoError(t, err)
	assert.Equal(t, files["Dockerfile"], expectedNodeJsDockerFile)
	assert.Equal(t, len(files), 1)
}

func TestNginxConfigurationAndDockerfile(t *testing.T) {
	files := make(map[string]string)
	err := prepareNginxImage(&testVersion, runtime.BaseImage{
		Tag:        "latest",
		Repository: "aurora/modem",
	}, "1.2.3", testFileWriter(files))
	assert.NoError(t, err)
	assert.Equal(t, files["Dockerfile"], expectedNginxDockerFile)
	assert.Equal(t, files["nginx.conf"], expectedNginxConfFile)
	assert.Equal(t, len(files), 2)
}

func testFileWriter(files map[string]string) util.FileWriter {
	return func(writer util.WriterFunc, filename string) error {
		buffer := new(bytes.Buffer)
		err := writer(buffer)
		if err == nil {
			files[filename] = buffer.String()
		}
		return err
	}
}
