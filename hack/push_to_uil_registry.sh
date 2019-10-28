#!/bin/bash

TARGET=container-registry-internal.aurora.skead.no/auroratest/architect:bleedingedge
TMPFOLDER=$(mktemp -d --suffix "architectimagetest")

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
BASE="$(dirname "$DIR")"

CMD="${BASE}/bin/amd64/architect"

cp $CMD $TMPFOLDER/main

cat << EOF > $TMPFOLDER/Dockerfile
FROM alpine:latest
ADD main /u01/bin/main
CMD ["/u01/bin/main"]
EOF

docker build -t $TARGET $TMPFOLDER && docker push $TARGET && rm -r $TMPFOLDER
