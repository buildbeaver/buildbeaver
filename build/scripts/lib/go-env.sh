#!/bin/bash
set -e
if [ -n "${BB_DEBUG}" ]; then
  set -x
fi

SCRIPT_DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
. "${SCRIPT_DIR}/env.sh"

export GODIR="${BUILD_DIR}/go"
export GOBIN="${GODIR}/bin"
export GOCACHE="${GODIR}/cache"

mkdir -p "${GOCACHE}" "${GOBIN}"
