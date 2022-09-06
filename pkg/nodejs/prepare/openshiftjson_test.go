package prepare

import (
	"encoding/json"
	"testing"

	"github.com/skatteetaten/architect/v2/pkg/config"
	"github.com/skatteetaten/architect/v2/pkg/config/runtime"
	"github.com/stretchr/testify/assert"
)

const OpenshiftJSONNewFormat = `
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

const OpenshiftJSONLegacyFormat = `
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

const OpenshiftJSONNewFormatSpaNotSet = `
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

const openshiftJSONWithLocations = `
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
		"use_static": "on"
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
			 "use_static": "on"
		  }
		},
		"index_other.html": {
		  "headers": {
			 "Cache-Control": "max-age=60",
			 "X-XSS-Protection": "0"
		  },
		  "gzip": {
			 "use_static": "off"
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

const openshiftJSONWithNoLocations = `
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
			"use_static": "on",
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
	openshiftJSON := openshiftJSON{}
	assert.NoError(t, json.Unmarshal([]byte(OpenshiftJSONNewFormat), &openshiftJSON))
	assert.NotNil(t, openshiftJSON.Aurora.Webapp)
	assert.False(t, openshiftJSON.Aurora.Webapp.DisableTryfiles)
}

func TestThatValuesAreSetAsExpected(t *testing.T) {
	openshiftJSON := openshiftJSON{}
	assert.NoError(t, json.Unmarshal([]byte(OpenshiftJSONNewFormatSpaNotSet), &openshiftJSON))
	assert.NotNil(t, openshiftJSON.Aurora.Webapp)
	assert.False(t, openshiftJSON.Aurora.Webapp.DisableTryfiles)
	assert.Equal(t, openshiftJSON.Aurora.Webapp.Headers["X-Frame-Options"], "sameorigin")
	assert.Equal(t, openshiftJSON.Aurora.Webapp.Path, "/")
	assert.Equal(t, openshiftJSON.Aurora.Webapp.StaticContent, "build")
}

func TestThatExcludeIsSetCorrectly(t *testing.T) {
	openshiftJSON := openshiftJSON{}
	assert.NoError(t, json.Unmarshal([]byte(OpenshiftJSONNewFormat), &openshiftJSON))
	openshiftJSON.Aurora.Exclude = []string{
		"test/test1.swf",
		"test/test2.swf",
	}
	nginxConf, _, err := mapObject(&openshiftJSON)

	assert.NoError(t, err)
	assert.Equal(t, openshiftJSON.Aurora.Exclude, nginxConf.Exclude)
}

func TestThatExcludeRegExIsValid(t *testing.T) {
	openshiftJSON := openshiftJSON{}
	assert.NoError(t, json.Unmarshal([]byte(OpenshiftJSONNewFormat), &openshiftJSON))
	openshiftJSON.Aurora.Exclude = []string{
		"(.*myapp)/(.+\\.php)$",
		".+\\.(?<ext>.*)$",
		"~*.+\\.(.+)$",
	}
	_, _, err := mapObject(&openshiftJSON)
	assert.NoError(t, err)
}

func TestThatExcludeRegExIsInvalid(t *testing.T) {
	t.SkipNow()
	openshiftJSON := openshiftJSON{}
	assert.NoError(t, json.Unmarshal([]byte(OpenshiftJSONNewFormat), &openshiftJSON))
	openshiftJSON.Aurora.Exclude = []string{
		"(.mya*pp)/(+\\.php)$",
	}
	_, _, err := mapObject(&openshiftJSON)
	assert.Error(t, err)
}

func TestThatLegacyFormatIsMappedCorrect(t *testing.T) {
	oldOpenShiftJSON := openshiftJSON{}
	newOpenShiftJSON := openshiftJSON{}
	oldOpenShiftJSON.Aurora.SPA = true
	assert.NoError(t, json.Unmarshal([]byte(OpenshiftJSONNewFormat), &newOpenShiftJSON))
	assert.NoError(t, json.Unmarshal([]byte(OpenshiftJSONLegacyFormat), &oldOpenShiftJSON))

	nOld, dOld, err := mapObject(&oldOpenShiftJSON)
	assert.NoError(t, err)
	nNew, dNew, err := mapObject(&newOpenShiftJSON)
	assert.NoError(t, err)
	assert.EqualValues(t, dNew, dOld)
	assert.EqualValues(t, nNew, nOld)

	assert.Equal(t, "/", nNew.Path)
	assert.Equal(t, "build", dNew.Static)
	assert.Equal(t, true, nNew.SPA)
}

func TestThatRootGzipIsPresentInNginx(t *testing.T) {
	openshiftJSON := openshiftJSON{}
	assert.NoError(t, json.Unmarshal([]byte(openshiftJSONWithLocations), &openshiftJSON))
	nginxfileData, _, err := mapObject(&openshiftJSON)

	assert.NoError(t, err)
	assert.NotNil(t, nginxfileData.Gzip)
	assert.Equal(t, "on", nginxfileData.Gzip.UseStatic)
}

func TestThatCustomLocationsIsPresentInNginx(t *testing.T) {
	openshiftJSON := openshiftJSON{}
	assert.NoError(t, json.Unmarshal([]byte(openshiftJSONWithLocations), &openshiftJSON))
	nginxfileData, _, err := mapObject(&openshiftJSON)

	assert.NoError(t, err)
	assert.Equal(t, 3, len(nginxfileData.Locations))

	// Test index.html configuration
	assert.Equal(t, 3, len(nginxfileData.Locations["index.html"].Headers))
	assert.Equal(t, "no-cache", nginxfileData.Locations["index.html"].Headers["Cache-Control"])
	assert.Equal(t, "1", nginxfileData.Locations["index.html"].Headers["X-XSS-Protection"])
	assert.Equal(t, "DENY", nginxfileData.Locations["index.html"].Headers["X-Frame-Options"])

	assert.NotNil(t, nginxfileData.Locations["index.html"].Gzip)
	assert.Equal(t, "on", nginxfileData.Locations["index.html"].Gzip.UseStatic)

	// Test index_other.html configuration
	assert.Equal(t, 2, len(nginxfileData.Locations["index_other.html"].Headers))
	assert.Equal(t, "max-age=60", nginxfileData.Locations["index_other.html"].Headers["Cache-Control"])
	assert.Equal(t, "0", nginxfileData.Locations["index_other.html"].Headers["X-XSS-Protection"])

	assert.NotNil(t, nginxfileData.Locations["index_other.html"].Gzip)
	assert.Equal(t, "off", nginxfileData.Locations["index_other.html"].Gzip.UseStatic)

	// Test index/other.html configuration
	assert.Equal(t, 2, len(nginxfileData.Locations["index/other.html"].Headers))
	assert.Equal(t, "no-store", nginxfileData.Locations["index/other.html"].Headers["Cache-Control"])
	assert.Equal(t, "1; mode=block", nginxfileData.Locations["index/other.html"].Headers["X-XSS-Protection"])

	assert.NotNil(t, nginxfileData.Locations["index/other.html"].Gzip)

}

func TestThatNoCustomLocationsIsPresentInNginx(t *testing.T) {
	openshiftJSON := openshiftJSON{}
	assert.NoError(t, json.Unmarshal([]byte(openshiftJSONWithNoLocations), &openshiftJSON))
	nginxfileData, _, err := mapObject(&openshiftJSON)

	assert.NoError(t, err)
	assert.Equal(t, 0, len(nginxfileData.Locations))
}

func mapObject(openshiftJSON *openshiftJSON) (*nginxfileData, *ImageMetadata, error) {
	dockerSpec := config.DockerSpec{
		PushExtraTags: config.ParseExtraTags("major"),
	}
	return mapOpenShiftJSONToTemplateInput(dockerSpec, openshiftJSON, "name", "name", runtime.NewAuroraVersion("version", false, "version", "version-b--baseimageversion"))
}
