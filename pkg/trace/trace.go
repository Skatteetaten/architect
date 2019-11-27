package trace

import (
	"bytes"
	"encoding/json"
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

func (t *Tracer) AddImageMetadata(kind string, data string) {
	if t.enabled {
		var x map[string]interface{}
		err := json.Unmarshal([]byte(data), &x)
		if err != nil {
			return
		}
		x["type"] = kind
		d, err := json.Marshal(x)
		if err != nil {
			return
		}
		t.send(string(d))
	}
}

func (t *Tracer) send(jsonStr string) {
	uri := t.url + "/api/v1/trace/" + t.context

	if t.enabled {
		req, err := http.NewRequest("POST", uri, bytes.NewBuffer([]byte(jsonStr)))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return
		}
		defer resp.Body.Close()
	}
}
