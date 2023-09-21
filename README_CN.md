# registry.k8s.io 中文说明

本项目实现了 registry.k8s.io 的后端，这是 Kubernetes 的容器镜像注册表。

已知的面向用户的问题将在[我们的问题跟踪器][issues]顶部固定。

有关实现的详细信息，请参阅 cmd/archeio

社区部署配置在 k8s.io 仓库中与其他社区基础设施部署一起记录，主要在[这里][infra-configs]。

有关发布到 registry.k8s.io 的信息，请参考 k8s.io 下 registry.k8s.io 的[文档][publishing]。

## 稳定性

registry.k8s.io 已经 GA，我们要求所有用户尽快从 k8s.gcr.io 迁移过来。

然而，毫无疑问：不要依赖于此注册表的实现细节。

**请注意，由于这是一个免费的、由志愿者管理的服务，所以没有正常运行时间SLA**。然而，我们会尽力回应问题，系统设计上也是可靠且低维护的。

如果你需要更高的正常运行时间保证，请考虑将镜像[mirroring]同步到你控制的位置。

**除了 registry.k8s.io 提供一个符合[OCI][distribution-spec]规范的注册表外：

API端点、IP地址和使用的后端服务可能会随着新资源的可用或者其他必要情况随时变化。**

**如果你需要在你的环境中允许列出域名或IP，我们强烈建议将镜像[mirroring]同步到你控制的位置。**

Kubernetes 项目目前正在向 GCP 和 AWS 发送流量

感谢他们的捐赠，但我们希望将来能够将流量重定向到更多

赞助商和他们各自的 API 端点，以保持项目

可持续性。

另请参阅：

- 我们[问题跟踪器][issues]中固定的问题

- 我们的[调试指南][debugging]，用于识别和解决或报告问题

- 我们的[镜像指南][mirroring]，说明如何镜像和使用镜像化的Kubernetes镜像

## 隐私

本项目遵守 Linux Foundation 隐私政策，文档在

https://registry.k8s.io/privacy

## 背景

以前，Kubernetes的所有镜像托管都是在gcr.io（"Google Container Registry"）。

我们因此从其他云提供商那里产生了大量出口流量成本，

特别是在这样做时，严重限制了我们使用

来自Google的GCP信用额度用于除托管最终用户下载之外的其他目标。

我们现在正在将所有流量转移到社区控制的域名后面，所以

我们可以快速实施节省成本措施，比如为AWS用户提供大部分流量

从由亚马逊资助的AWS本地存储中获取，或者可能利用未来其他提供商。

关于为什么我们做这个以及我们对kubernetes镜像做了什么改变，

请参阅：https://kubernetes.io/blog/2022/11/28/registry-k8s-io-faster-cheaper-ga

本质上，这个仓库实现了那里概述步骤的后端源码。

有关更多详细信息，请参阅：["Why We Moved the Kubernetes Image Registry"](https://www.youtube.com/watch?v=9CdzisDQkjE)

## 社区、讨论、贡献和支持

学习如何参与Kubernetes社区，请访问社区页面。

你可以在以下位置联系到本项目的维护者：

- [Slack](http://slack.k8s.io/) in channel `#sig-k8s-infra`
- [Mailing List](https://groups.google.com/forum/#!forum/kubernetes-sig-k8s-infra)

### 行为准则

参与Kubernetes社区的行为受到Kubernetes行为准则的约束。

[owners]: https://git.k8s.io/community/contributors/guide/owners.md
[Creative Commons 4.0]: https://git.k8s.io/website/LICENSE
[distribution-spec]: https://github.com/opencontainers/distribution-spec
[publishing]: https://git.k8s.io/k8s.io/registry.k8s.io#managing-kubernetes-container-registries
[infra-configs]: https://github.com/kubernetes/k8s.io/tree/main/infra/gcp/terraform
[mirroring]: ./docs/mirroring/README.md
[debugging]: ./docs/debugging.md
[issues]: https://github.com/kubernetes/registry.k8s.io/issues
