package trace

import (
	"bytes"
	"fmt"
	"net/http"
)

func NewTracer(sporingsUrl string, context string) *Tracer {
	return &Tracer{
		url:     sporingsUrl,
		context: context,
		enabled: sporingsUrl != "" && context != "",
	}
}

type Tracer struct {
	url     string
	context string
	enabled bool
}

func (t *Tracer) AddImageMetadata(key string, data interface{}) {
	if t.enabled {
		t.send(fmt.Sprintf(`{"%s": %s}`, key, data))
	}
}

func (t *Tracer) send(jsonStr string) {
	uri := t.url + "/api/v1/trace/" + t.context

	if t.enabled {
		req, err := http.NewRequest("POST", uri, bytes.NewBuffer([]byte(jsonStr)))
		req.Header.Set("Content-Type", "application/json")
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
}
