# Mirroring With Containerd

# Identifying Images to Mirror

If you're using containerd as a Kubernetes [CRI] implementation, containerd
uses the ["pause" image][pause] from Kubernetes in every pod.
You may want to mirror this critical image to your own host.

The version used by default can be found by `containerd config default | grep sandbox_image`.

Containerd config is generally at `/etc/containerd/config.toml` and may contain
a customized "sandbox image" rather than the default, for more details see:
https://github.com/containerd/containerd/blob/main/docs/cri/config.md#registry-configuration

## Mirroring Images

See our general list of [mirroring options](./README.md#Mirroring-Images)

# Using Mirrored Images


You may want to configure `sandbox_image` under `[plugins."io.containerd.grpc.v1.cri"]`
to point to your own mirrored image for "pause".

`containerd` also supports configuring mirrors for registry hosts.

If you're using containerd with Kubernetes, see:
https://github.com/containerd/containerd/blob/main/docs/cri/config.md#registry-configuration

If you're using containerd directly, see:
https://github.com/containerd/containerd/blob/main/docs/hosts.md

[containerd]: https://containerd.io/
[pause]: https://www.ianlewis.org/en/almighty-pause-container
[CRI]: https://kubernetes.io/docs/concepts/architecture/cri/
