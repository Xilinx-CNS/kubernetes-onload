# `Onload` Custom Resource Definition (CRD)

User-provided
`Onload` [Custom Resources (CR)](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)
must conform to the
`Onload` [Custom Resource Definition (CRD)](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/)
schema, which is defined in [onload_types.go](../api/v1alpha1/onload_types.go) and distributed
in [YAML format](../config/crd/bases/onload.amd.com_onloads.yaml) as part of Onload Operator's generated bundle.

It's properties are described within any cluster the Onload Operator is deployed to; reproduced below.

Some properties are passed directly through to KMM. Refer to KMM documentation for further details on the
`regexp`, `build`, and `kernelModuleImage` properties.

```text
$ kubectl explain onload

KIND:     Onload
VERSION:  onload.amd.com/v1alpha1

DESCRIPTION:
     Onload is the Schema for the onloads API

FIELDS:
   apiVersion   <string>
     APIVersion defines the versioned schema of this representation of an
     object. Servers should convert recognized schemas to the latest internal
     value, and may reject unrecognized values. More info:
     https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources

   kind <string>
     Kind is a string value representing the REST resource this object
     represents. Servers may infer this from the endpoint the client submits
     requests to. Cannot be updated. In CamelCase. More info:
     https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds

   metadata     <Object>
     Standard object's metadata. More info:
     https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata

   spec <Object>
     Spec is the top-level specification for onload and related products that
     are controlled by the onload operator

   status       <Object>
     Status contains the statuses for onload and related products that are
     controlled by the onload operator
```

## onload.spec

```text
$ kubectl explain onload.spec

KIND:     Onload
VERSION:  onload.amd.com/v1alpha1

RESOURCE: spec <Object>

DESCRIPTION:
     Spec is the top-level specification for onload and related products that
     are controlled by the onload operator

FIELDS:
   devicePlugin <Object> -required-
     DevicePlugin is the specification of the device plugin that will add a new
     onload resource into the cluster.

   onload       <Object> -required-
     Onload is the specification of the version of onload to be used by this CR

   selector     <map[string]string> -required-
     Selector defines the set of nodes that this onload CR will run on.

   serviceAccountName   <string> -required-
     ServiceAccountName is the name of the service account that the objects
     created by the onload operator will use.
```

### onload.spec.onload

```text
$ kubectl explain onload.spec.onload

KIND:     Onload
VERSION:  onload.amd.com/v1alpha1

RESOURCE: onload <Object>

DESCRIPTION:
     Onload is the specification of the version of onload to be used by this CR

FIELDS:
   imagePullPolicy      <string>
     ImagePullPolicy is the policy used when pulling images. More info:
     https://kubernetes.io/docs/concepts/containers/images#updating-images

   kernelMappings       <[]Object> -required-
     KernelMappings is a list of pairs of kernel versions and container images.
     This allows for flexibility when there are heterogenous kernel versions on
     the nodes in the cluster.

   userImage    <string> -required-
     UserImage is the image that contains the built userland objects, used by
     the cplane and deviceplugin DaemonSets.

   version      <string> -required-
     Version string to associate with this onload CR.
```

#### onload.spec.onload.kernelMappings

```text
$ kubectl explain onload.spec.onload.kernelMappings

KIND:     Onload
VERSION:  onload.amd.com/v1alpha1

RESOURCE: kernelMappings <[]Object>

DESCRIPTION:
     KernelMappings is a list of pairs of kernel versions and container images.
     This allows for flexibility when there are heterogenous kernel versions on
     the nodes in the cluster.

FIELDS:
   build        <Object>
     Build specifies the parameters that are to be passed to the Kernel Module
     Management operator when building the images that contain the module. The
     build process creates a new image which will be written to the location
     specified by the `KernelModuleImage` parameter. If empty no builds will
     take place.

   kernelModuleImage    <string> -required-
     KernelModuleImage is the image that contains the out-of-tree kernel modules
     used by Onload.

   regexp       <string> -required-
     Regexp is a regular expression that is used to match against the kernel
     versions of the nodes in the cluster

   sfc  <map[string]>
     SFC optionally specifies that the controller will manage the SFC kernel
     module.
```

```text
$ kubectl explain onload.spec.onload.kernelMappings.build

KIND:     Onload
VERSION:  onload.amd.com/v1alpha1

RESOURCE: build <Object>

DESCRIPTION:
     Build specifies the parameters that are to be passed to the Kernel Module
     Management operator when building the images that contain the module. The
     build process creates a new image which will be written to the location
     specified by the `KernelModuleImage` parameter. If empty no builds will
     take place.

FIELDS:
   buildArgs    <[]Object>
     BuildArgs is an array of build variables that are provided to the image
     building backend.

   dockerfileConfigMap  <Object> -required-
     ConfigMap that holds Dockerfile contents
```

```text
$ kubectl explain onload.spec.onload.kernelMappings.build.buildArgs

KIND:     Onload
VERSION:  onload.amd.com/v1alpha1

RESOURCE: buildArgs <[]Object>

DESCRIPTION:
     BuildArgs is an array of build variables that are provided to the image
     building backend.

     BuildArg represents a build argument used when building a container image.

FIELDS:
   name <string> -required-

   value        <string> -required-
```

### onload.spec.devicePlugin

```text
$ kubectl explain onload.spec.devicePlugin

KIND:     Onload
VERSION:  onload.amd.com/v1alpha1

RESOURCE: devicePlugin <Object>

DESCRIPTION:
     DevicePlugin is the specification of the device plugin that will add a new
     onload resource into the cluster.

FIELDS:
   baseMountPath        <string>
     BaseMountPath is a prefix to be applied to all onload file mounts in the
     container's filesystem.

   binMountPath <string>
     BinMountPath is the location to mount onload binaries in the container's
     filesystem.

   hostOnloadPath       <string>
     HostOnloadPath is the base location of onload files on the host filesystem.

   imagePullPolicy      <string>
     ImagePullPolicy is the policy used when pulling images. More info:
     https://kubernetes.io/docs/concepts/containers/images#updating-images

   libMounthPath        <string>
     LibMountPath is the location to mount onload libraries in the container's
     filesystem.

   maxPodsPerNode       <integer>
     MaxPodsPerNode is the number of Kubernetes devices that the Onload Device
     Plugin should register with the kubelet. Notionally this is equivalent to
     the number of pods that can request an onload resource on each node.

   mountOnload  <boolean>
     MountOnload is used by the Onload Device Plugin to decide whether to mount
     the `onload` script as a file in the container's filesystem. `onload` is
     mounted at `<baseMountPath>/<binMountpath>` Mutually exclusive with Preload

   setPreload   <boolean>
     Preload determines whether the Onload Device Plugin will set LD_PRELOAD for
     pods using Onload. Mutually exclusive with MountOnload
```

## Footnotes

```yaml
SPDX-License-Identifier: MIT
SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.
```
