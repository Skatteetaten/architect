#!/bin/bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
BASE="$(dirname "$DIR")"

CMD="${BASE}/bin/amd64/architect"

#dlv exec ${CMD} -- build -f "$DIR/test.json"  -v 
${CMD} build -f "$DIR/testbcs/test.json"
${CMD} build -f "$DIR/testbcs/nodejs.json"
${CMD} build -f "$DIR/testbcs/tagwith.json"
${CMD} build -f "$DIR/testbcs/retag.json"
