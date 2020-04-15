#!/usr/bin/env bash

set -euo pipefail

cp jdk-11/bellsoft-jdk*.tar.gz images
cp jre-11/bellsoft-jre*.tar.gz images

PATTERN="([0-9]+)\.([0-9]+)\.([0-9]+)-([0-9]+)"
if [[ $(cat jdk-11/version) =~ ${PATTERN} ]]; then
  echo "${BASH_REMATCH[1]}.${BASH_REMATCH[2]}.${BASH_REMATCH[3]}_${BASH_REMATCH[4]}" > images/version
else
  echo "version is not semver" 1>&2
  exit 1
fi
