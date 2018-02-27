package prepare

import (
	"encoding/json"
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

func TestThatLegacyFormatIsMappedCorrect(t *testing.T) {
	oldOpenShiftJson := openshiftJson{}
	newOpenShiftJson := openshiftJson{}
	oldOpenShiftJson.Aurora.SPA = true
	assert.NoError(t, json.Unmarshal([]byte(OPENSHIFT_JSON_NEW_FORMAT), &newOpenShiftJson))
	assert.NoError(t, json.Unmarshal([]byte(OPENSHIFT_JSON_LEGACY_FORMAT), &oldOpenShiftJson))
	tiOld := mapOpenShiftJsonToTemplateInput(&oldOpenShiftJson, "name", "name", "version")
	tiNew := mapOpenShiftJsonToTemplateInput(&newOpenShiftJson, "name", "name", "version")
	assert.EqualValues(t, tiNew, tiOld)
	assert.Equal(t, "/", tiNew.Path)
	assert.Equal(t, "build", tiNew.Static)
	assert.Equal(t, true, tiNew.SPA)
}
