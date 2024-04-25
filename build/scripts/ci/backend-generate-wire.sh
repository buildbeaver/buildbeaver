#!/bin/bash
set -e
if [ -n "${BB_DEBUG}" ]; then
  set -x
fi

SCRIPT_DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
. "${SCRIPT_DIR}/../lib/go-env.sh"
check_deps "wire"

for wire_file in backend/*/app/wire.go backend/*/app/*/wire.go; do
  pushd "$(dirname "${wire_file}")"
    wire
  popd
done