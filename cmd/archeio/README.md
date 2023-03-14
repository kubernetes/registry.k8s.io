# Archeio

αρχείο (archeío) is roughly? Greek for "registry"

This binary is a custom redirect/alias server for the Kubernetes project's 
OCI artifact ("docker image") hosting.

Current design details will be detailed here as they mature.

For more current details see also: https://github.com/kubernetes/k8s.io/wiki/New-Registry-url-for-Kubernetes-(registry.k8s.io)

**NOTE**: The code in this binary is **not** intended to be fully reusable,
it is the most efficient and expedient implementation of
Kubernetes SIG K8s-Infra's needs. However, some of the packages under
[`pkg/`](./../../pkg/) may be useful if you have a similar problem,
and they should be pretty generalized and re-usable.

Please also see the main repo README and in particular [the "stability" note](../../README.md#stability).

-----

For a rough TLDR of the current design:

- Images are hosted primarily in the existing Kubernetes [GCR](https://gcr.io/) registry
- Mirrors *of content* blobs are hosted in S3 buckets in AWS
- AWS clients are detected by client IP address and redirect to a local S3 bucket copy
*only* when requesting content blobs, *not* manifests, manifest lists, tags etc.
- GCP clients and other requests are rerouted to the regional upstream source Artifact Registries

This allows us to offload substantial bandwidth securely, while not having to fully
implement a registry from scratch and maintaining the project's existing security
controls around the GCP source registries (implemented elsewhere in the Kubernetes project).
We only re-route some content-addressed storage requests to additional hosts.

Clients do still need to either pull by digest (`registry.k8s.io/foo@sha256:...`),
verify sigstore signatures, or else trust that the redirector instance is secure.

We maintain relatively tight control over the production redirector instance and
the source registries. Very few contributors have access to this infrastructure.

We have a development instance at https://registry-sandbox.k8s.io which is
*not* supported for any usage outside of the development of this project and 
may or may not be working at any given time. 
Changes will be deployed there before we deploy to production, and be exercised
by a subset of Kubernetes' own CI.


For more detail see:
- How requests are handled: [docs/request-handling.md](./docs/request-handling.md)
- How we test registry.k8s.io changes: [docs/testing.md](./docs/testing.md)
- For IP matching info for both AWS and GCP ranges: [`pkg/net/cloudcidrs`](./../../pkg/net/cloudcidrs)

----

Historical Context:

**You must join one of the open community mailinglists below to access the original design doc.**

The original design doc is shared with members of
[dev@kubernetes.io](https://groups.google.com/a/kubernetes.io/g/dev), 
anyone can join this list and gain access to read
[the document](https://docs.google.com/document/d/1yNQ7DaDE5LbDJf9ku82YtlKZK0tcg5Wpk9L72-x2S2k/). 
It is not accessible to accounts that are not members of the Kubernetes mailinglist
due to organization constraints and joining the list is the **only** way to gain
access. See https://git.k8s.io/community/community-membership.md

It is not fully reflective of the current design anyhow, but some may find it
interesting.

Originally the project primarily needed to take advantage of an offer from Amazon
to begin paying for AWS user traffic, which was the majority of our traffic and
cost a lot due to high amounts of egress traffic between GCP<>AWS.

In addition, in order to get the registry.k8s.io domain in place, initially we
only served a trivial redirect to the existing registry 
(https://k8s.gcr.io), so we could safely start to move users / clients to the new domain
that would eventually serve the more complex version.

Since then we've redesigned a bit to make populating content into AWS async and
not blocked on the image promoter, as well as extending our Geo-routing approach
to detect and route users on dimensions other than "is a known AWS IP in a known AWS region".

More changes will come in the future, and these implementation details while documented
**CANNOT** be depended on.
