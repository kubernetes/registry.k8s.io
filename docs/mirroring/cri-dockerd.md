# Mirroring With cri-dockerd

# Identifying Images to Mirror

If you're using [cri-dockerd] as a Kubernetes [CRI] implementation, cri-dockerd
uses the ["pause" image][pause] from Kubernetes to implement pods.
You may want to mirror this critical image to your own host.

To find the default pause image you can run:
```
cri-dockerd --help | grep pod-infra-container-image
```

## Mirroring Images

See our general list of [mirroring options](./README.md#Mirroring-Images)

# Using Mirrored Images

For pause you can set the `--pod-infra-sandbox-container-image` flag.
https://github.com/Mirantis/cri-dockerd/blob/47abdab2c31ffc8b54c826063760662590ef3801/config/options.go#L107

cri-dockerd does not appear to support configuring mirrors more generally.


[cri-dockerd]: https://github.com/Mirantis/cri-dockerd/
[pause]: https://www.ianlewis.org/en/almighty-pause-container
[CRI]: https://kubernetes.io/docs/concepts/architecture/cri/
