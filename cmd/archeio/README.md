# Archeio

αρχείο (archeío) is roughly? Greek for "registry"

This binary is a custom redirect/alias server for the Kubernetes project's 
OCI artifact ("docker image") hosting.

Current design details will be detailed here as they mature.

The original design doc is shared with members of
[dev@kubernetes.io](https://groups.google.com/a/kubernetes.io/g/dev), 
anyone can join this list and gain access to read
[the document](https://docs.google.com/document/d/1yNQ7DaDE5LbDJf9ku82YtlKZK0tcg5Wpk9L72-x2S2k/). 
It is not accessible to accounts that are not members of the Kubernetes mailinglist
due to organization constraints and joining the list is the most reliable way to gain
access. See https://git.k8s.io/community/community-membership.md

For more current details see also: https://github.com/kubernetes/k8s.io/wiki/New-Registry-url-for-Kubernetes-(registry.k8s.io)

**NOTE**: The code in this binary is **not** intended to be fully reusable,
it is the most efficient and expedient implementation of
Kubernetes SIG K8s-Infra's needs. However, some of the packages under
[`pkg/`](./../../pkg/) may be useful if you have a similar problem,
and they should be pretty generalized and re-usable.

-----

For a rough TLDR of the current design:

- Images are hosted primarily in the existing Kubernetes [GCR](https://gcr.io/) registry
- Mirrors *of content* blobs are hosted in S3 buckets in AWS
- AWS clients are detected by client IP address and redirect to a local S3 bucket copy
*only* when requesting content blobs, *not* manifests, manifest lists, tags etc.
- All other requests are redirected to the original upstream registry

For more detail: see [docs/request-handling.md](./docs/request-handling.md)

In addition, in order to get the registry.k8s.io domain in place, initially this
binary is *only* serving the trivial redirect to the existing registry 
(https://k8s.gcr.io), so we can safely move users / clients to the new domain
that will eventually serve the more complex version.

Development is at https://registry-sandbox.k8s.io which is *not* supported for
any usage outside of the development of this project and may or may not be
working at any given time. Changes will be deployed there before we deploy
to production, and be exercised by a subset of Kubernetes' own CI.

For AWS client-IP matching, see [`pkg/net/cidrs/aws`](./../../pkg/net/cidrs/aws)
