# `onload-module` Container Image

The container image must be built using the:

* exact kernel version headers of target node(s), and the
* exact Onload version/revision deployed with the Onload Operator
  and/or Onload Device Plugin.

## Building within the cluster

This directory is not relevant to building within the cluster.

KMM can build `onload-module` container images automatically in-cluster for use
by Kubernetes Onload. In-cluster, KMM selects the `DTK_AUTO` container image by
itself. Only a `onload-source` container image and `Dockerfile` therefore needs
to be supplied to KMM (typically via the Onload Operator).

KMM can also perform builds against non-running RHCOS though the use of its
`PreflightValidation` CR.

## Building outside the cluster for OpenShift

When building outside the cluster, dependencies and their versioning are
gathered independently of KMM.

The [Makefile](./Makefile) attempts to derive them for you:

```sh
make onload-module-dtk OCP_VERSION=4.12.12 ONLOAD_SOURCE=docker.io/onload/onload-source:<onload-version>
```

The above requires:

* `make`
* `jq`
* `docker` or `podman-docker`
* `oc`

The `OCP_VERSION` variable is used to select the Red Hat Driver Toolkit (DTK)
image corresponding to your release of OCP. Each release of OCP has a
corresponding Red Hat CoreOS (RHCOS) kernel version associated with it, and
those kernel headers are embedded in the DTK image.

Alternatively, to avoid using `oc` command, supply the image directly with
`make ... DTK_AUTO=quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:...`.
To find your DTK image digest, use
`oc adm release info --image-for=driver-toolkit`
within the cluster or refer to your OCP client's `release.txt`.

---

(c) Copyright 2023 Advanced Micro Devices, Inc.
