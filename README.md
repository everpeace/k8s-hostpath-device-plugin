# k8s-hostpath-device-plugin

This is a very thin device plugin which just exposed a host path to a container.

This can be seen as non limited resources, things like "capabilitites" of a host. However, due to kubernetes/kubernetes#59380, current device plugin(v1beta1) doesn't support unlimited extended resources. Currently a hack would be to set the number of resources advertised by the device plugin to a very high mumber.

Why not using `hostPath` volume??  The question is natural.  But, `hostPath` volume assumes all the nodes in the cluster have the `hostPath`.  What happened if only part of the nodes serve the host path? Assume user pod's spec declares a host path volume like below:

```yaml
kind: Pod
spec:
...
  volumes:
  - name: vol
    hostPath:
      path: /mnt/volume
```

Then, how to make sure that the pod is scheduled to a node which has host path `/mnt/volume`??

This device plugin solve this by serving the exising hostpath as extended resource like this:

```yaml
kind: Pod
...
    resources:
      limits:
        # the resource mounts hostPath=/sample to contianerPath=/sample
        hostpath-device.k8s.io/sample: 1
```

Thus, it can guarantee that the pod will be scheduled to a node which has the host path.

## Build

```shell
docker build . -t k8s-hostpath-device-plugin
```

## Deploy

First you edit [`example/config.yaml`](example/config.yaml).  Then, you can deploy the device plugin:

```shell
kustomize build example/ | kubectl apply -f -
```

## Try with Kind

```shell
# create a kind cluster
# - create kind cluster
# - prepare hostpath=/sample
# - install cert-manager
$ make dev-cluster

# deploy
# - k8s-hostpath-device-plugin
# - its webhook
$ make dev-deploy
```

```shell
# create pod requesting 'hostpath-device.k8s.io/sample=1'
$ cat << EOT | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: test-hostpath-sample
spec:
  restartPolicy: Never
  containers:
  - name: ctr
    image: busybox
    command:
    - sh
    - -c
    - |
      set -ex
      ls -al /sample
      cat /sample/hello
    resources:
      limits:
        # the resource mounts hostPath=/sample to contianerPath=/sample
        hostpath-device.k8s.io/sample: 1
EOT

# you can see pod can access to /sample from a container
$ kubectl logs test-hostpath-sample
+ ls -al /sample
total 12
drwxr-xr-x    2 root     root          4096 Aug 27 16:43 .
drwxr-xr-x    1 root     root          4096 Aug 27 17:00 ..
-rw-r--r--    1 root     root             6 Aug 27 16:43 hello
+ cat /sample/hello
hello
+ exit 0

# cleanup
$ make dev-clean
```

## How to make a release

`release` make target tags a commit and push the tag to `origin`. Release process will run in GitHub Action.

```shell
$ RELEASE_TAG=$(git semv patch) # next patch version.  bump major/minor version if necessary
$ make release RELEASE=true RELEASE_TAG=${RELEASE_TAG}
```

## License

MIT License.  See [LICENSE](LICENSE) file.
