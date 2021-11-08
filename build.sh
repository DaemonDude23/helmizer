#!/bin/bash

set -e

TAG="${1}"
#REGISTRY_URL="docker.pkg.github.com/DaeonDude23/helmizer/helmizer"
REGISTRY_URL="docker.k8s.home/daemondude23/helmizer/helmizer"

if [ $# -eq 0 ]; then
    echo "error: Tag required. Exiting"
    exit 1
fi

# docker
DOCKER_CREATE_DATE="$(date -u +'%Y-%m-%dT%H:%M:%SZ')"
printf "\n * Docker container label (timestamp): %s" "$DOCKER_CREATE_DATE"

docker build \
    -t ${REGISTRY_URL}:${TAG} \
    -t ${REGISTRY_URL}:latest . \
    --label "org.opencontainers.image.created=$DOCKER_CREATE_DATE" \
    --label "org.label-schema.build-date=$DOCKER_CREATE_DATE" \

echo "Push to container registry?"
select yn in "Yes" "No"; do
    case $yn in
        Yes ) docker push ${REGISTRY_URL}:${TAG}; docker push ${REGISTRY_URL}:latest; exit;;
        No ) echo "No further action being taken"; exit;;
    esac
done
