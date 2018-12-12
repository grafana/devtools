#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

#IMAGES=$(make images)
IMAGES="grafana/devtools-github-archive-parser grafana/devtools-github-event-aggregator"
IMAGE_TAG=$(git rev-parse HEAD)

push_image() {
    local image="$1"
    echo "Pushing ${image}:${IMAGE_TAG}"
    docker push ${image}:${IMAGE_TAG}

    echo "Pushing ${image}:latest"
    docker push ${image}:latest
}

for image in ${IMAGES}; do
    if [[ "$image" == *"build"* ]]; then
        continue
    fi
    push_image "${image}" &
done

wait

