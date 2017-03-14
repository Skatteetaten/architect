package nexus

import (
	"net/http"
	"os"
	"io"
)

type Nexus struct {
	BaseUrl    string
	ArtifactId string
	GroupId    string
	Version    string
	Type 	string
}

func (n Nexus) hentLeveransepakke() (string, error) {
	url := n.BaseUrl + "/service/local/artifact/maven/content?g=" + n.GroupId + "&a=" + n.ArtifactId + "&v=" + n.Version + "&e=" + n.Type + "&c=Leveransepakke&r=public-with-staging"

	fileLeveransepakke := "/tmp/" + n.ArtifactId + "-" + n.Version + "-Leveransepakke." + n.Type
	fileCreated, err := os.Create(fileLeveransepakke)
	if err != nil  {
		return "", err
	}
	defer fileCreated.Close()

	httpResponse, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer httpResponse.Body.Close()

	_, err = io.Copy(fileCreated, httpResponse.Body)
	if err != nil  {
		return "", err
	}

	return fileLeveransepakke, nil
}