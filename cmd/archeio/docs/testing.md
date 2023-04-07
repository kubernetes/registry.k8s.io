# Testing

## Unit Tests

Archeio has unit tests covering 100% of application code (`cmd/archeio/app/...`) 
and general packages (`pkg/...`). The only code without 100% coverage is in main.

These are standard Go unit tests. In addition to typical unit tests with granular
methods, we also have unit tests covering the HTTP Handlers and full 
[request handling flow](./request-handling.md).

These tests run on every pull request and must pass before merge.

**This level of coverage must be maintained**, it is imperative that we have robust
testing in this project that may soon serve all Kubernetes project image downloads.
We automatically enforce 100% code coverage for archeio sources.

Coverage results are visible by clicking the `pull-registry-test` context link
at the bottom of the pull request.

Coverage results can be viewed locally by `make test` + open `bin/all-filtered.html`.

## Integration Tests

Package `main` code not covered by unit tests is covered by integration tests.

Integration tests are written as a go unit test named with prefix `TestIntegration`
in the `main` package which allows `make unit` to skip them.

The integration tests run the actual application `main()`, and pull image(s)
through a running instance using [crane].

`make integration` runs only integration tests.

The integration tests are able to exploit running against a local instance without
a loadbalancer etc in front and fake the client IP address to test provider-IP
codepaths.

These tests run on every pull request in `pull-registry-test` and must pass before merge.

## E2E Testing

Changes to archeio are auto-deployed to the registry-sandbox.k8s.io staging intance
and NOT to registry.k8s.io. registry.k8s.io serves stable releases.

### e2e tests

We have quick and cheap e2e results using real clients in `make e2e-test`
and `make e2e-test-local`. We run `make e2e-test` against the staging instance.

These are limited to clients we can run locally and in containerize CI
without privilege escalation (e.g. [crane] again).

These run immediately in the staging deploy jobs and
continuously against the staging instance here:

https://testgrid.k8s.io/sig-k8s-infra-registry#registry-sandbox-e2e-gcp
https://testgrid.k8s.io/sig-k8s-infra-registry#registry-sandbox-e2e-aws

`make e2e-test-local` runs against PRs to ensure the e2e tests themselves work
and must pass before merge.

### Cluster e2e Testing

The instance at registry-sandbox.k8s.io has [kops] cluster CI running in AWS 
pointed at it to validate pulling from AWS. The kops clusters run in random
regions.

This E2E CI should be consulted before promoting code to stable release + registry.k8s.io.

Results are visible in [testgrid] at: https://testgrid.k8s.io/sig-k8s-infra-registry#Summary

The Kubernetes project itself has substantial assorted CI usage of the production instance
and many CI jobs that primarily exist for other purposes will alert us if pulling from it fails.
This includes many variations on "real" clusters and image build clients.

[crane]: https://github.com/google/go-containerregistry/blob/main/cmd/crane/README.md
[kops]: https://github.com/kubernetes/kops
[testgrid]: https://github.com/GoogleCloudPlatform/testgrid
