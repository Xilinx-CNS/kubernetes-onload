# Build Onload images

This is a temporary directory (until ON-15022 is done) to build Onload images for the Onload Operator v3 to consume.

Please check the Makefile and set these parameters to suit your Kubernetes cluster:

* `DTK_AUTO` is the base image to build the Onload and SFC kernel modules, which should match the worker's kernel.
* `ONLOAD_MODULE_IMAGE` is the image registry location to push the Onload and SFC kernel image.
* `ONLOAD_USER_IMAGE` is the image registry location to push the Onload userland image.

Further notes:

* Please make sure the Onload CR points to the right images after changing the default `ONLOAD_MODULE_IMAGE` and `ONLOAD_USER_IMAGE`.
* Please check that the Onload userland base image does not impose libc incompatibility with the Onloaded application.
* The Makefile uses OpenOnload v8.1.0 as an example, but the Onload version could be changed too.

Example run from with the overridden image registry:

```
make -C hack/build \
    ONLOAD_MODULE_IMAGE=kitchen-sink.kube.test:5000/onload-module:v8.1.0-4.18.0-372.49.1.el8_6.x86_64 \
    ONLOAD_USER_IMAGE=kitchen-sink.kube.test:5000/onload-user:v8.1.0
```

Onload CR diff:

```diff
diff --git a/config/samples/onload_v1alpha1_onload.yaml b/config/samples/onload_v1alpha1_onload.yaml
index 0ffba0c..0d9ab7e 100644
--- a/config/samples/onload_v1alpha1_onload.yaml
+++ b/config/samples/onload_v1alpha1_onload.yaml
@@ -55,9 +55,9 @@ spec:
     # Example image locations using openshift local image registry.
     kernelMappings:
       - regexp: '^.*\.x86_64$'
-        kernelModuleImage: image-registry.openshift-image-registry.svc:5000/onload-clusterlocal/onload-module:v8.1.0-${KERNEL_FULL_VERSION}
+        kernelModuleImage: kitchen-sink.kube.test:5000/onload-module:v8.1.0-4.18.0-372.49.1.el8_6.x86_64
         sfc: {}
-    userImage: image-registry.openshift-image-registry.svc:5000/onload-clusterlocal/onload-user:v8.1.0
+    userImage: kitchen-sink.kube.test:5000/onload-user:v8.1.0
     version: 8.1.0
     imagePullPolicy: Always
   devicePlugin:
```

Copyright (c) 2023 Advanced Micro Devices, Inc.
