# Mirroring With Containerd

# Identifying Images to Mirror

If you're using containerd as a Kubernetes [CRI] implementation, containerd
uses the ["pause" image][pause] from Kubernetes in every pod.
You may want to mirror this critical image to your own host.

The version used by default can be found by `containerd config default | grep sandbox_image`.

# Using Mirrored Images

`containerd` supports configuring mirrors for registry hosts.

If you're using containerd with Kubernetes, see:
https://github.com/containerd/containerd/blob/main/docs/cri/config.md#registry-configuration

Additionally, you may want to configure `sandbox_image` under `[plugins."io.containerd.grpc.v1.cri"]`
to point to your own mirrored image for "pause".

If you're using containerd directly, see:
https://github.com/containerd/containerd/blob/main/docs/hosts.md


[pause]: https://www.ianlewis.org/en/almighty-pause-container
[CRI]: https://kubernetes.io/docs/concepts/architecture/cri/
