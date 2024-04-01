#!/bin/bash
set -eu

ARCH=$(uname -m)
# lowercase OS name just in case it's Darwin or smth
OS=$(uname -s | tr '[:upper:]' '[:lower:]')

# install pact go library
# must be -mod=readonly, because it will use vendor mode by default
# and will complain saying `module lookup disabled by -mod=vendor`
echo "Installing pact-go CLI in ${PACT_BIN_PATH}"
GOBIN=${PACT_BIN_PATH} go install -mod=readonly github.com/pact-foundation/pact-go/v2
echo "Using pact-go at ${PACT_BIN_PATH}/pact-go"

echo "Installing Pact FFI native library"
if [[ $OS = "linux" ]] || [[ $OS = "darwin" && $ARCH = "arm64" ]]; then
    echo "SUDO REQUIRED"
    echo "This script will install a native Pact library under system location,"
    echo "which is not writable by default."
    echo "Please enter your password when prompted."
    run_as="sudo"
else
    run_as=""
fi 
$run_as "${PACT_BIN_PATH}/pact-go" -l DEBUG install
