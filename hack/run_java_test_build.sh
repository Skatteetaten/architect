#!/bin/bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
BASE="$(dirname "$DIR")"

CMD="${BASE}/bin/architect"

exec ${CMD} build bc -f "$DIR/testbcs/test.json"
