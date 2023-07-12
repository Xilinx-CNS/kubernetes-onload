#! /bin/bash
# SPDX-License-Identifier: MIT
# SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.

me="$(basename "$0")"

err()  { echo >&2 "$*"; }
log()  { err "$me: $*"; }
fail() { log "$*"; exit 1; }
try()  { "$@" || fail "FAILED: $*"; }
tryquiet()  { "$@" >/dev/null || fail "FAILED: $*"; }

usage() {
    err
    err "usage:"
    err "  $me [options]"
    err
    err "options:"
    err "  --tarball <onload tarball>"
    err "  --kernel-version <version>"
    err "  --dtk <image>"
    err "  --ocp-version <version>"
    err "  --node-name <name>"
    err "  --authfile <filepath>"
    err "  --registry <name>"
    err "  --debug"
    err
    exit 1
}

function check_image_exists() {
    local image_name="$1"
    sudo podman image exists "$image_name"
    return $?
}

build_sfc_module() {
    local dockerfile="sfc-module.Dockerfile"
    local image_name="$REGHOST/openshift-kmm/sfc-module:$ONLOAD_VERSION-$NODE_KVER"
    if check_image_exists "$image_name"; then 
        echo "sfc-module image already exists"
        return
    fi

    echo "Building sfc-module image"
    sudo podman build --authfile="$AUTHFILE" --net=host \
    --no-cache \
    --build-arg KERNEL_VERSION="$NODE_KVER" \
    --build-arg DTK_AUTO="$DTK_IMAGE" \
    --build-arg ONLOAD_VERSION="$ONLOAD_TARBALL" \
    -t "$image_name" \
    -f $dockerfile .
}

build_onload_module() {
    local dockerfile="onload-module.Dockerfile"
    local image_name="$REGHOST/onload-clusterlocal/onload-module:$ONLOAD_VERSION-$NODE_KVER"
    if check_image_exists "$image_name"; then 
        echo "onload-module image already exists"
        return
    fi

    echo "Building onload-module image"
    sudo podman build --authfile="$AUTHFILE" --net=host \
    --no-cache \
    --build-arg KERNEL_VERSION="$NODE_KVER" \
    --build-arg DTK_AUTO="$DTK_IMAGE" \
    --build-arg ONLOAD_VERSION="$ONLOAD_TARBALL" \
    --build-arg ONLOAD_BUILD_PARAMS="$ONLOAD_BUILD_PARAMS" \
    -t "$image_name" \
    -f $dockerfile .
}

build_onload_userland() {
    local dockerfile="onload-user.Dockerfile"
    local image_name="$REGHOST/onload-clusterlocal/onload-user:$ONLOAD_VERSION"
    if check_image_exists "$image_name"; then 
        echo "onload-user image already exists"
        return
    fi

    echo "Building onload-user image"
    sudo podman build --authfile="$AUTHFILE" --net=host \
    --no-cache \
    --build-arg ONLOAD_VERSION="$ONLOAD_TARBALL" \
    --build-arg ONLOAD_BUILD_PARAMS="$ONLOAD_BUILD_PARAMS" \
    -t "$image_name" \
    -f $dockerfile .
}

build_onload_device_plugin() {
    local dockerfile="onload-device-plugin.Dockerfile"
    local image_name="$REGHOST/onload-clusterlocal/onload-device-plugin:$ONLOAD_VERSION-latest"
    if check_image_exists "$image_name"; then 
        echo "onload-device-plugin image already exists"
        return
    fi

    local onload_image_name="$REGHOST/onload-clusterlocal/onload-user"
    local onload_image_tag="$ONLOAD_VERSION"
    if ! check_image_exists "$onload_image_name:$onload_image_tag"; then
        echo "onload-user image doesn't exist. Building now"
        build_onload_userland
    fi

    echo "Building onload-device-plugin image"
    sudo podman build --authfile="$AUTHFILE" --net=host \
    --no-cache \
    --build-arg ONLOAD_IMAGE="$onload_image_name" \
    --build-arg IMAGE_TAG="$onload_image_tag" \
    -t "$image_name" \
    -f $dockerfile ../onload/deviceplugin
}

######################################################################
# main()

OCP_VERSION=""
DTK_IMAGE=""
DEFAULT_NODE_NAME="compute-0"
NODE_KVER=""
AUTHFILE=""
REGHOST=""
ONLOAD_TARBALL=""
ONLOAD_BUILD_PARAMS=""
ONLOAD_VERSION=""

while [ $# -gt 0 ]; do
  case "$1" in
    --tarball)        shift; ONLOAD_TARBALL="$1";;
    --kernel-version) shift; NODE_KVER="$1";;
    --dtk)            shift; DTK_IMAGE="$1";;
    --ocp-version)    shift; OCP_VERSION="$1";;
    --node-name)      shift; DEFAULT_NODE_NAME="$1";;
    --authfile)       shift; AUTHFILE="$1";;
    --registry)        shift; REGHOST="$1";;
    --debug)          ONLOAD_BUILD_PARAMS+=" --debug ";;
    -*)  usage;;
    *)   break;;
  esac
  shift
done
[ $# = 0 ] || usage

if [ -z "$DTK_IMAGE" ]; then
    if [ -z "$OCP_VERSION" ]; then
        err "Don't know what ocp version to target"
        exit 1
    fi
    if ! command -v oc &> /dev/null; then
        err "openshift client could not be found!"
        err "Cannot determine dtk image -- please specify DTK image to use " 
        err "with the flag '--dtk'"
        exit 1
    fi
    DTK_IMAGE=$(oc adm release info "$OCP_VERSION" --image-for=driver-toolkit)
fi

if [ -z "$NODE_KVER" ]; then
    if ! command -v oc &> /dev/null; then
        err "openshift client could not be found!"
        exit 1
    fi
    NODE_KVER=$(oc describe node "$DEFAULT_NODE_NAME" | grep 'Kernel Version' | awk '{print $3}')
fi

if [ -z "$REGHOST" ]; then
    err "--registry not specified, will attempt to get the internal registry"
    err "of the cluster with the oc client"
    if ! command -v oc &> /dev/null; then
        err "openshift client could not be found!"
        exit 1
    fi
    REGHOST=$(oc get route default-route -n openshift-image-registry --template='{{ .spec.host }}')
fi

if [ -z "$ONLOAD_TARBALL" ]; then
    err "Please specify onload tarball"
    usage
fi

ONLOAD_VERSION=$(basename "$ONLOAD_TARBALL" .tgz)

build_sfc_module

build_onload_module

build_onload_userland

build_onload_device_plugin
