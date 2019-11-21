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
		json.Unmarshal([]byte(data), &x)
		x["type"] = "kind"
		d, _ := json.Marshal(x)
		t.send(string(d))
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
