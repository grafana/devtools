#!/usr/bin/env bash

set -e

DS_TYPE=${1:-mysql}
DS_NAME=${2:-github_stats}
FROM=${3:-2014-12-31T23:00:00.000Z}
OUTPUT=${4:-"grafana-dashboards/build"}

echo "info Generating dashboards DS_TYPE=${DS_TYPE}, DS_NAME=${DS_NAME}, FROM=${FROM}, OUTPUT=${OUTPUT}"

mkdir -p ${OUTPUT}

jsonnet -J ${GRAFONNET_LIB} -m ${OUTPUT} grafana-dashboards/dashboards.jsonnet --tla-str dsType=${DS_TYPE} --tla-str dsName=${DS_NAME} --tla-str from=${FROM}
