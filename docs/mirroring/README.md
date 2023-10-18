# Mirroring

This guide covers mirroring images you use on registry.k8s.io
to a host under your own control and using those images.

The specific sub-steps will depend on the tools you use, but in general you will need to:

1. Identify the images you need: [Identifying Images To Mirror][identifying-images]
2. Mirror those images to your own registry: [Mirroring Images][mirroring-images]
3. Configure your tools to use the mirrored images: [Using Mirrored Images][using-mirrored-images]

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
- For cri-o see: [cri-o.md](./cri-o.md)
- For cri-dockerd see: [cri-dockerd.md](./cri-dockerd.md)
- For kubeadm see: [kubeadm.md](./kubeadm.md)
- For kOps see: [kOps.md](./kOps.md)
- For Cluster API see: [cluster-api.md](./cluster-api.md)


## Mirroring Images
<!--
NOTE: Wherever possible do not duplicate external content.

Instead, link to existing official guides and merely provide a lightweight pointer here.

See: https://kubernetes.io/docs/contribute/style/content-guide/#dual-sourced-content
-->

Here are some options for copying images you wish to mirror to your own registry.

<!-- FOSS Mirroring Tools Go First Below Here! -->
<!-- Commercial / Non-FOSS Mirroring Options Go Further Below -->

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

Once you're done, see [Using Mirrored Images][using-mirrored-images].

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

Once you're done, see [Using Mirrored Images][using-mirrored-images].

### Mirroring With Harbor

You can use Harbor to set up a proxy cache for Kubernetes images.

From the Harbor web interface, go to "Registries" and click "New Endpoint".
Create an endpoint `registry.k8s.io` with the endpoint URL https://registry.k8s.io.
Go to "Projects" and click "New Project".
Create a project named something like 'k8s', click "Proxy Cache" and select your `registry.k8s.io` endpoint.
Docs: https://goharbor.io/docs/2.1.0/administration/configure-proxy-cache/

Once you're done, see [Using Mirrored Images][using-mirrored-images].

<!-- NON-FOSS Mirroring Tools Go Below Here! -->

### Mirroring With ECR

AWS ECR wrote a guide for configuring a `registry.k8s.io` pull-through cache here:

https://aws.amazon.com/blogs/containers/announcing-pull-through-cache-for-registry-k8s-io-in-amazon-elastic-container-registry/

After following this guide, you may additionally want to see our [Using Mirrored Images][using-mirrored-images] reference below.


## Using Mirrored Images
<!--
NOTE: Wherever possible do not duplicate external content.

Instead, link to existing official guides and merely provide a lightweight pointer here.

See: https://kubernetes.io/docs/contribute/style/content-guide/#dual-sourced-content
-->

In many cases it is sufficient to update the `image` fields in your
Kubernetes manifests (deployments, pods, replicasets, etc) to reference
your mirrored images instead.

For specific tools we have these guides:

- For containerd see: [containerd.md](./containerd.md)
- For cri-o see: [cri-o.md](./cri-o.md)
- For cri-dockerd see: [cri-dockerd.md](./cri-dockerd.md)
- For kubeadm see: [kubeadm.md](./kubeadm.md)
- For kOps see: [kOps.md](./kOps.md)
- For Cluster API see: [cluster-api.md](./cluster-api.md)

[identifying-images]: #Identifying-Images-To-Mirror
[mirroring-images]: #Mirroring-Images
[using-mirrored-images]: #Using-Mirrored-Images
[community-images]: https://github.com/kubernetes-sigs/community-images

