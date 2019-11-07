package trace

import (
	"bytes"
	"github.com/Sirupsen/logrus"
	"net/http"
)

func NewTracer(sporingsUrl string, context string) *Tracer{
	return &Tracer{
		url: sporingsUrl,
		context: context,
	}

}

type Tracer struct {
	url string
	context string
}

func (t *Tracer ) AddImageMetadata(data interface{}) {
	logrus.Info("Sender til sporingslogger")
	t.send(`{"docker": "Hello from architect"}`)
}

func (t* Tracer) send(jsonStr string) {

	uri := t.url + "/api/v1/trace/" + t.context

	logrus.Info("uri ", uri)
	logrus.Info("Sending ", jsonStr)
	req, err := http.NewRequest("POST", uri, bytes.NewBuffer([]byte(jsonStr)))
	if err != nil {
		panic(err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

}




