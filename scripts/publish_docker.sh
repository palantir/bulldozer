#!/bin/bash

set -eu

docker login -u "${DOCKERHUB_USERNAME}" -p "${DOCKERHUB_PASSWORD}"

if git describe --exact-match --tags HEAD; then
    docker tag palantirtechnologies/bulldozer palantirtechnologies/bulldozer:$(./godelw project-version)
    docker push palantirtechnologies/bulldozer:$(./godelw project-version)
else
    docker tag palantirtechnologies/bulldozer palantirtechnologies/bulldozer:snapshot
    docker push palantirtechnologies/bulldozer:snapshot
fi
