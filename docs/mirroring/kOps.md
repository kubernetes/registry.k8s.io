# Mirroring with kOps

## Identifying Images To Mirror

`kops get assets` can list images and files needed by kOps.

Docs: https://kops.sigs.k8s.io/cli/kops_get_assets/

## Mirroring Images

`kops get assets --copy` can be used to mirror.

See:
- https://kops.sigs.k8s.io/cli/kops_get_assets/
- https://kops.sigs.k8s.io/operations/asset-repository/

See also our general list of [mirroring options](./README.md#Mirroring-Images)

## Using Mirrored Images

kOps has documentation for using local assets to create a cluster at:
https://kops.sigs.k8s.io/operations/asset-repository/

You can also configure containerd to use registry mirrors for in the kOps cluster spec.
You'll need to add an entry for `"registry.k8s.io"` with your mirror.

Docs: https://kops.sigs.k8s.io/cluster_spec/#registry-mirrors
