package trace

import (
	"bytes"
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
	t.send(`{ "docker": "dunno" }`)
}

func (t* Tracer) send(jsonStr string) {

	uri := t.url + "/api/v1/trace/" + t.context

	req, err := http.NewRequest("POST", uri, bytes.NewBuffer([]byte(jsonStr)))
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

}




