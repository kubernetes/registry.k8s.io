# Debugging issues with registry.k8s.io

The registry.k8s.io is a Kubernetes container images registry that behaves generally like an [OCI](https://github.com/opencontainers/distribution-spec) compliant registry. Since registry.k8s.io is a proxy routing traffic to the closest available source, you will need connectivity to several domains to download images. It is also best for performance to create your own registry mirror.

When you are debugging issues, make sure you run these commands on the node that is attempting to run images. Things may be working fine on your laptop, but not on the Kubernetes node.

<!--TODO: identify what this looks like on s3 etc.-->
> **Note**
>
> If you see a [403 error][http-403] like `Your client does not have permission to get URL`,
> this error is not specific to the Kubernetes project / registry.k8s.io and
> you need to work with your cloud vendor / service provider to get unblocked
> by GCP.
>
> Please file an issue with your provider, the Kubernetes project does not
> control this and it is not specific to us.
>
> If you are a cloud vendor / service provider please contact
> [Google Edge Network](mailto:noc@google.com)

## Verify DNS resolution

You may use the `dig` or `nslookup` command to validate DNS resolution of the registry.k8s.io domain or any domain it references. For example, running `dig registry.k8s.io` should return an answer that contains:

```log
;; ANSWER SECTION:
registry.k8s.io.	3600	IN	A	34.107.244.51
```

If you cannot successfully resolve a domain, check your DNS configuration, often configured in your resolv.conf file.

## Verify HTTP connectivity

You may use `curl` or `wget` to validate HTTP connectivity. For example, running `curl -v https://registry.k8s.io/v2/` should return an answer that contains:

```log
< HTTP/2 200
< docker-distribution-api-version: registry/2.0
< x-cloud-trace-context: ca200d1c5a504b919e999b0cf80e3b71
< date: Fri, 17 Mar 2023 09:13:18 GMT
< content-type: text/html
< server: Google Frontend
< content-length: 0
< via: 1.1 google
< alt-svc: h3=":443"; ma=2592000,h3-29=":443"; ma=2592000
<
```

If do not have HTTP connectivity, check your firewall or HTTP proxy settings.

## Verify image repositories and tags

You may use `crane` or `oras` to validate the available tags in the registry. You may also use [https://explore.ggcr.dev/?repo=registry.k8s.io](https://explore.ggcr.dev/?repo=registry.k8s.io) to verify the existence of an image repository and tag, but these commands will verify your node can access them. For example, the `crane ls registry.k8s.io/pause` or `oras repo tags registry.k8s.io/pause` will return:

```log
0.8.0
1.0
2.0
3.0
3.1
3.2
3.3
3.4.1
3.5
3.6
3.7
3.8
3.9
go
latest
sha256-7031c1b283388d2c2e09b57badb803c05ebed362dc88d84b480cc47f72a21097.sig
sha256-9001185023633d17a2f98ff69b6ff2615b8ea02a825adffa40422f51dfdcde9d.sig
test
test2
```

## Verify image pulls

Since registry.k8s.io proxies image components to the nearest source, you should validate the ability to pull images. The ability to pull images should be tested on the machine running the image which will often be a node in your Kubernetes cluster. The location where you pull image components from depends on the source IP address of the node.

You may use commands such as `crane`, `oras`, `crictl` or `docker` to verify the ability to pull an image. If you run the command `crane pull --verbose registry.k8s.io/pause:3.9 pause.tgz` for example, you will see it query registry.k8s.io first and then at least two other domains to download the image. If things are working correctly and you ran `crane pull --verbose registry.k8s.io/pause:3.9 pause.tgz 2>&1 | grep 'GET https'` (from Colorado):

```log
2023/03/17 04:45:48 --> GET https://registry.k8s.io/v2/
2023/03/17 04:45:48 --> GET https://registry.k8s.io/v2/pause/manifests/3.9
2023/03/17 04:45:48 --> GET https://us-west1-docker.pkg.dev/v2/k8s-artifacts-prod/images/pause/manifests/3.9
2023/03/17 04:45:48 --> GET https://registry.k8s.io/v2/pause/manifests/sha256:8d4106c88ec0bd28001e34c975d65175d994072d65341f62a8ab0754b0fafe10
2023/03/17 04:45:48 --> GET https://us-west1-docker.pkg.dev/v2/k8s-artifacts-prod/images/pause/manifests/sha256:8d4106c88ec0bd28001e34c975d65175d994072d65341f62a8ab0754b0fafe10
2023/03/17 04:45:49 --> GET https://registry.k8s.io/v2/pause/blobs/sha256:e6f1816883972d4be47bd48879a08919b96afcd344132622e4d444987919323c
2023/03/17 04:45:49 --> GET https://prod-registry-k8s-io-us-west-2.s3.dualstack.us-west-2.amazonaws.com/containers/images/sha256%3Ae6f1816883972d4be47bd48879a08919b96afcd344132622e4d444987919323c
2023/03/17 04:45:49 --> GET https://registry.k8s.io/v2/pause/blobs/sha256:61fec91190a0bab34406027bbec43d562218df6e80d22d4735029756f23c7007 [body redacted: omitting binary blobs from logs]
2023/03/17 04:45:49 --> GET https://prod-registry-k8s-io-us-west-2.s3.dualstack.us-west-2.amazonaws.com/containers/images/sha256%3A61fec91190a0bab34406027bbec43d562218df6e80d22d4735029756f23c7007 [body redacted: omitting binary blobs from logs]
```

From my location, the pull command accesses registry.k8s.io, us-west1-docker.pkg.dev and prod-registry-k8s-io-us-west-2.s3.dualstack.us-west-2.amazonaws.com. You will need to have DNS and HTTP access to these domains on your node to pull images.

It's also possible to run these commands on your node if you don't have SSH access by using `kubectl run`:

```sh
kubectl run --rm -it crane --restart=Never --image=gcr.io/go-containerregistry/crane --overrides='{"spec": {"hostNetwork":true}}' -- pull --verbose registry.k8s.io/pause:3.9 /dev/null
```

## Example Logs

If there are problems accessing registry.k8s.io, you are likely to see failures starting pods with an `ErrImagePull` status. The `kubectl describe pod` command may give you more details:

```log
  Warning  Failed     2s (x2 over 16s)  kubelet            Failed to pull image "registry.k8s.io/pause:3.10": rpc error: code = NotFound desc = failed to pull and unpack image "registry.k8s.io/pause:3.10": failed to resolve reference "registry.k8s.io/pause:3.10": registry.k8s.io/pause:3.10: not found
  Warning  Failed     2s (x2 over 16s)  kubelet            Error: ErrImagePull
```

If you were to check your kubelet log for example, you might see (with something like `journalctl -xeu kubelet`):

```log
Mar 17 11:33:05 kind-control-plane kubelet[804]: E0317 11:33:05.192844     804 kuberuntime_manager.go:862] container &Container{Name:my-puase-container,Image:registry.k8s.io/pause:3.10,Command:[],Args:[],WorkingDir:,Ports:[]ContainerPort{},Env:[]EnvVar{},Resources:ResourceRequirements{Limits:ResourceList{},Requests:ResourceList{},},VolumeMounts:[]VolumeMount{VolumeMount{Name:kube-api-access-4bv66,ReadOnly:true,MountPath:/var/run/secrets/kubernetes.io/serviceaccount,SubPath:,MountPropagation:nil,SubPathExpr:,},},LivenessProbe:nil,ReadinessProbe:nil,Lifecycle:nil,TerminationMessagePath:/dev/termination-log,ImagePullPolicy:IfNotPresent,SecurityContext:nil,Stdin:false,StdinOnce:false,TTY:false,EnvFrom:[]EnvFromSource{},TerminationMessagePolicy:File,VolumeDevices:[]VolumeDevice{},StartupProbe:nil,} start failed in pod my-pause_default(4b642716-1dba-44d4-833b-1eccd6b6ca7a): ErrImagePull: rpc error: code = NotFound desc = failed to pull and unpack image "registry.k8s.io/pause:3.10": failed to resolve reference "registry.k8s.io/pause:3.10": registry.k8s.io/pause:3.10: not found
```

You may see similar errors in the containerd log (with something like `journalctl -xeu containerd`):

```log
Mar 17 11:33:04 kind-control-plane containerd[224]: time="2023-03-17T11:33:04.658642300Z" level=info msg="PullImage \"registry.k8s.io/pause:3.10\""
Mar 17 11:33:05 kind-control-plane containerd[224]: time="2023-03-17T11:33:05.189169600Z" level=info msg="trying next host - response was http.StatusNotFound" host=registry.k8s.io
Mar 17 11:33:05 kind-control-plane containerd[224]: time="2023-03-17T11:33:05.191777300Z" level=error msg="PullImage \"registry.k8s.io/pause:3.10\" failed" error="rpc error: code = NotFound desc = failed to pull and unpack image \"registry.k8s.io/pause:3.10\": failed to resolve reference \"registry.k8s.io/pause:3.10\": registry.k8s.io/pause:3.10: not found"
```

## Example issues

- https://github.com/kubernetes/registry.k8s.io/issues/137#issuecomment-1376574499
- https://github.com/kubernetes/registry.k8s.io/issues/174#issuecomment-1467646821
- https://github.com/kubernetes-sigs/kind/issues/1895#issuecomment-1468991168
- https://github.com/kubernetes/registry.k8s.io/issues/174#issuecomment-1467646821
- https://github.com/kubernetes/registry.k8s.io/issues/154#issuecomment-1435028502

[http-403]: https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/403
