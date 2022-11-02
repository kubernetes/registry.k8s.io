# Request Handling

Requests to artifacts.k8s.io follow the following flow:

1. If it's a request for `/`: redirect to our wiki page about the project
1. If it's a request for `/privacy`: redirect to linux foundation privacy policy page
1. If it's not a request for `/` or `/privacy` and does not start with `/binaries/`: 404 error
1. For binary requests, all of which start with `/binaries/`:
    - If it's security sensitive: serve it directly.
    - If it's not a known AWS IP: redirect to the canonical location
    -  If it's a known AWS IP AND HEAD request for the layer succeeeds in S3: redirect to S3
    -  If it's a known AWS IP AND HEAD fails: redirect to canonical location

Currently the `canonical location` is https://artifacts.k8s.io.  This will obviously need to be changed before we stand this up on artifacts.k8s.io.


# Security considerations

We assume that users trust artifacts.k8s.io, they need not trust the mirrors as long as they are verifying hashes.

For this to work, we identify hashes (and gpg keys) and serve them directly from artifacts.k8s.io without a redirect.  They are also typically small (<1 KB), so this doesn't cost us much vs redirecting.