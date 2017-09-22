#!/bin/bash

set -eu

docker login -u "${DOCKERHUB_USERNAME}" -p "${DOCKERHUB_PASSWORD}"
docker tag palantirtechnologies/bulldozer palantirtechnologies/bulldozer:$(./godelw project-version)
docker push palantirtechnologies/bulldozer