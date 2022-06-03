#!/bin/bash

set -e

make clean
make test-xml
make test-coverage
go test -short -coverprofile=bin/cov.out `go list ./... | grep -v vendor/`
make

#Create archive
ARCHIVEFOLDER="architect_app"
mkdir bin/$ARCHIVEFOLDER
cp bin/architect bin/$ARCHIVEFOLDER/architect
#Copy metadata if available
if test -f "metadata/openshift.json"; then
  mkdir bin/$ARCHIVEFOLDER/metadata
  cp metadata/openshift.json bin/$ARCHIVEFOLDER/metadata/openshift.json
else
  echo "NB: Openshift metadata not found"
fi
echo "Creating archive architect.zip:"
cd bin
zip -r architect.zip $ARCHIVEFOLDER
#cleanup
rm -rf $ARCHIVEFOLDER
