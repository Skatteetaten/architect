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

const openshiftJsonJSONWithLocations = `
{
	"docker": {
	  "maintainer": "Aurora OpenShift Utvikling <utvpaas@skatteetaten.no>",
	  "labels": {
		 "io.k8s.description": "Demo application with React on Openshift.",
		 "io.openshift.tags": "openshift,react,nodejs"
	  }
	},
	"web": {
	  "nodejs": {
		 "assets": "api",
		 "main": "api/server.js",
		 "waf": "aurora-standard",
		 "runtime": "nodeLTS"
	  },
	  "webapp": {
		 "content": "app",		 
		 "static": "app"
	  },
	  "gzip": {
		"use": "on",
		"min_length": 2048,
		"vary": "on"
	 },
	 "headers": {
		"X-Some-Header": "Verdi"
	 },
	 "locations": {
		"index.html": {
		  "headers": {
			 "Cache-Control": "no-cache",
			 "X-XSS-Protection": "1",
			 "X-Frame-Options": "DENY"
		  },
		  "gzip": {
			 "use": "on",
			 "min_length": 1024,
			 "vary": "on"
		  }
		},
		"index_other.html": {
		  "headers": {
			 "Cache-Control": "max-age=60",
			 "X-XSS-Protection": "0"
		  },
		  "gzip": {
			 "use": "off"
		  }
		},
		"index/other.html": {
			"headers": {
			   "Cache-Control": "no-store",
			   "X-XSS-Protection": "1; mode=block"
			}
		}
	 }
	}
 } 
`

const openshiftJsonJSONWithNoLocations = `
{
	"docker": {
	  "maintainer": "Aurora OpenShift Utvikling <utvpaas@skatteetaten.no>",
	  "labels": {
		 "io.k8s.description": "Demo application with React on Openshift.",
		 "io.openshift.tags": "openshift,react,nodejs"
	  }
	},
	"web": {
	  "nodejs": {
		 "assets": "api",
		 "main": "api/server.js",
		 "waf": "aurora-standard",
		 "runtime": "nodeLTS"
	  },
	  "webapp": {
		 "content": "app",
		 "gzip": {
			"use": "on",
			"min_length": 2048,
			"vary": "on"
		 },
		 "headers": {
			"X-Some-Header": "Verdi"
		 },
		 "static": "app"
	  }
	}
 } 
`

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
	_, _, err := mapObject(&openshiftJson)
	assert.EqualError(t, err, "Config a_value_not_whitelisted is not allowed to override with Architect.")

	openshiftJson.Aurora.NodeJS.Overrides = map[string]string{
		"client_max_body_size": "51m",
	}
	_, _, err = mapObject(&openshiftJson)
	assert.EqualError(t, err, "Value on client_max_body_size should be on the form Nm where N is between 1 and 50")

	openshiftJson.Aurora.NodeJS.Overrides = map[string]string{
		"client_max_body_size": "50m",
	}
	_, _, err = mapObject(&openshiftJson)
	assert.NoError(t, err)
	openshiftJson.Aurora.NodeJS.Overrides = map[string]string{
		"client_max_body_size": "2m",
	}
	assert.NoError(t, err)
}

func TestThatExcludeIsSetCorrectly(t *testing.T) {
	openshiftJson := openshiftJson{}
	assert.NoError(t, json.Unmarshal([]byte(OPENSHIFT_JSON_NEW_FORMAT), &openshiftJson))
	openshiftJson.Aurora.Exclude = []string{
		"test/test1.swf",
		"test/test2.swf",
	}
	nginxConf, _, err := mapObject(&openshiftJson)

	assert.NoError(t, err)
	assert.Equal(t, openshiftJson.Aurora.Exclude, nginxConf.Exclude)
}

func TestThatExcludeRegExIsValid(t *testing.T) {
	openshiftJson := openshiftJson{}
	assert.NoError(t, json.Unmarshal([]byte(OPENSHIFT_JSON_NEW_FORMAT), &openshiftJson))
	openshiftJson.Aurora.Exclude = []string{
		"(.*myapp)/(.+\\.php)$",
		".+\\.(?<ext>.*)$",
		"~*.+\\.(.+)$",
	}
	_, _, err := mapObject(&openshiftJson)
	assert.NoError(t, err)
}

func TestThatExcludeRegExIsInvalid(t *testing.T) {
	t.SkipNow()
	openshiftJson := openshiftJson{}
	assert.NoError(t, json.Unmarshal([]byte(OPENSHIFT_JSON_NEW_FORMAT), &openshiftJson))
	openshiftJson.Aurora.Exclude = []string{
		"(.mya*pp)/(+\\.php)$",
	}
	_, _, err := mapObject(&openshiftJson)
	assert.Error(t, err)
}

func TestThatLegacyFormatIsMappedCorrect(t *testing.T) {
	oldOpenShiftJson := openshiftJson{}
	newOpenShiftJson := openshiftJson{}
	oldOpenShiftJson.Aurora.SPA = true
	assert.NoError(t, json.Unmarshal([]byte(OPENSHIFT_JSON_NEW_FORMAT), &newOpenShiftJson))
	assert.NoError(t, json.Unmarshal([]byte(OPENSHIFT_JSON_LEGACY_FORMAT), &oldOpenShiftJson))

	nOld, dOld, err := mapObject(&oldOpenShiftJson)
	assert.NoError(t, err)
	nNew, dNew, err := mapObject(&newOpenShiftJson)
	assert.NoError(t, err)
	assert.EqualValues(t, dNew, dOld)
	assert.EqualValues(t, nNew, nOld)

	assert.Equal(t, "/", nNew.Path)
	assert.Equal(t, "build", dNew.Static)
	assert.Equal(t, true, nNew.SPA)
}

func TestThatRootGzipIsPresentInNginx(t *testing.T) {
	openshiftJson := openshiftJson{}
	assert.NoError(t, json.Unmarshal([]byte(openshiftJsonJSONWithLocations), &openshiftJson))
	nginxfileData, _, err := mapObject(&openshiftJson)

	assert.NoError(t, err)
	assert.NotNil(t, nginxfileData.Gzip)
	assert.Equal(t, "on", nginxfileData.Gzip.Use)
	assert.Equal(t, 2048, nginxfileData.Gzip.MinLength)
	assert.Equal(t, "on", nginxfileData.Gzip.Vary)
	assert.Equal(t, "", nginxfileData.Gzip.Proxied)
	assert.Equal(t, "", nginxfileData.Gzip.Types)
	assert.Equal(t, "", nginxfileData.Gzip.Disable)
}

func TestThatCustomLocationsIsPresentInNginx(t *testing.T) {
	openshiftJson := openshiftJson{}
	assert.NoError(t, json.Unmarshal([]byte(openshiftJsonJSONWithLocations), &openshiftJson))
	nginxfileData, _, err := mapObject(&openshiftJson)

	assert.NoError(t, err)
	assert.Equal(t, 3, len(nginxfileData.Locations))

	// Test index.html configuration
	assert.Equal(t, 3, len(nginxfileData.Locations["index.html"].Headers))
	assert.Equal(t, "no-cache", nginxfileData.Locations["index.html"].Headers["Cache-Control"])
	assert.Equal(t, "1", nginxfileData.Locations["index.html"].Headers["X-XSS-Protection"])
	assert.Equal(t, "DENY", nginxfileData.Locations["index.html"].Headers["X-Frame-Options"])

	assert.NotNil(t, nginxfileData.Locations["index.html"].Gzip)
	assert.Equal(t, "on", nginxfileData.Locations["index.html"].Gzip.Use)
	assert.Equal(t, 1024, nginxfileData.Locations["index.html"].Gzip.MinLength)
	assert.Equal(t, "on", nginxfileData.Locations["index.html"].Gzip.Vary)
	assert.Equal(t, "", nginxfileData.Locations["index.html"].Gzip.Proxied)
	assert.Equal(t, "", nginxfileData.Locations["index.html"].Gzip.Types)
	assert.Equal(t, "", nginxfileData.Locations["index.html"].Gzip.Disable)

	// Test index_other.html configuration
	assert.Equal(t, 2, len(nginxfileData.Locations["index_other.html"].Headers))
	assert.Equal(t, "max-age=60", nginxfileData.Locations["index_other.html"].Headers["Cache-Control"])
	assert.Equal(t, "0", nginxfileData.Locations["index_other.html"].Headers["X-XSS-Protection"])

	assert.NotNil(t, nginxfileData.Locations["index_other.html"].Gzip)
	assert.Equal(t, "off", nginxfileData.Locations["index_other.html"].Gzip.Use)
	assert.Equal(t, 0, nginxfileData.Locations["index_other.html"].Gzip.MinLength)
	assert.Equal(t, "", nginxfileData.Locations["index_other.html"].Gzip.Vary)
	assert.Equal(t, "", nginxfileData.Locations["index_other.html"].Gzip.Proxied)
	assert.Equal(t, "", nginxfileData.Locations["index_other.html"].Gzip.Types)
	assert.Equal(t, "", nginxfileData.Locations["index_other.html"].Gzip.Disable)

	// Test index/other.html configuration
	assert.Equal(t, 2, len(nginxfileData.Locations["index/other.html"].Headers))
	assert.Equal(t, "no-store", nginxfileData.Locations["index/other.html"].Headers["Cache-Control"])
	assert.Equal(t, "1; mode=block", nginxfileData.Locations["index/other.html"].Headers["X-XSS-Protection"])

	assert.NotNil(t, nginxfileData.Locations["index/other.html"].Gzip)
	assert.Equal(t, "", nginxfileData.Locations["index/other.html"].Gzip.Use)
	assert.Equal(t, 0, nginxfileData.Locations["index/other.html"].Gzip.MinLength)
	assert.Equal(t, "", nginxfileData.Locations["index/other.html"].Gzip.Vary)
	assert.Equal(t, "", nginxfileData.Locations["index/other.html"].Gzip.Proxied)
	assert.Equal(t, "", nginxfileData.Locations["index/other.html"].Gzip.Types)
	assert.Equal(t, "", nginxfileData.Locations["index/other.html"].Gzip.Disable)
}

func TestThatNoCustomLocationsIsPresentInNginx(t *testing.T) {
	openshiftJson := openshiftJson{}
	assert.NoError(t, json.Unmarshal([]byte(openshiftJsonJSONWithNoLocations), &openshiftJson))
	nginxfileData, _, err := mapObject(&openshiftJson)

	assert.NoError(t, err)
	assert.Equal(t, 0, len(nginxfileData.Locations))
}

func mapObject(openshiftJson *openshiftJson) (*NginxfileData, *DockerfileData, error) {
	dockerSpec := config.DockerSpec{
		PushExtraTags: config.ParseExtraTags("major"),
	}
	return mapOpenShiftJsonToTemplateInput(dockerSpec, openshiftJson, "name", "name", runtime.NewAuroraVersion("version", false, "version", runtime.CompleteVersion("version-b--baseimageversion")))
}
