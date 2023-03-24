# Mirroring

This guide covers mirroring images you use on registry.k8s.io
to a host under your own control and using those images.

The specific sub-steps will depend on the tools you use, but in general you will need to:

1. Identify the images you need: [Identifying Images To Mirror](#Identifying-Images-To-Mirror)
2. Mirror those images to your own registry: [Mirroring Images](#Mirroring-Images)
3. Configure your tools to use the mirrored images: [Using Mirrored Images](#Using-Mirrored-Images)

We have guides here for each of these steps.

## Identifying Images To Mirror
<!--
NOTE: Wherever possible do not duplicate external content.

Instead, link to existing official guides and merely provide a lightweight pointer here.

See: https://kubernetes.io/docs/contribute/style/content-guide/#dual-sourced-content
-->

<!--TODO: Generically identifying registry.k8s.io images in manifests / charts / addons.-->

If you have a running cluster then our [community-images] krew plugin can
help you identify Kubernetes Project image references to mirror like this:

```console
kubectl community-images --mirror
```

**NOTE**: This will only find images specified in your currently running pods,
and not for example the "pause" image used to implement pods in containerd / cri-o / dockershim.

For specific tools we have these guides:

- For containerd see: [containerd.md](./containerd.md)
- For kubeadm see: [kubeadm.md](./kubeadm.md)
- For kOps see: [kOps.md](./kOps.md)


## Mirroring Images
<!--
NOTE: Wherever possible do not duplicate external content.

Instead, link to existing official guides and merely provide a lightweight pointer here.

See: https://kubernetes.io/docs/contribute/style/content-guide/#dual-sourced-content
-->

Here are some options for copying images you wish to mirror to your own registry.

<!-- FOSS Mirroring Tools First -->

### Mirroring With `crane` Or `gcrane`

`crane` is an open-source tool for interacting with remote images and registries.
`gcrane` is a superset of crane with GCP specific additional features.

For `crane` use `crane copy registry.k8s.io/pause:3.9 my-registry.com/pause:3.9`.
Docs: https://github.com/google/go-containerregistry/blob/main/cmd/crane/doc/crane_copy.md

For `gcrane` see: https://cloud.google.com/container-registry/docs/migrate-external-containers

To mirror all images surfaced by [community-images], you can use this shell snippet:
```shell
# set MIRROR to your own host
export MIRROR=my-registry.com
# copy all Kubernetes project images in your current cluster to MIRROR
kubectl community-images --mirror --plain |\
   xargs -i bash -c 'set -x; crane copy "$1" "${1/registry.k8s.io/'"${MIRROR}"'}"' - '{}'
```

### Mirroring With `oras`

`oras` is an open-source tool for managing images and other artifacts in OCI registries.

For `oras` use `oras copy registry.k8s.io/pause:3.9 my-registry.com/pause:3.9`.
Docs: https://oras.land/cli_reference/4_oras_copy/

To mirror all images surfaced by [community-images], you can use this shell snippet:
```shell
# set MIRROR to your own host
export MIRROR=my-registry.com
# copy all Kubernetes project images in your current cluster to MIRROR
kubectl community-images --mirror --plain |\
   xargs -i bash -c 'set -x; oras copy "$1" "${1/registry.k8s.io/'"${MIRROR}"'}"' - '{}'
```


### Mirroring With Harbor

You can use Harbor to set up a proxy cache for Kubernetes images.

From the Harbor web interface, go to "Registries" and click "New Endpoint".
Create an endpoint `registry.k8s.io` with the endpoint URL https://registry.k8s.io.
Go to "Projects" and click "New Project".
Create a project named something like 'k8s', click "Proxy Cache" and select your `registry.k8s.io` endpoint.
Docs: https://goharbor.io/docs/2.1.0/administration/configure-proxy-cache/

<!-- NON-FOSS Mirroring Tools Below Here -->


## Using Mirrored Images
<!--
NOTE: Wherever possible do not duplicate external content.

Instead, link to existing official guides and merely provide a lightweight pointer here.

See: https://kubernetes.io/docs/contribute/style/content-guide/#dual-sourced-content
-->

<!--TODO: cri-o-->

A simple option for many cases is to update the `image` fields in your 
Kubernetes manifests (deployments, pods, replicasets, etc) to reference
your mirrored images instead.

For specific tools we have these guides:

- For containerd see: [containerd.md](./containerd.md)
- For kubeadm see: [kubeadm.md](./kubeadm.md)
- For kOps see: [kOps.md](./kOps.md)

[community-images]: https://github.com/kubernetes-sigs/community-images
