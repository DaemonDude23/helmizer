#!/bin/bash
set -ex
TAG="${1}"
docker build -t docker.pkg.github.com/chicken231/helmizer/helmizer:${TAG} -t docker.pkg.github.com/chicken231/helmizer/helmizer:latest .
docker push docker.pkg.github.com/chicken231/helmizer/helmizer:${TAG}
