#!/bin/bash
set -ex
TAG="${1}"
docker build -t ghcr.io/chicken231/helmizer/helmizer:${TAG} -t ghcr.io/chicken231/helmizer/helmizer:latest .
docker push ghcr.io/chicken231/helmizer/helmizer:${TAG}
