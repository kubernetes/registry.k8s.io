# registry.k8s.io

This project implements the backend for registry.k8s.io, Kubernetes's container
image registry.

Known user-facing issues will be pinned at the top of [our issue tracker][issues].

For details on the implementation see [cmd/archeio](./cmd/archeio/README.md)

The community deployment configs are documented at in the k8s.io repo with
the rest of the community infra deployments, but primarily 
[here][infra-configs].

For publishing to registry.k8s.io, refer to [the docs][publishing] at in k8s.io 
under `registry.k8s.io/`.

## Stability

registry.k8s.io is GA and we ask that all users migrate from k8s.gcr.io as
soon as possible.

However, unequivocally: **DO NOT depend on the implementation details of this registry.**

**Please note that there is NO uptime SLA as this is a free, volunteer managed
service**. We will however do our best to respond to issues and the system is
designed to be reliable and low-maintenance. If you need higher uptime guarantees
please consider [mirroring] images to a location you control.

**Other than `registry.k8s.io` serving an [OCI][distribution-spec] compliant registry:
API endpoints, IP addresses, and backing services used 
are subject to change at _anytime_ as new resources become available or as otherwise
necessary.**

**If you need to allow-list domains or IPs in your environment, we highly recommend
[mirroring] images to a location you control instead.**

The Kubernetes project is currently sending traffic to GCP and AWS
thanks to their donations but we hope to redirect traffic to more
sponsors and their respective API endpoints in the future to keep the project
sustainable.

See Also:
- Pinned issues in our [our issue tracker][issues]
- Our [debugging guide][debugging] for identifying and resolving or reporting issues
- Our [mirroring guide][mirroring] for how to mirror and use mirrored Kubernetes images

## Privacy

This project abides by the Linux Foundation privacy policy, as documented at
https://registry.k8s.io/privacy

## Background

Previously all of Kubernetes' image hosting has been out of gcr.io ("Google Container Registry").

We've incurred significant egress traffic costs from users on other cloud providers
in particular in doing so, severely limiting our ability to use the 
GCP credits from Google for purposes other than hosting end-user downloads.

We're now moving to shift all traffic behind a community controlled domain, so
we can quickly implement cost-cutting measures like serving the bulk of the traffic
for AWS-users from AWS-local storage funded by Amazon, or potentially leveraging
other providers in the future.

For additional context on why we did this and what we're changing about kubernetes images
see: https://kubernetes.io/blog/2022/11/28/registry-k8s-io-faster-cheaper-ga

Essentially, this repo implements the backend sources for the steps outlined there.

For a talk with more details see: ["Why We Moved the Kubernetes Image Registry"](https://www.youtube.com/watch?v=9CdzisDQkjE)

## Community, discussion, contribution, and support

Learn how to engage with the Kubernetes community on the [community page](http://kubernetes.io/community/).

You can reach the maintainers of this project at:

- [Slack](http://slack.k8s.io/) in channel `#sig-k8s-infra`
- [Mailing List](https://groups.google.com/forum/#!forum/kubernetes-sig-k8s-infra)

### Code of conduct

Participation in the Kubernetes community is governed by the [Kubernetes Code of Conduct](code-of-conduct.md).

[owners]: https://git.k8s.io/community/contributors/guide/owners.md
[Creative Commons 4.0]: https://git.k8s.io/website/LICENSE
[distribution-spec]: https://github.com/opencontainers/distribution-spec
[publishing]: https://git.k8s.io/k8s.io/registry.k8s.io#managing-kubernetes-container-registries
[infra-configs]: https://github.com/kubernetes/k8s.io/tree/main/infra/gcp/terraform
[mirroring]: ./docs/mirroring/README.md
[debugging]: ./docs/debugging.md
[issues]: https://github.com/kubernetes/registry.k8s.io/issues
