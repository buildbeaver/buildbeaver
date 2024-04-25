#!/bin/bash
set -e
if [ -n "${BB_DEBUG}" ]; then
  set -x
fi

SCRIPT_DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
. "${SCRIPT_DIR}/../lib/node-env.sh"
check_deps "goimports"

cd "backend"
out="$(find . -type f -name '*.go' -not -path '*/vendor/*' -not -path '*/wire_gen.go' -exec goimports -d {} \;)"
if [ "${out}" != "" ]; then
  echo ""
  echo "Looks like you forgot to run 'goimports' before committing the following files:"
  echo "${out}"
  exit 1
fi
echo "No linting issues found"