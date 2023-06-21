# Onload application examples

## cns-sfnettest

Here, we run a small utility `sfnt-pingpong` to demonstrate Onload acceleration.

Create new `onload-sfnettest` containers with:

```
$ oc apply -f examples/cns-sfnettest.yaml
```
This will create a new BuildConfig and then a couple of pods running on the `compute-0` and `compute-1` workers. These pods will have Onload user components installed and the `sfnt-pingpong` binary in the working directory.

Get the SFC IP address of the `onload-sfnettest-server` pod:

```console
$ oc describe pod onload-sfnettest-server | grep AddedInterface
  Normal  AddedInterface  24s   multus   Add eth0 [192.168.8.114/23] from openshift-sdn
  Normal  AddedInterface  24s   multus   Add net1 [198.19.0.1/16] from default/ipvlan-sf0
```

The server pod is already running the accelerated `sfnt-pingpong` instance.

Run the client:
```console
sh-4.4# LD_PRELOAD=/opt/onload/usr/lib64/libonload.so ./sfnt-pingpong udp 198.19.0.1
oo:sfnt-pingpong[36]: Using Onload 6700428b 2023-06-07 master  [1]
oo:sfnt-pingpong[36]: Copyright (c) 2002-2023 Advanced Micro Devices, Inc.
# cmdline: ./sfnt-pingpong udp 198.19.0.1
# version: no-version
# src: 90b95efce4ec5ab3987cb432a34702e0
# date: Fri Jun  9 13:54:06 UTC 2023
# uname: Linux onload-sfnettest-client 4.18.0-372.49.1.el8_6.x86_64 #1 SMP Thu Mar 9 21:11:55 EST 2023 x86_64 x86_64 x86_64 GNU/Linux
# cpu: model name       : Intel Xeon Processor (Icelake)
# lspci: 01:00.0 Ethernet controller: Red Hat, Inc. Virtio network device (rev 01)
# lspci: 10:00.0 Ethernet controller: Solarflare Communications SFC9220 10/40G Ethernet Controller (rev 02)
# ram: MemTotal:        8144232 kB
# tsc_hz: 1995297020
# LD_PRELOAD=/opt/onload/usr/lib64/libonload.so
# onload_version=6700428b 2023-06-07 master
# server LD_PRELOAD=/opt/onload/usr/lib64/libonload.so
# percentile=99
#
#       size    mean    min     median  max     %ile    stddev  iter
        0       7768    3233    6456    3254610 12403   15768   193000
        1       8066    3454    6454    5525470 12527   39210   186000
        2       8482    3315    8421    7203420 12430   28682   177000
        4       8390    3519    7674    4546294 13765   23643   179000
        8       8467    3499    7795    4991713 13858   22238   177000
        16      7670    3217    6432    7130696 12047   35748   196000
        32      7671    3294    6465    9233952 12279   25130   195000
        64      27209   3439    6403    34432313        13022   352361  67000
        128     9172    3704    6514    5618707 12370   83807   164000
        256     8011    3901    6887    3973178 12264   20089   188000
        512     9240    4772    8617    7012960 13073   31035   162000
        1024    10403   4984    9801    6895483 13712   45897   144000
        1472    10788   5200    10126   5061479 13786   26337   140000
        1473    11375   9516    10645   11413595        16687   43356   132000
        2048    11176   9692    10710   3349849 16312   17651   134000
        4096    13297   11073   11807   5814951 17939   66048   113000
        8192    19454   16604   18798   7731185 25139   35741   77000
        16384   37006   26000   28395   5560660 43588   194081  43000
        32768   59470   47274   50749   9983434 70614   182896  26000
```

(The anomalies in `max` may be specific to the author's development virtualised setup.)

Try running with and without `onload` to compare the reported performance.

Copyright (c) 2023 Advanced Micro Devices, Inc.
