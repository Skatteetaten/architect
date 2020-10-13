#!/bin/bash
# Run without sudo, but remember 'docker login' b4 running

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
BASE="$(dirname "$DIR")"

CMD="${BASE}/bin/architect"

#dlv exec ${CMD} -- build -f "$DIR/test.json"  -v 
#${CMD} build bc -f "$DIR/testbcs/test.json"
${CMD} build bc -f "$DIR/testbcs/nodejs.json"
#${CMD} build bc -f "$DIR/testbcs/tagwith.json"
#${CMD} build bc -f "$DIR/testbcs/retag.json"
#${CMD} build bc -f "$DIR/testbcs/nodejs_tagwith.json"
#${CMD} build bc -f "$DIR/testbcs/nodejs_retag.json"
#${CMD} build bc -f "$DIR/testbcs/nodejs_tagwith_snapshot.json"
#${CMD} build bc -f "$DIR/testbcs/nodejs_retag_snapshot.json"
#${CMD} build bc -f "$DIR/testbcs/tagwith_snapshot.json"
#${CMD} build bc -f "$DIR/testbcs/retag_snapshot.json"
