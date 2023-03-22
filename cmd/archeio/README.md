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

- Images are hosted primarily in [Artifact Registry][artifact-registry] instances as the source of truth
  - Why AR?
    - Kubernetes has non-trivial tooling for managing, securing, monitoring etc. of our registries using GCP APIs that fill gaps in the OCI distribution spec, and otherwise allow synchronization and discovery of content, notably https://github.com/opencontainers/distribution-spec/issues/222
    - We have directly migrated all of this infrastructure from GCR (previously k8s.gcr.io) to AR with ~no code changes
    - Until recently our infrastructure funding has primarily been GCP (now matched by AWS) and GCP remains a significant source of funding
- Mirrors *of content-addressed* layers are hosted in S3 buckets in AWS
  - Why mirror only [Content-Addresed][content-addressed] Image APIs?
    - Image Layers (which are content-addressed) in particular are the vast majority of bandwidth == costs. Serving them from one cloud to another is expensive.
    - Content Addressed APIs are relatively safe to serve from untrusted or less-secured hosts, since all major clients confirming the result matches the requested digest
- We detect client IP address and match it to Cloud Providers we use in order to serve content-addressed API calls from the most local and cost-effective copy
- Other API calls (getting and listing tags etc) are redirected to the regional upstream source-of-truth Artifact Registries

This allows us to offload substantial bandwidth securely, while not having to fully
implement a registry from scratch and maintaining the project's existing security
controls around the GCP source registries (implemented elsewhere in the Kubernetes project).
We only re-route some content-addressed storage requests to additional hosts.

Clients do still need to either pull by digest (`registry.k8s.io/foo@sha256:...`),
verify sigstore signatures, or else trust that the redirector instance is secure,
but not the S3 buckets or additional future content-addressed storage hosts.

We maintain relatively tight control over the production redirector instance and
the source registries. Very few contributors have access to this infrastructure.

We have a development instance at https://registry-sandbox.k8s.io which is
*not* supported for any usage outside of the development of this project and 
may or may not be working at any given time. 
Changes will be deployed there before we deploy to production, and be exercised
by a subset of Kubernetes' own CI.

Mirroring content-addressed content to object storage is currently handled by [`cmd/geranos`](./../geranos).

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

[artifact-registry]: https://cloud.google.com/artifact-registry
[content-addressed]: https://en.wikipedia.org/wiki/Content-addressable_storage
