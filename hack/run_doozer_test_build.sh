#!/bin/bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
BASE="$(dirname "$DIR")"

CMD="${BASE}/bin/amd64/architect"

#dlv exec ${CMD} -- build -f "$DIR/test.json"  -v
${CMD} build -f "$DIR/testbcs/testdoozer.json"
${CMD} build -f "$DIR/testbcs/testdoozer_with_tweezer.json"
${CMD} build -f "$DIR/testbcs/testdoozer_architect.json"
