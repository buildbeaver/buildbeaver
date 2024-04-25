#!/bin/bash
set -e
if [ -n "${BB_DEBUG}" ]; then
  set -x
fi

###############################################################################
# Setup
###############################################################################
SCRIPT_DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
SRC_BASE="${SCRIPT_DIR}/../.."
VERSION=$("${SCRIPT_DIR}"/version-info.sh)

###############################################################################
# Default configuration
###############################################################################
REGION="us-west-2"
DISK_SIZE="20"
TAG=""
PLATFORM="amazon"
TYPE="runner"
PACKER_BUILDER=""

###############################################################################
# Option parsing
###############################################################################
print_usage () {
  echo "build-vm.sh - Builds a BuildBeaver VM"
  echo "-p <platform>        Valid options: 'amazon', 'qemu'"
  echo "-s <server_type>     Valid options: 'runner'"
  echo "-d <disk_size>       The size of the SSD disk in GB to provision with the VM. Defaults to $DISK_SIZE."
  echo "-r <region>          The Amazon region to build the VM in. Only used if platform is amazon."
  echo "-t <resource_prefix> A string to include in the name of the vm."
}

while getopts "s:r:t:p:d:" opt; do
  case $opt in
    p) PLATFORM=$OPTARG ;;
    s) TYPE=$OPTARG ;;
    d) DISK_SIZE=$OPTARG ;;
    r) REGION=$OPTARG ;;
    t) TAG=$OPTARG ;;
    \?)
      echo "Invalid option: -$OPTARG" >&2
      print_usage
      exit 1
      ;;
  esac
done

###############################################################################
# Functions
###############################################################################
configure_qemu() {
  local output_dir="${SRC_BASE}/build/output/packer/qemu/${TYPE}-${VERSION}"
  mkdir -p "${output_dir}"
  output_dir=$(realpath "${output_dir}")
  export PKR_VAR_qemu_output_directory="${output_dir}"
  export PKR_VAR_qemu_headless=false
}

configure_amazon() {
  case "$REGION" in
    'us-east-1') export PKR_VAR_aws_source_ami="ami-01d08089481510ba2" ;;
    'us-west-2') export PKR_VAR_aws_source_ami="ami-0e6dff8bde9a09539" ;;
    'ap-southeast-2') export PKR_VAR_aws_source_ami="ami-030a8d0e06463671c" ;;
    *) echo "Unsupported region $REGION" && print_usage && exit 1
  esac
  export PKR_VAR_aws_region="${REGION}"
}

configure_runner() {
  PACKER_FILE="${SRC_BASE}/build/packer/bb-runner.pkr.hcl"
  case "$PLATFORM" in
    'amazon')
      PACKER_BUILDER="amazon-ebs.bb-runner"
      export PKR_VAR_aws_root_disk_size="${DISK_SIZE}"
      ;;
    'qemu')
      PACKER_BUILDER="qemu.bb-runner"
      ;;
    *) echo "Unsupported platform $PLATFORM" && print_usage && exit 1
  esac
}

###############################################################################
# Execution
###############################################################################
case "$PLATFORM" in
  'amazon') configure_amazon;;
  'qemu')   configure_qemu;;
  *) echo "Unsupported platform $PLATFORM" && print_usage && exit 1
esac

case "$TYPE" in
  'runner') configure_runner ;;
  *) echo "Invalid server type '$TYPE'" && print_usage && exit 1
esac

export PKR_VAR_resource_prefix="${TAG}"
export PKR_VAR_buildbeaver_version="${VERSION}"
export PACKER_CACHE_DIR="${SRC_BASE}/build/output/packer/cache"

cd "${SRC_BASE}/build/packer" && packer build --force -only $PACKER_BUILDER "$(basename "$PACKER_FILE")"