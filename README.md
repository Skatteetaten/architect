```
_______  _______  _______          __________________ _______  _______ _________
(  ___  )(  ____ )(  ____ \|\     /|\__   __/\__   __/(  ____ \(  ____ \\__   __/
| (   ) || (    )|| (    \/| )   ( |   ) (      ) (   | (    \/| (    \/   ) (   
| (___) || (____)|| |      | (___) |   | |      | |   | (__    | |         | |   
|  ___  ||     __)| |      |  ___  |   | |      | |   |  __)   | |         | |   
| (   ) || (\ (   | |      | (   ) |   | |      | |   | (      | |         | |   
| )   ( || ) \ \__| (____/\| )   ( |___) (___   | |   | (____/\| (____/\   | |   
|/     \||/   \__/(_______/|/     \|\_______/   )_(   (_______/(_______/   )_(   
                                                                                 
```

# Introduction

Architect is a tool for building container images. It accepts a software deliverable as input, 
usually a Maven artifact containing a runnable application, and builds a container image using the 
deliverable as the main component of the build context.

Architect was designed with the OpenShift Custom Build Strategy in mind.
Thus, it will normally be embedded in a custom builder container. 

However, Architect can also run outside Docker and OpenShift, e.g. on a developer workstation.

The input deliverable must contain a runnable application and must meet certain requirements
regarding content and file structure. Architect is highly opinionated in this regard.

Architect will perform the following tasks: 
 
* Download and prepare the deliverable in order to use it as the 'context' for a container build.
* Create the application layer file structure.
* Create and push image layer blobs.
* Update and push the image manifest.
* Update and push the container the configuration layer blob.  
* Create a set of image tags.

# Concepts

## The deliverable

The deliverable is the main input to the image build task. It is identified by the Maven coordinates 
which is supplied to Architect as build configuration variables. 

A specially tailored base image is associated with every deliverable. 

### Java application

#### Base image

Base image name is ```aurora/wingnut8``` or ```aurora/wingnut11```

#### Content

This deliverable contains the following: 

* Java libraries
* Start script (optional)
* Liveness and readiness scripts (optional)
* Metadata file
* Application resources

#### Prepare for build

Architect creates several scripts and files during the prepare stage. 

* Default start script if not provided. The main Java class must be specified in the metadata file.
* Default liveness and readiness scripts
* Applications files are prepared and the layer file structure is created.
* Log folder is created with correct file permissions.

#### Metadata file

The metadata file, openshift.json, contains information required to prepare the application layer as well as the 
start script, liveness and readiness scripts.

### NodeJs Application

#### Base image

Base image name is ```aurora/wrench8```, ```aurora/wrench10```, or ```aurora/wrench12``` depending on NodeJS version.

#### Content

This deliverable contains the following 

* NodeJs bundle with the application  
* Metadata file
* liveness and readiness scripts (Optional)
* Static resources

#### Prepare for build
Architect creates several scripts and files during the prepare stage.

* The main Javascript file must be specified in the metadata file.
* Add/create liveness and readiness scripts
* Application files are extracted and the layer file structure is created.
* Log folder is created with correct permissions.

### Doozer Application

#### Baseimage

Doozer builds supports an arbitrary base image. 

#### Content
This deliverable contains the following
* Application bundle
* Metadata file

#### Prepare for build
Architect creates several files during the prepare stage

* Start script must be specified in the metadata file.
* Application is extracted to the location specified in the metadata file.
* Application files are prepared and the layer file structure is created.  
* Log folder is created with correct permissions.


## Deliverable version types

Architect will create a set of image tags derived from the deliverable version and the build configuration 
variables.  
 
This section outlines the relationship between deliverale version and the tags that Architect will create.  
The purpose and characteristics of each tag is described in the next section.
 
### Normal version

A normal version according to the semantic versioning specification has the form ```X.Y.Z```.
Refer to semver.org for details.
 
If the version is a normal version, then Architect will create a full set of tags including latest, 
semantic versioning tags, and Aurora version tag.
 
### Snapshot version

If the version contains the word SNAPSHOT which may appear anywhere in the version string, 
then it is considered to be snapshot version, f.ex ```2.1.0-SNAPSHOT```.
 
If the version is a snapshot version, then Architect will only create a SNAPSHOT tag in addition to
the Aurora version tag.
 
### Pre release versions and other versions

The version is neither a normal version or a snapshot version, f.ex ```2.1.0-ALPHA```. 
 
In this case Architect will only create the Aurora version tag.

## Output image name

## Image tags

Architect will create a number of tags depending on the use case, deliverable version and build variables.
 
* Aurora version tag
* latest tag
* Semantic versioning tags
* Temporary tag
* Snapshot tag
 
### Aurora version tag

The Aurora version tag will always be created when an image is built. It is derived from the deliverable
version, the version of Architect as well as the the base image name and version.
 
For example, suppose that the deliverable version is ```1.4.51```, the Architect version is ```2.2.3```
and the Java base image version is ```1.4.0```, then the resulting Aurora version will be 

```1.4.51-b2.2.3-oracle8-1.4.0```
 
### Latest tag

The ```latest``` tag will normally reference the image with the highest precedence.

By default, Architect will not overwrite an existing ```latest``` tag that references an image with 
an Aurora version with higher semantic precedence than the new image. 

The build variable ```EXTRA_TAGS``` can be used if Architect should not create the ```latest``` tag.
 
### Semantic versioning tags

The semantic versioning tags gives the user more fine grained control of the deployment of the image. 

An OpenShift image stream will be notified whenever the image referenced by an image tag is updated.
 
Semantic versioning tags allows the user to pin one or more segments of the version number, ```X.Y.Z```.
Users can also add metadata to versions with ```+``` suffix that complies with ```[0-9A-Za-z]+```. 
The metadata will also be used as filter when comparing versions, meaning ```1.0.0+somemeta``` differs from ```1.0.0+othermeta```.
 
* The major tag includes only the major version ```X```. By pinning the major version number, a new deployment 
will be triggered when either the minor or patch version is changed.
* The minor tag includes the minor version ```X.Y```. By pinning the minor version number, X.Y, a new deployment 
will be triggered when the patch version is changed.
* The patch version is the full semantic version number, ```X.Y.Z```.

By default, Architect will not overwrite existing semantic versioning tags that reference an image with 
an Aurora version that has higher semantic precedence than the new image. This behaviour may be
overriden with the build variable TAG_OVERWRITE.

The build variable ```EXTRA_TAGS``` can be used to specify what semantic versioning tags to create.
 
### Temporary tag

The temporary tag is specified with the variable ```TAG_WITH```. Architect will not create any other tags 
except for the Aurora version tags.
 
### Snapshot tag
 
The snapshot tag is equal to the artifact version, f.ex. ```feature_AOS_540_Add_logic-SNAPSHOT```.

# How to use it?

## Use cases

Architect supports three use cases - normal build, temporary build or retag a temporary build.
 
### Normal build

In this use case Architect will build a Docker image from a Maven artifact. The artifact may be a
snapshot or a released version.
 
### Temporary build

This use case is triggered by assigning a value to the variable ```TAG_WITH```. This value is used as a tag name. 
 
The difference between this use case and a normal build is: 
 
Architect will only create a temporary tag in addition to the Aurora version tag. 
Architect will derive a full set of tags, depending on the version type, that will be stored as Docker
environment variables.
 
### Retag image from temporary build

This use case assumes that a temporary build has already been performed. Architect will not perform a 
Docker build. 
 
The variable ```RETAG_WITH``` identifies a previously built image.

## Jenkins pipeline

Architect will typically be invoked from a Jenkins pipeline script by using the OpenShift client
```oc(1)```. 

This requires an existing OpenShift build configuration in the cluster.

## Local build

Run the Architect binary from the commandline. A file that contains the required build variables must
be specified, f.ex: 

```architect build -f test.json -v ```

## Build variables
 
* ARTIFACT_ID, GROUP_ID and VERSION - Identifies the Maven artifact.

* BASE_IMAGE_REGISTRY, DOCKER_BASE_NAME, DOCKER_BASE_VERSION - Architect will use this as the base image. 

* TAG_WITH - Indicates that Architect should perform a temporary build.

* RETAG_WITH - Indicates that Architect should retag the image from a temporary build.

* TAG_OVERWRITE - Normally, Architect will not overwrite existing semantic versioning tags from a previous 
build if the existing ones refers to an image which have a higher precedence than the new one. 
Setting this variable to true indicates that Architect should overwrite existing semantic versioning tags 
even if the existing ones have a higher precedence. 

* BUILDER_VERSION - Architect version.

* EXTRA_TAGS - Specify exacly which tags to create. For example by specifying ```EXTRA_TAGS="latest,major"```
the minor and patch tags will not be created.

# How to build Architect?

```
make # Build the application. Is is written to bin/<achitecture>
make test # Runs test, go vet and go fmt. Should be run before every checkin
```

# How to test architect ?

```
make test                        # unit tests
source ./hack/initCredentials.sh # Load nexus credentials 
./hack/run_happy_day_test.sh     # Run integration tests
```

## Dependecies

We use go modules. 

Dependencies are managed via `go.mod`. Remember to run `go mod tidy` after dependency update.

## Building
The build is orchestrated on Jenkins, with Jenkinsfile

