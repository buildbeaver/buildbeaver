#!/bin/bash
set -e
set -x

DIR=$(mktemp -d -t "bb-test-XXXXXXX")
cd "${DIR}"

if [ -z "$1" ]; then
  echo "Expected the first argument to be the path to the directory containing the buildbeaver build config"
  exit 1
fi

if [ ! -d "$1" ]; then
  echo "Expected the first argument to be the path to the directory containing the buildbeaver build config"
    exit 1
fi

if [ -z "$2" ]; then
  echo "Expected the second argument to be the bb command to execute"
  exit 1
fi

# Copy test pipeline config into the git repo
cp -R "${1}"/* "${DIR}/"

echo "Skeleton repo for testing bb" > README.md
git init
git add -A
git commit -m "Initial commit"

set +e
$2
exit_code=$?
set -e

echo "BB finished with exit code: $exit_code"
exit $exit_code