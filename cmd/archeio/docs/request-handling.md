# Request Handling

Requests to archeio follows the following flow:

1. If it's a request for `/`: redirect to our wiki page about the project
2. If it's not a request for `/` and does not start with `/v2/`: 404 error
3. For registry API requests, all of which start with `/v2/`:
  - If it's not a blob request: redirect to Upstream Registry
  - If it's not a known AWS IP: redirect to Upstream Registry
  -  If it's a known AWS IP AND HEAD request for the layer succeeeds in S3: redirect to S3
  -  If it's a known AWS IP AND HEAD fails: redirect to Upstream Registry

Currently the `Upstream Registry` is https://k8s.gcr.io.

Or in chart form:
```mermaid
flowchart TD

A(Does the request path start with /v2/?) -->|No, it is not a registry API call| B(Is the request for /?)
B -->|No, it is an unknown path| D[Serve 404 error]
B -->|Yes| E[Serve redirect to registry wiki page]
A -->|Yes, it is a registry API call| F(Is it a blob request?)
F -->|No| G[Serve redirect to Upstream Registry https://k8s.gcr.io]
F -->|Yes, it matches known blob request format| H(Is the client IP known to be from AWS?)
H -->|No| G
H -->|Yes| I(Does the blob exist in S3?<br/>Check by way of cached HEAD on the bucket we've selected based on client IP.)
I -->|No| G
I -->|Yes| J[Redirect to blob copy in S3]
```
