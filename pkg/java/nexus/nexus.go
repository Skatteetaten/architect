package nexus

import (
	"net/http"
	"os"
	"io"
	"mime"
	"io/ioutil"
	"path/filepath"
)

type Nexus struct {
	BaseUrl    string
	ArtifactId string
	GroupId    string
	Version    string
	Type 	string
}

func (n Nexus) downloadArtifactPackage() (string, error) {
	url := n.BaseUrl + "/service/local/artifact/maven/content?g=" + n.GroupId + "&a=" + n.ArtifactId + "&v=" + n.Version + "&e=" + n.Type + "&c=Leveransepakke&r=public-with-staging"

	httpResponse, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer httpResponse.Body.Close()

	_, params, err := mime.ParseMediaType(httpResponse.Header.Get("content-disposition"))
	if err != nil {
		return "", err
	}
	dir, err := ioutil.TempDir("", "package")
	if err != nil {
		return "", err
	}
	fileName := filepath.Join(dir, params["filename"])

	fileCreated, err := os.Create(fileName)
	if err != nil  {
		return "", err
	}
	defer fileCreated.Close()


	_, err = io.Copy(fileCreated, httpResponse.Body)
	if err != nil  {
		return "", err
	}

	return fileName, nil
}