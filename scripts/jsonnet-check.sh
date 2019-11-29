#!/usr/bin/env bash

# Check whether jsonnet and grafonnet-lib are installed
JSONNET_HELP_URL="https://github.com/google/jsonnet"
GRAFONNET_LIB_REPO="https://github.com/grafana/grafonnet-lib"

EXIT_CODE=0

which jsonnet >/dev/null
if [ $? -ne 0 ]; then
	echo "Jsonnet not found."
	echo "Please install Jsonnet and ensure 'jsonnet' is available in your PATH."
	echo "See ${JSONNET_HELP_URL} for installation instructions."
	echo
	EXIT_CODE=1
fi

if [ "$GRAFONNET_LIB" = "" ]; then
	echo "grafonnet-lib not found."
	echo "Please clone the ${GRAFONNET_LIB_REPO} repo and export local path of repo, i.e. export GRAFONNET_LIB=/my/path/to/grafonnet-lib"
	echo
	EXIT_CODE=1
fi

exit $EXIT_CODE
