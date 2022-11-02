# Testing

## Unit Tests

redirectserver has unit tests covering 100% of application code (`cmd/archeio/app/...`)
and general packages (`pkg/...`). The only code without 100% coverage is in main.

These are standard Go unit tests. In addition to typical unit tests with granular
methods, we also have unit tests covering the HTTP Handlers and full 
[request handling flow](./request-handling.md).


**This level of coverage must be maintained**, it is imperative that we have robust
testing in this project that may soon serve all Kubernetes project artifact downloads.
TODO: this should be enforced by CI. Currently it is enforced by reviewers.

Coverage results are visible by clicking the `pull-oci-proxy-test` context link
at the bottom of the pull request.

Coverage results can be viewed locally by `make test` + open `bin/all-filtered.html`.

## Integration Tests

Package `main` code not covered by unit tests is covered by integration tests.

Integration tests are written as a go unit test named with prefix `TestIntegration`
in the `main` package which allows `make unit` to skip them.

The integration tests run the actual application `main()`, and pull image(s)
through a running instance using [crane].

`make integration` runs only integration tests.

Because our CI runs primarily in GCP, this currently covers pulling from outside AWS.

## E2E Testing

Changes to archeio are auto-deployed to artifacts-sandbox.k8s.io and NOT to
artifacts.k8s.io. artifacts.k8s.io serves stable releases.

The instance at artifacts-sandbox.k8s.io has [kops] cluster CI running in AWS
pointed at it to validate pulling from AWS. The kops clusters run in random
regions.

This E2E CI should be consulted before promoting code to stable release + artifacts.k8s.io.

Results are visible in [testgrid] at: https://testgrid.k8s.io/sig-k8s-infra-oci-proxy#Summary

[kops]: https://github.com/kubernetes/kops
[testgrid]: https://github.com/GoogleCloudPlatform/testgrid
