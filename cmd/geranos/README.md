# Geranos

γερανός (geranós) is Greek for "crane"

This binary is a tool based on [crane] which is used to copy image layers
from registries to object storage for backing [archeio](./../archeio)

Currently it only supports Google Container Registry / Artifact Registry to S3.

Other object stores can be easily added, but container registry portability is blocked
on https://github.com/opencontainers/distribution-spec/issues/222

[crane]: https://github.com/google/go-containerregistry/tree/main/cmd/crane