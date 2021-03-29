#!/bin/bash
set -e

type glide 2> /dev/null || /bin/sh -c "export GOPATH=$GOROOT && curl -k https://glide.sh/get | sh"
type go-junit-report 2> /dev/null || go get -u github.com/jstemmer/go-junit-report
type gocov 2> /dev/null || go get github.com/axw/gocov/gocov
type gocov-xml 2> /dev/null || go get github.com/AlekSi/gocov-xml


GOPATH=$GOROOT glide install

export JUNIT_REPORT=TEST-junit.xml
export COBERTURA_REPORT=coverage.xml

# Go get is not the best way of installing.... :/
export PATH=$PATH:$HOME/go/bin

make clean 

#Create executable in /bin/amd64
make

#Run test and coverage
make test

#Create archive
ARCHIVEFOLDER="architect_app"
mkdir bin/amd64/$ARCHIVEFOLDER
cp bin/amd64/architect bin/amd64/$ARCHIVEFOLDER/architect
#Copy metadata if available
if test -f "metadata/openshift.json"; then
  mkdir bin/amd64/$ARCHIVEFOLDER/metadata
  cp metadata/openshift.json bin/amd64/$ARCHIVEFOLDER/metadata/openshift.json
else
  echo "NB: Openshift metadata not found"
fi
echo "Creating archive architect.zip:"
cd bin/amd64
zip -r architect.zip $ARCHIVEFOLDER
#cleanup
rm -rf $ARCHIVEFOLDER
