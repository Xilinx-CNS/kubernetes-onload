# Onload Operator and Onload Device Plugin for Kubernetes and OpenShift

NOTE: The project is under development.

## Prerequisites for building

1) Go version 1.21+.
2) The binaries of `oc`, `kubectl` present in `$PATH`.
3) Docker or a symbolic link to an API equivalent (e.g. `podman`) present in
`$PATH`.
4) The following exports:
```text
export KUBECONFIG=<Path to Kubeconfig for your deployment>
export REGISTRY_URL=<URL for your docker registry>
```
5) A docker registry location that is accessible from the cluster.

Do note the following deployment instructions assume seemless push/pull
capabilities to the above registry. If your registry is not running with
TLS setup, additional configuration will be necessary for pushing images,
and to enable access from your deployment.

6) The ability to pull appropriate RedHat images from Quay.io. To do so, this
   requires the Red Hat pull secret to be setup appropriately for your
   container builder.

## Caveats for deployment

* Do note the following process will cause a reload Onload (and optionally SFC)
  kernel modules on the deployment and uninstall. As such, network configuration
  for the interfaces will be wiped on reload. It is therefore necessary for a
  mechanism to enable the appropriate interfaces regain the correct
  network-configuration post deployment. This can be achieved with DHCP or
  machine configuration. This is particularly important for deployments
  using Solarflare interfaces for the control plane traffic.

* SFC interface names can adjust on installation. This is because the default
  interface names have changed from v4 to v5 of the SFC kernel module. As such,
  make sure appropriate measures have been taken for any additional network
  configurations that depend on this information.


## Deploying the Onload Operator

### Build Onload images

The required Onload images are: source, userland and kernel.

1. Clone OpenOnload from
[https://github.com/Xilinx-CNS/onload](https://github.com/Xilinx-CNS/onload)
2. Create an Onload source tarball:
```text
scripts/onload_mkdist
```
3. Create Onload source and userland images, to be pushed later to the image
registry of your choice:
```text
scripts/onload_mkcontainer --source $REGISTRY_URL/onload-source:latest --user $REGISTRY_URL/onload-user:latest *.tgz
```
By default, the Onload userland uses UBI libc. Please check and patch the
Dockerfile if this is incompatible with your application's environment.

4. Push them to the image registry:
```text
docker push $REGISTRY_URL/onload-source:latest
docker push $REGISTRY_URL/onload-user:latest
```

5. In this repository, build an Onload kernel image:
```text
make onload-module-dtk
ONLOAD_SOURCE_IMAGE_REPO=$REGISTRY_URL/onload-source
ONLOAD_SOURCE_IMAGE_TAG=latest
ONLOAD_MODULE_IMAGE_REPO=$REGISTRY_URL/onload-module
```

Please note that the `onload-module-dtk` target is currently tailored to
OpenShift. Please edit
[build/onload-module/Makefile](build/onload-module/Makefile) to accommodate
non-OpenShift kernels.

6. Export the following Onload information for tagging the Onload kernel image.
```text
export ONLOAD_VERSION=<onload-tag>
export ONLOAD_KERNEL_VERSION=<kernel version to be built>
```
n.b. `onload-tag` can be of the form of a release e.g. `v8.1.0` or a git commit
hash, and `onload_kernel_version` is expected to be of the form
`4.18.0-372.49.1.el8_6.x86_64`.


7. Push the Onload kernel image (copied the autogenerated hash and kernel
version):
```text
docker push $REGISTRY_URL/onload-module:$ONLOAD_VERSION-$ONLOAD_KERNEL_VERSION
```


## SFC MachineConfig (for OpenShift Users using SFC NICs for the control plane)

For users in the above configuration MachineConfig is recommended for
controlling the loading and unloading of SFC kernel modules. This process
ensures a clean installation/uninstallation process for Onload Operator
by ensuring the interfaces remain operational independently.

### Building SFC MachineConfig

Appropriate MachineConfig can be generated with the following commands.

For users of Openshift v4.10.0-4.13.0:

```text
 make sfc-mc-docker-build IMG=$REGISTRY_URL/onload-module:$ONLOAD_VERSION-$ONLOAD_KERNEL_VERSION
```

For users of OpenShift v4.14.0+:

```text
 make sfc-mc-docker-build IMG=$REGISTRY_URL/onload-module:$ONLOAD_VERSION-$ONLOAD_KERNEL_VERSION OPENSHIFT_VER=4.14.0
```

### Deploying the SFC MachineConfig

```text
make sfc-mc-deploy
```

### Prepare Kubernetes cluster

The Onload Operator v3 depends on third-party software:

1. Kernel Module Manamagent (KMM) Operator v1.1.1. For installation
instructions please see
[https://openshift-kmm.netlify.app/documentation/install/](official documentation).
2. Multus CNI. A sample configuration using macvlan can be found
[https://github.com/k8snetworkplumbingwg/multus-cni/blob/master/examples/macvlan-pod.yml](here).

Please note that `NetworkAttachmentDefinition` is used later in the definition
of Onloaded applications.


### Build and deploy Onload Operator v3


1. Create and push the Onload Operator controller image:
```text
make docker-build docker-push IMG=$REGISTRY_URL/operator:latest
```
2. Create and push the Onload Device Plugin image:
```text
make device-plugin-docker-build device-plugin-docker-push
DEVICE_IMG=$REGISTRY_URL/deviceplugin:latest
```
3. Deploy the Onload Operator v3:
```text
make deploy IMG=$REGISTRY_URL/operator:latest
```
4. Patch the following Onload CR sample accordingly:
```diff
diff --git a/config/samples/onload_v1alpha1_onload.yaml
b/config/samples/onload_v1alpha1_onload.yaml
index 0ffba0c..2768bdc 100644
--- a/config/samples/onload_v1alpha1_onload.yaml
+++ b/config/samples/onload_v1alpha1_onload.yaml
@@ -55,9 +55,9 @@ spec:
     # Example image locations using openshift local image registry.
     kernelMappings:
       - regexp: '^.*\.x86_64$'
+        kernelModuleImage: $REGISTRY_URL/onload-module:$ONLOAD_VERSION-$ONLOAD_KERNEL_VERSION
         sfc: {}
+    userImage: $REGISTRY_URL/onload-user:latest
+    version: $ONLOAD_VERSION
     imagePullPolicy: Always
   devicePlugin:
+    devicePluginImage: $REGISTRY_URL/deviceplugin:latest
```
Please also make sure `devicePluginImage` is correct. Another important field
is the node `selector`, which tells the Onload Operator where to deploy the
Onload Device Plugin DaemonSet and Module kind (KMM).

N.B. For deployments utilising our SFC MachineConfig, the following patch
is required.

```diff
diff --git a/config/samples/onload_v1alpha1_onload.yaml b/config/samples/onload_v1alpha1_onload.yaml
index 0ffba0c..50babed 100644
--- a/config/samples/onload_v1alpha1_onload.yaml
+++ b/config/samples/onload_v1alpha1_onload.yaml
@@ -56,7 +56,6 @@ spec:
     kernelMappings:
       - regexp: '^.*\.x86_64$'
         kernelModuleImage: image-registry.openshift-image-registry.svc:5000/onload-clusterlocal/onload-module:v8.1.0-${KERNEL_FULL_VERSION}
-        sfc: {}
     userImage: image-registry.openshift-image-registry.svc:5000/onload-clusterlocal/onload-user:v8.1.0
     version: 8.1.0
     imagePullPolicy: Always
```


5. Finally, deploy the Onload CR:
```text
oc apply -k config/samples/
```

## Run Onloaded application

At this point, Onload is deployed at the cluster, and the users can run
Onloaded applications, e.g.
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: test
  annotations:
    k8s.v1.cni.cncf.io/networks: ipvlan-sf0
spec:
  restartPolicy: Never
  securityContext:
    runAsNonRoot: true
    seccompProfile:
      type: RuntimeDefault
  containers:
  - name: test
    image: test:latest
    imagePullPolicy: Always
    command:
    - /test
    resources:
      limits:
        amd.com/onload: 1
    securityContext:
      allowPrivilegeEscalation: false
      capabilities:
        drop:
          - ALL
  nodeName: compute-0
```

There are two key fields in the above example CR:
1. `metadata.annotations` adds the acceleratable interfaces to the pods.
2. `spec.containers[].resources.limits` injects Onload, i.e. devfs and \*.so
files, and also sets `LD_PRELOAD`.

No further modifications are required to enable Onloaded applications.

---

Copyright (c) 2023 Advanced Micro Devices, Inc.