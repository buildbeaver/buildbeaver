#!/bin/bash
set -e
if [ -n "${BB_DEBUG}" ]; then
  set -x
fi

SCRIPT_DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
. "${SCRIPT_DIR}/env.sh"

NODE_PATH="${BUILD_DIR}/node/node_modules"
export NODE_PATH
export PATH="${PATH}:${NODE_PATH}/.bin"