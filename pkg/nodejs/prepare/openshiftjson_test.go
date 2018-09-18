package prepare

import (
	"encoding/json"
	"github.com/skatteetaten/architect/pkg/config"
	"github.com/skatteetaten/architect/pkg/config/runtime"
	"github.com/stretchr/testify/assert"
	"testing"
)

const OPENSHIFT_JSON_NEW_FORMAT = `
{
    "web": {
        "webapp": { 
            "path": "/",
            "content": "build"
        },
        "nodejs": {
            "main": "api/server.js"
        }
    }
}`

const OPENSHIFT_JSON_LEGACY_FORMAT = `
{
    "web": {
        "spa": true,
        "path": "/",
        "static": "build",
        "nodejs": {
            "main": "api/server.js"
        }
    }
}`

const OPENSHIFT_JSON_NEW_FORMAT_SPA_NOT_SET = `
{
    "web": {
        "webapp": {
            "headers": {
                "X-Frame-Options": "sameorigin"
            },
            "path": "/",
            "content": "build", 
            "disableTryfiles": false
        },
        "nodejs": {
            "main": "api/server.js"
        }
    }
}`

func TestThatSpaDefaultToTrueInWebAppBlock(t *testing.T) {
	openshiftJson := openshiftJson{}
	assert.NoError(t, json.Unmarshal([]byte(OPENSHIFT_JSON_NEW_FORMAT), &openshiftJson))
	assert.NotNil(t, openshiftJson.Aurora.Webapp)
	assert.False(t, openshiftJson.Aurora.Webapp.DisableTryfiles)
}

func TestThatValuesAreSetAsExpected(t *testing.T) {
	openshiftJson := openshiftJson{}
	assert.NoError(t, json.Unmarshal([]byte(OPENSHIFT_JSON_NEW_FORMAT_SPA_NOT_SET), &openshiftJson))
	assert.NotNil(t, openshiftJson.Aurora.Webapp)
	assert.False(t, openshiftJson.Aurora.Webapp.DisableTryfiles)
	assert.Equal(t, openshiftJson.Aurora.Webapp.Headers["X-Frame-Options"], "sameorigin")
	assert.Equal(t, openshiftJson.Aurora.Webapp.Path, "/")
	assert.Equal(t, openshiftJson.Aurora.Webapp.StaticContent, "build")
}

func TestThatOverridesAreWhitelistedAndSetCorrectly(t *testing.T) {
	openshiftJson := openshiftJson{}
	assert.NoError(t, json.Unmarshal([]byte(OPENSHIFT_JSON_NEW_FORMAT), &openshiftJson))
	openshiftJson.Aurora.NodeJS.Overrides = map[string]string{
		"a_value_not_whitelisted": "value",
	}
	_, err := mapObject(&openshiftJson)
	assert.EqualError(t, err, "Config a_value_not_whitelisted is not allowed to override with Architect.")

	openshiftJson.Aurora.NodeJS.Overrides = map[string]string{
		"client_max_body_size": "51m",
	}
	_, err = mapObject(&openshiftJson)
	assert.EqualError(t, err, "Value on client_max_body_size should be on the form Nm where N is between 1 and 50")

	openshiftJson.Aurora.NodeJS.Overrides = map[string]string{
		"client_max_body_size": "50m",
	}
	_, err = mapObject(&openshiftJson)
	assert.NoError(t, err)
	openshiftJson.Aurora.NodeJS.Overrides = map[string]string{
		"client_max_body_size": "2m",
	}
	assert.NoError(t, err)
}

func TestThatLegacyFormatIsMappedCorrect(t *testing.T) {
	oldOpenShiftJson := openshiftJson{}
	newOpenShiftJson := openshiftJson{}
	oldOpenShiftJson.Aurora.SPA = true
	assert.NoError(t, json.Unmarshal([]byte(OPENSHIFT_JSON_NEW_FORMAT), &newOpenShiftJson))
	assert.NoError(t, json.Unmarshal([]byte(OPENSHIFT_JSON_LEGACY_FORMAT), &oldOpenShiftJson))
	tiOld, err := mapObject(&oldOpenShiftJson)
	assert.NoError(t, err)
	tiNew, err := mapObject(&newOpenShiftJson)
	assert.NoError(t, err)
	assert.EqualValues(t, tiNew, tiOld)
	assert.Equal(t, "/", tiNew.Path)
	assert.Equal(t, "build", tiNew.Static)
	assert.Equal(t, true, tiNew.SPA)
}

func mapObject(openshiftJson *openshiftJson) (*templateInput, error) {
	dockerSpec := config.DockerSpec{
		PushExtraTags: config.ParseExtraTags("major"),
	}
	return mapOpenShiftJsonToTemplateInput(dockerSpec, openshiftJson, "name", "name", runtime.NewAuroraVersion("version", false, "version", runtime.CompleteVersion("version-b--baseimageversion")))
}
