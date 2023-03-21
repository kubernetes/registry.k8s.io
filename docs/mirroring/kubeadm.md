# Mirroring with Kubeadm

## Identifying Images To Mirror

You can use `kubeadm config images list` to get a list of images kubeadm requires.

For more see:
https://kubernetes.io/docs/reference/setup-tools/kubeadm/kubeadm-config/#cmd-config-images-list

## Mirroring Images

See our general list of [mirroring options](./README.md#Mirroring-Images)

## Using Mirrored Images

To use kubeadm with mirrored images, you can pass the `--image-repository` flag
to [`kubeadm init`][kubeadm init] or the `imageRepository` field of [kubeadm config].

[kubeadm init]: https://kubernetes.io/docs/reference/setup-tools/kubeadm/kubeadm-init/
[kubeadm config]: https://kubernetes.io/docs/reference/setup-tools/kubeadm/kubeadm-init/#config-file
