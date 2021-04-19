#!/bin/bash
# Run without sudo, but remember 'docker login' b4 running

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
BASE="$(dirname "$DIR")"

CMD="${BASE}/bin/architect"

${CMD} build bc -f "$DIR/testbcs/test.json"
${CMD} build bc -f "$DIR/testbcs/nodejs.json"
${CMD} build bc -f "$DIR/testbcs/tagwith.json"
${CMD} build bc -f "$DIR/testbcs/nodejs_tagwith.json"
${CMD} build bc -f "$DIR/testbcs/retag.json"
${CMD} build bc -f "$DIR/testbcs/nodejs_retag.json"

