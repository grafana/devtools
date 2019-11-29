#!/usr/bin/env bash

set -e

DS_TYPE=${1:-mysql}
DS_NAME=${2:-github_stats}
FROM=${3:-2014-12-31T23:00:00.000Z}

jsonnet -J ${GRAFONNET_LIB} -m grafana-dashboards/build grafana-dashboards/dashboards.jsonnet --tla-str dsType=${DS_TYPE} --tla-str dsName=${DS_NAME} --tla-str from=${FROM}
