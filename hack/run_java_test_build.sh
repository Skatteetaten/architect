#!/bin/bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
BASE="$(dirname "$DIR")"

CMD="${BASE}/bin/amd64/architect"

#dlv exec ${CMD} -- build -f "$DIR/test.json"  -v
exec ${CMD} build -f "$DIR/test.json" -v
