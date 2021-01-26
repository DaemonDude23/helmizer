#!/bin/bash
set -e
TAG="${1}"
docker build \
    -t docker.pkg.github.com/chicken231/helmizer/helmizer:${TAG} \
    -t docker.pkg.github.com/chicken231/helmizer/helmizer:latest .

echo "Push to container registry?"
select yn in "Yes" "No"; do
    case $yn in
        Yes ) docker push docker.pkg.github.com/chicken231/helmizer/helmizer:${TAG};;
        No ) echo "No further action being taken"; exit;;
    esac
done
