package docker

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestCreatedAndHistory(t *testing.T) {
	println("Running test")

	data, err := os.ReadFile("testdata/container_config.json")
	if err != nil {
		t.Error("Unable to read test data")
	}

	var config ContainerConfig
	err = json.Unmarshal(data, &config)
	if err != nil {
		t.Errorf("Unable to unmarshal test data")
	}

	wip := config.CleanCopy()

	old := wip.Created
	wip.setCreatedTimestamp()
	modified := wip.Created
	wip.addHistoryEntry()

	assert.NotEqual(t, modified, old)

	assert.Equal(t, "architect", wip.History[10].CreatedBy)
}
