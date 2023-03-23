# Mirroring with kOps

## Identifying Images To Mirror

TODO: Help Wanted!

## Mirroring Images

See our general list of [mirroring options](./README.md#Mirroring-Images)

## Using Mirrored Images

You can configure containerd to use registry mirrors for in the kOps cluster spec.
You'll need to add an entry for `"registry.k8s.io"` with your mirror.

Docs: https://kops.sigs.k8s.io/cluster_spec/#registry-mirrors
