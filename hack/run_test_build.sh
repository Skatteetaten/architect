#!/bin/bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
BASE="$(dirname "$DIR")"

CMD="${BASE}/bin/amd64/architect"

exec ${CMD} build -f "$DIR/test.json"  -v
