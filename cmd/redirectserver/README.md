# redirectserver

redirectserver will serve non-image kubernetes artifacts, on artifacts.k8s.io.

This is still under development, but the code is derived from the code serving registry.k8s.io

Broadly the idea is that we continue to serve hash files (and other security "roots") directly from artifacts.k8s.io.
However, for the (larger) binary downloads themselves, we can redirect to a mirror that is closer to the user.

By placing mirrors in AWS S3 and other cloud providers, users get faster downloads and the cost to serve artifacts.k8s.io
is lower, because "local" downloads are typically charged at a lower rate.

For a rough TLDR of the current design:

- Content is server in the existing GCS bucket
- Mirrors are hosted in S3 buckets in AWS
- AWS clients are detected by client IP address and redirect to a local S3 bucket copy
*only* when requesting content, *not* hash files, gpg keys etc.
- Other requests are redirected to existing serving infrastructure

For more detail see:
- [docs/request-handling.md](./docs/request-handling.md)
- [docs/testing.md](./docs/testing.md)

Development is at https://artifacts-sandbox.k8s.io which is *not* supported for
any usage outside of the development of this project and may or may not be
working at any given time. Changes will be deployed there before we deploy
to production, and be exercised by a subset of Kubernetes' own CI.

For AWS client-IP matching, see [`pkg/net/cidrs/aws`](./../../pkg/net/cidrs/aws)
