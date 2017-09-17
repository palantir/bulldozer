#!/usr/bin/env bash

##### Config #####

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
ROOT_DIR="$( cd "$SCRIPT_DIR/.." && pwd)"
CLIENT_DIR="${ROOT_DIR}/client"

##################

die () {
    echo >&2 "$1"
    exit 1
}

have_yarn() {
    hash yarn &> /dev/null
}

have_node6() {
    hash node &> /dev/null && [ $(node --version | sed 's/^v\([0-9]\).*/\1/') -ge 6 ]
}

check_prereqs() {
  have_node6 || die "Error: node 6.x or newer must be installed and on the PATH."
  have_yarn || die "Error: yarn must be installed and on the PATH."
}

install_and_bundle() {
  cd ${CLIENT_DIR}
  yarn install
  ./node_modules/.bin/webpack
}

##### Main #####

check_prereqs
install_and_bundle
echo "Installed dependencies and built client bundle"