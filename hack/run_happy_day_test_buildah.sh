#!/bin/bash
# NB: Needs to be run with sudo"

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
BASE="$(dirname "$DIR")"

CMD="${BASE}/bin/amd64/architect"

#dlv exec ${CMD} -- build -f "$DIR/test.json"  -v
${CMD} build -f "$DIR/testbcs/test-buildah.json"
${CMD} build -f "$DIR/testbcs/tagwith-buildah.json"
${CMD} build -f "$DIR/testbcs/retag-buildah.json"
${CMD} build -f "$DIR/testbcs/nodejs_tagwith-buildah.json"
${CMD} build -f "$DIR/testbcs/nodejs_retag-buildah.json"
${CMD} build -f "$DIR/testbcs/nodejs_tagwith_snapshot-buildah.json"
${CMD} build -f "$DIR/testbcs/nodejs_retag_snapshot-buildah.json"
${CMD} build -f "$DIR/testbcs/tagwith_snapshot-buildah.json"
${CMD} build -f "$DIR/testbcs/retag_snapshot-buildah.json"
