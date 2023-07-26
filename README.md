# Onload Integration for Kubernetes and OpenShift

## Deployment guide

This section outlines the steps required to deploy Onload components in an OCP cluster in order to enable Onload acceleration in user pods.

Expected environment:
- (WIP) Onload/master
- (WIP) OCP 4.12 with:
  - Node Feature Discovery (NFD) Operator
  - Kernel Module Management (KMM) Operator ([Redhat docs](https://docs.openshift.com/container-platform/4.12/hardware_enablement/kmm-kernel-module-management.html), [codebase docs](https://kmm.sigs.k8s.io/documentation/deploy_kmod/))

#### Base images required

For Onload & SFC:
* ubi8:8.8
* ubi8-minimal:8.8
* golang:1.20.4
* `DTK_AUTO`

The `DTK_AUTO` image is determined by the KMM Operator depending on the kernel
versions of the nodes in your cluster. To get the location of this image you
can run:
```console
$ oc adm release info $OCP_VERSION --image-for=driver-toolkit
```
Where `$OCP_VERSION` is the version of the Openshift cluster on which you will
deploy Onload.


For Sfptpd:
* ubi9-minimal:9.2

For Sfnettest example pod:
* ubi8-init:8.8

#### Build sources

When following the build steps below to build from source in a cluster, ensure
the source location paths are correct for your environment. The default source
location paths are [onload](https://github.com/Xilinx-CNS/onload) and this
repository. This behaviour can be changed by editing
[onload-sources.conf](base/onload-sources.conf).

The format of the source files must be `*.tar.gz` as the following steps access
and download them using `ADD` commands in Dockerfiles.

### SFC out-of-tree driver

This section may be skipped without affecting the other deployment steps, however, the out-of-tree `sfc` driver is currently required when using the provided `onload` driver with a Solarflare card.

To dynamically load an out-of-tree `sfc` driver before accelerated workloads are started, follow [Deploy using KMM](#deploy-using-kmm). If newer driver features are required at boot time, also follow [Day-0 MachineConfig](#day-0-deployment-using-machineconfig).

Alternatively, an out-of-tree `sfc` driver may be deployed with a user-supported method beyond the scope of this document, such as a custom kernel build or in-house OS image. Note that network interface names can be fixed with UDEV rules -- on a RHCOS node within OpenShift, the directory `/etc/udev/rules.d/` can be written to with a `MachineConfig` CR.

#### Deploy using KMM

_The recommended way to install onload's `sfc` driver._

Applying the following manifest will cause the automated build and loading of the `sfc` driver. Specifically, the files in [sfc/kmm/](./sfc/kmm/) will configure a `Module` CR with instructions for KMM to build onload's `sfc` in a new container and deploy that to all nodes.

To deploy only to nodes with Solarflare cards (PCIe Subsystem Vendor ID: 1924), modify the [Module YAML](./sfc/kmm/sfc-module.yaml) to utilise that NFD-provided node feature:

```yaml
  selector:
    feature.node.kubernetes.io/pci-1924.present: "true"
```

Given the cluster's `NodeFeatureDiscovery` CR included the `configData` of:

```yaml
      sources:
        pci:
          deviceClassWhitelist:
            - "1924"
          deviceLabelFields:
            - "subsystem_vendor"
```

Before you apply the `Module` custom resource for the SFC driver you must
remove the existing in-tree driver.

For each appropriate node in the cluster:
```console
# rmmod sfc
```

Then apply the manifest:

```console
$ oc apply -k sfc/kmm/
```

#### Day-0 deployment using MachineConfig

To apply the SFC MachineConfig, it is a prerequisite that there exists a local
docker registry containing an image with the SFC module built and present in
`/opt` in the node filesystem. This is expected to be present at the following
location:

```
image-registry.openshift-image-registry.svc:5000/openshift-kmm/sfc-module
```

The generated image must be labelled with the exact kernel version necessary
for the worker nodes. All worker nodes must share the same kernel version.

Apply the SFC MachineConfig:

```console
$ butane sfc/mco/99-sfc-machineconfig.bu -d sfc/mco/ -o sfc/mco/99-sfc-machineconfig.yaml
$ oc apply -f sfc/mco/99-sfc-machineconfig.yaml
```

Please ensure that the OpenShift version specified within the above butane file
tracks the appropriate specification revision. Stable versions can be checked
[here](https://github.com/coreos/butane/blob/main/docs/specs.md#stable-specification-versions).

E.g. OpenShift 4.10.62 -> 4.10.0


### Onload manifests

Create and apply the composed Kustomize manifest that defines Onload resources:

```
$ oc apply -f onload/imagestream/imagestream.yaml
$ oc new-project onload-runtime
$ oc apply [--dry-run=client] -k onload/dev
```

The "onload/imagestream/imagestream.yaml" manifest will create the new `onload-clusterlocal` namespace for ImageStream(s), referring to the Onload resources built locally in the cluster.

Another `onload-runtime` namespace is configurable. The users who change it, also need to patch the `namespace` field in the corresponding "kustomization.yaml" file, e.g. "onload/dev/kustomization.yaml" in the above case.

### Disable chronyd

This will disable the `chronyd.service` on `worker` role nodes.

```
$ butane sfptpd/99-worker-chronyd.bu -o sfptpd/99-worker-chronyd.yaml
$ oc apply -f sfptpd/99-worker-chronyd.yaml
```

Please ensure that the OpenShift version specified within the above butane file
tracks the appropriate specification revision. Stable versions can be checked
[here](https://github.com/coreos/butane/blob/main/docs/specs.md#stable-specification-versions).

E.g. OpenShift 4.10.62 -> 4.10.0


### sfptpd

```console
$ oc create -k sfptpd
```

or with separate build and deploy processes:

To build images in cluster:
```console
$ oc create -k sfptpd/build
```

To deploy:
```console
$ oc create -f sfptpd/deploy
```

Please note that `sfptpd/deploy/1000-sfptpd-daemonset.yaml` uses the interface name `sf0` as a placeholder for development purposes. This should be modified to use an appropriate value before deploying to the cluster.

### Onloaded application

Here we suggest running the `sfnt-pingpong` from [https://github.com/Xilinx-CNS/cns-sfnettest.git](https://github.com/Xilinx-CNS/cns-sfnettest.git).

Create a new BuildConfig to build the app and a couple of new demo pods with:

```
$ oc apply -k examples/profiles/latency/
```

It should create pods `onload-sfnettest-server` and `onload-sfnettest-client` on workers `compute-0` and `compute-1` respectively.

Get the SFC IP address of the `onload-sfnettest-server` pod:
```console
$ oc describe pod onload-sfnettest-server | grep AddedInterface
  Normal  AddedInterface  24s   multus   Add eth0 [192.168.8.114/23] from openshift-sdn
  Normal  AddedInterface  24s   multus   Add net1 [198.19.0.1/16] from default/ipvlan-sf0
```

The server pod is already running the accelerated `sfnt-pingpong` instance.

Run the client:
```
sh-4.4# ./sfnt-pingpong udp 198.19.0.1
```

Based on [examples/sfnettest/README.md](examples/sfnettest/README.md).

## Uninstall

Use `oc delete` to uninstall Onload and example pods:
```
$ oc delete -k examples/profiles/latency
$ oc delete project sfptpd
$ oc delete -f 99-worker-chronyd.yaml
$ oc delete -k onload/dev
$ oc delete project onload-runtime
$ oc delete -f onload/imagestream/imagestream.yaml
```

To remove SFC Module:
```console
$ oc delete -k sfc/kmm/
```

To remove SFC MachineConfig
```console
$ oc delete -f sfc/mco/99-sfc-machineconfig.yaml
```
(Applying and removing MachineConfig might reboot the targeted nodes.)

Make sure the Onload kernel modules are unloaded, i.e. by running directly in the worker nodes:
```
# lsmod | grep onload
# rmmod onload sfc_char sfc_resource
```
This is because, by the time of the module removal, the kernel modules may still be in use by userspace, and the early revisions of KMM won't retry the module unloading.

Kustomize will remove build config but not build products. Confirm the state of uploaded Onload images:
```
$ oc get image | grep onload
```
Remove any outstanding manually with `oc delete image`. (Not providing any automated invocation to prevent removal of the false-positive images.)

## Troubleshooting

Onload comes with a troubleshooting container with pre-installed utilities like `onload_stackdump`. Create a privileged debugging container to use these utilities interactively:

```console
$ oc debug --image-stream=onload-clusterlocal/onload-diagnostics:v8.1.0 node/compute-0
Temporary namespace openshift-debug-d2ss2 is created for debugging node...
Starting pod/compute-0-debug ...
To use host binaries, run `chroot /host`
Pod IP: 192.168.128.6
If you don't see a command prompt, try pressing enter.
sh-4.4# onload_stackdump
#stack-id stack-name      pids
0         -               4159878
```

## Deploying artifacts into an airgapped cluster

### Images required

This is a list of the images that are currently needed as reported by podman:
```console
$ sudo podman images --format "table {{.Repository}} {{.Tag}}"
REPOSITORY                                                                                           TAG
default-route-openshift-image-registry.apps.test.kube.test/default/onload-sfnettest                  sfnettest-1.6.0-rc1
default-route-openshift-image-registry.apps.test.kube.test/sfptpd/sfptpd                             git-ab881b3
default-route-openshift-image-registry.apps.test.kube.test/onload-clusterlocal/onload-module         git27b3826-4.18.0-372.49.1.el8_6.x86_64
default-route-openshift-image-registry.apps.test.kube.test/onload-clusterlocal/onload-device-plugin  latest
default-route-openshift-image-registry.apps.test.kube.test/onload-clusterlocal/onload-user           git27b3826
default-route-openshift-image-registry.apps.test.kube.test/openshift-kmm/sfc-module                  git27b3826-4.18.0-372.49.1.el8_6.x86_64
```

### Getting the images

The images can either be built using podman directly (currently unsupported) or
within a cluster with access to a git web host, either github.com or a locally
hosted clone of \<named repos\>.


#### Pulling images from a cluster
First follow the section about [logging in via podman](#podman-login) below.

The images can be pulled via the following command:
```console
$ sudo podman pull ${REGHOST}/openshift-kmm/sfc-module:git27b3826-4.18.0-372.49.1.el8_6.x86_64 --tls-verify=false
```

Repeat this process for each image you want to pull from the cluster.
The image specification to pull from the cluster should be of the form:
```
REGISTRY/OPENSHIFT_NAMESPACE/IMAGE_NAME:IMAGE_TAG
```

`podman save` can be used to save the into a local file, e.g:
```console
$ sudo podman save -o images/sfc.tar default-route-openshift-image-registry.apps.test.kube.test/openshift-kmm/sfc-module:git27b3826-4.18.0-372.49.1.el8_6.x86_64
```
which will write the image into `images/sfc.tar`. Then use `podman load` to
load the image from the written file:
```
$ sudo podman load -i images.tar
```

### Podman login

In order to be able to pull/push images into the cluster's image registry you
must log in with podman.

1. log in to openshift cluster
```console
$ oc login -u kubeadmin -p PASSWORD
```
2. Get the name of the image registry
```console
$ oc get route default-route -n openshift-image-registry --template='{{ .spec.host }}'
```
This can be stored in a local variable
```console
$ REGHOST=`oc get route default-route -n openshift-image-registry --template='{{ .spec.host }}'`
```
3. Actually login to the image registry with podman
```console
$ sudo podman login -u kubeadmin -p $(oc whoami -t) ${REGHOST} --tls-verify=false
```

#### A Note on `--tls-verify=false`

Without doing any extra steps podman will complain about the certificates used
by the image registry. For development purposes `--tls-verify=false` can be
used to bypass this issue. Production environments shall follow internal
certificate handling procedures to secure podman access.

### Pushing the images to the cluster's image-registry

For each image you want to push run the command:
```console
$ sudo podman push REGISTRY/OPENSHIFT_NAMESPACE/IMAGE_NAME:TAG --tls-verify=false
```

### Deploying sfc driver

#### KMM Module deployment

If KMM v1.0 is used, the in-tree driver must be removed from the worker nodes
before deployment. This is a one-time action which is automated by the
MachineConfig component upon reboot. In KMM v1.1 onwards, add
`inTreeModuleToRemove: sfc` to the `sfc` Module CR.

```sh
rmmod sfc
```

```console
$ oc create -k sfc/deploy
```
#### MCO deployment (for day 0 deployments)

```console
$ butane sfc/mco/99-sfc-machineconfig.bu -d sfc/mco/ -o sfc/mco/99-sfc-machineconfig.yaml
$ oc apply -f sfc/mco/99-sfc-machineconfig.yaml
```

### Deploying onload

```console
$ oc new-project onload-runtime
$ oc create -k onload/deploy
```

### Deploying sfptpd

```console
$ oc create -k sfptpd/deploy
```

### Deploying the example application (sfnt-pingpong)

```console
$ oc create -k examples/sfnettest/deploy
```

or to use an example profile:
```console
$ oc create -k examples/profiles/latency/deploy
```

Then follow instruction for running the client in [examples/README.md](examples/README.md)

Copyright (c) 2023 Advanced Micro Devices, Inc.
