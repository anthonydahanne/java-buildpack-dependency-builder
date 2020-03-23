#!/usr/bin/env bash

set -euo pipefail

SOURCE_DIRECTORY="java-14-release"
DESTINATION_DIRECTORY="images"

PATTERN="jdk-([0-9]+)\.?([0-9]+)?\.?([0-9]+)?\+([0-9]+)"

build() {
  pushd ${SOURCE_DIRECTORY} > /dev/null

    bash configure \
      --disable-warnings-as-errors \
      --with-cacerts-file=$(pwd)/$(ls ../cacerts-repository/*.jks) \
      --with-debug-level=release \
      --with-freetype-include=/usr/include/freetype2 \
      --with-freetype-lib=/usr/lib/x86_64-linux-gnu \
      --with-native-debug-symbols=none \
      --with-vendor-name="Pivotal Software Inc" \
      --with-vendor-url="https://pivotal.io" \
      --with-vendor-version-string="Pivotal" \
      --with-version-build=$(build_number) \
      --without-version-opt \
      --without-version-pre

    make product-images legacy-jre-image

  popd > /dev/null
}

build_number() {
  for TAG in $(hg log -r "." --template "{latesttag}\n" | tr ":" "\n"); do
    if [[ ${TAG} =~ ${PATTERN} ]]; then
      printf "%02.f" ${BASH_REMATCH[4]:-0}
      return
    fi
  done
}

package() {
  chmod -R a+r ${SOURCE_DIRECTORY}/build/linux-x86_64-server-release/images

  local identifier="$(version)-$(platform)"

  tar czvf ${DESTINATION_DIRECTORY}/openjdk-jre-${identifier}.tar.gz -C ${SOURCE_DIRECTORY}/build/linux-x86_64-server-release/images/jre .
  tar czvf ${DESTINATION_DIRECTORY}/openjdk-jdk-${identifier}.tar.gz -C ${SOURCE_DIRECTORY}/build/linux-x86_64-server-release/images/jdk .
  echo $(semver) >> ${DESTINATION_DIRECTORY}/version
}

platform() {
  lsb_release -cs
}

semver() {
  pushd ${SOURCE_DIRECTORY} > /dev/null

    for TAG in $(hg log -r "." --template "{latesttag}\n" | tr ":" "\n"); do
      if [[ ${TAG} =~ ${PATTERN} ]]; then
        echo "${BASH_REMATCH[1]:-0}.${BASH_REMATCH[2]:-0}.${BASH_REMATCH[3]:-0}-$(printf "%02.f" ${BASH_REMATCH[4]:-0})"
        return
      fi
    done

  popd > /dev/null
}

version() {
  pushd ${SOURCE_DIRECTORY} > /dev/null

    for TAG in $(hg log -r "." --template "{latesttag}\n" | tr ":" "\n"); do
      if [[ ${TAG} =~ ${PATTERN} ]]; then
        echo "${BASH_REMATCH[1]:-0}.${BASH_REMATCH[2]:-0}.${BASH_REMATCH[3]:-0}_$(printf "%02.f" ${BASH_REMATCH[4]:-0})"
        return
      fi
    done

  popd > /dev/null
}

build
package