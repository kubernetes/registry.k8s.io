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

- For containerd see: [containerd.md](./containerd.md)
- For kubeadm see: [kubeadm.md](./kubeadm.md)


## Mirroring Images
<!--
NOTE: Wherever possible do not duplicate external content.

Instead, link to existing official guides and merely provide a lightweight pointer here.

See: https://kubernetes.io/docs/contribute/style/content-guide/#dual-sourced-content
-->

This section covers some options for copying images you wish to mirror to your own registry.

### Mirroring With `crane` Or `gcrane`

`crane` is an open-source tool for interacting with remote images and registries.
`gcrane` is a superset of crane with GCP specific additional features.

For `crane` use `crane copy registry.k8s.io/pause:3.9 my-registry.com/pause:3.9`.
Docs: https://github.com/google/go-containerregistry/blob/main/cmd/crane/doc/crane_copy.md

For `gcrane` see: https://cloud.google.com/container-registry/docs/migrate-external-containers


### Mirroring With `oras`

`oras` is an open-source tool for managing images and other artifacts in OCI registries.

For `oras` use `oras copy registry.k8s.io/pause:3.9 my-registry.com/pause:3.9`.
Docs: https://oras.land/cli_reference/4_oras_copy/


### Mirroring With Harbor

You can use Harbor to set up a proxy cache for Kubernetes images.

From the Harbor web interface, go to "Registries" and click "New Endpoint".
Create an endpoint `registry.k8s.io` with the endpoint URL https://registry.k8s.io.
Go to "Projects" and click "New Project".
Create a project named something like 'k8s', click "Proxy Cache" and select your `registry.k8s.io` endpoint.
Docs: https://goharbor.io/docs/2.1.0/administration/configure-proxy-cache/


## Using Mirrored Images
<!--
NOTE: Wherever possible do not duplicate external content.

Instead, link to existing official guides and merely provide a lightweight pointer here.

See: https://kubernetes.io/docs/contribute/style/content-guide/#dual-sourced-content
-->

<!--TODO: cri-o, general manifests-->

- For containerd see: [containerd.md](./containerd.md)
- For kubeadm see: [kubeadm.md](./kubeadm.md)
