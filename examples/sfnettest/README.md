# Onload application examples

## cns-sfnettest

Here, we run a small utility `sfnt-pingpong` to demonstrate Onload acceleration.

Create new `onload-sfnettest` containers with:

```
$ oc apply -k examples/sfnettest/
```
This will create a new BuildConfig and then a couple of pods running on the `compute-0` and `compute-1` workers. These pods will have Onload user components installed and the `sfnt-pingpong` binary in the working directory.

Get the SFC IP address of the `onload-sfnettest-server` pod:

```console
$ oc describe pod onload-sfnettest-server | grep AddedInterface
  Normal  AddedInterface  24s   multus   Add eth0 [192.168.8.114/23] from openshift-sdn
  Normal  AddedInterface  24s   multus   Add net1 [198.19.0.1/16] from default/ipvlan-sf0
```

The server pod is already running the accelerated `sfnt-pingpong` instance.

Run the client from the `onload-sfnettest-client` pod:
```console
sh-4.4# ./sfnt-pingpong udp 198.19.0.1
```

## Running with onload runtime profiles

If you wish to run onload with a runtime profile for example:
```console
$ onload -p latency APP
```
This can be accomplished by applying a kustomization to supply the appropriate
environment variables to the pod. Included in this repo are two examples of how
a profile can be applied: `latency` and `throughput-base`. These examples are
provided in `examples/profiles`.

To apply the profile use the `oc` command line tool:
```console
$ oc create -k examples/profiles/latency/
```

This will create a `ConfigMap` resource and patch the pod definitions in `cns-sfnettest.yaml`
to use the environment variables.


Copyright (c) 2023 Advanced Micro Devices, Inc.
