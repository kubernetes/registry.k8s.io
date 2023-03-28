# Mirroring with Cluster API

## Identifying Images To Mirror

You can use [`clusterctl`](https://cluster-api.sigs.k8s.io/clusterctl/overview.html) to list the images used by Cluster API and the Cluster API provider in use:
```
clusterctl init list-images --infrastructure <infrastructure-provider>
```
For more details see: 
https://cluster-api.sigs.k8s.io/clusterctl/commands/additional-commands.html#clusterctl-init-list-images

## Mirroring Images

See our general list of [mirroring options](./README.md#Mirroring-Images)

## Using Mirrored Images

To use Cluster API with mirrored images, you can configure clusterctl to use image overrides.

For more details see:
https://cluster-api.sigs.k8s.io/clusterctl/configuration.html#image-overrides
