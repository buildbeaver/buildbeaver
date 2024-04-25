#!/bin/bash
set -e
if [ -n "${BB_DEBUG}" ]; then
  set -x
fi

SCRIPT_DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
. "${SCRIPT_DIR}/../lib/node-env.sh"
check_deps "prettier"

cd "frontend"
out="$(prettier --list-different 'src/**/*.ts*' || :)"
if [ "${out}" != "" ]; then
  echo ""
  echo "Looks like you forgot to run 'yarn format' before committing the following files:"
  echo "${out}"
  exit 1
fi
echo "No linting issues found"