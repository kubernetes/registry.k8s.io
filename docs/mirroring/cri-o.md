# Mirroring With cri-o

# Identifying Images to Mirror

If you're using [cri-o] as a Kubernetes [CRI] implementation, cri-o
uses the ["pause" image][pause] from Kubernetes to implement pods.
You may want to mirror this critical image to your own host.

The pause image configured can be found by running:
```shell
cri-o config | grep pause_image
```

## Mirroring Images

See our general list of [mirroring options](./README.md#Mirroring-Images)

# Using Mirrored Images

For pause see `pause_image` in the `cri.image` config docs:
https://github.com/cri-o/cri-o/blob/main/docs/crio.conf.5.md#crioimage-table

cri-o also supports configuring mirrors for registry hosts, which is documented at:
https://github.com/containers/image/blob/main/docs/containers-registries.conf.5.md

You can use containers-registries.conf to configure a mirror for registry.k8s.io


[cri-o]: https://cri-o.io/
[pause]: https://www.ianlewis.org/en/almighty-pause-container
[CRI]: https://kubernetes.io/docs/concepts/architecture/cri/
