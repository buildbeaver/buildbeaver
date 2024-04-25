#!/bin/bash

set -e

print_header() {
  echo
  echo "========================="
  echo -e "\033[0;32m$1\033[0m"
  echo "========================="
  echo
}

print_header "BB E2E runner script"

# Ensure we can run git commands within the mounted directory
git config --global --add safe.directory /development/buildbeaver

pushd /development/buildbeaver/test
# Activate the python venv for our tests.
print_header "Activating our Python virtual environment"
source venv/bin/activate

# Install our Python Core SDK
print_header "Installing BB Python Core SDK"
pip install ../sdk/core/python/client

# Install the rest of our requirements
print_header "Installing E2E pip requirements"
pip install -r requirements.txt

print_header "Running tests"
pytest $1
popd # /development/buildbeaver/test