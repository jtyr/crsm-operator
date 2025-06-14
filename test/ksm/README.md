# CRSM Operator testing

This folder contains files that can help to setup CRSM Operator and run manual
tests on it.

## Requirements

- [K3D](https://k3d.io)
- [Helm](https://helm.sh)
- [Helmfile](https://github.com/helmfile/helmfile)
- `kubectl`

## Usage

All the following commands should be run from the root of the Git repository.

Create K3D cluster:

```shell
k3d cluster create --config test/ksm/k3d.yaml
```

Install CRSM CRD:

```shell
make install
```

Run the operator:

```shell
make run
```

Create CRSM resource:

```shell
kubectl apply -f test/ksm/crsm-resource-version.yaml
```

Check the generated `ConfigMap`:

```shell
kubectl get cm -o yaml crsm-test
```

Install KSM:

```shell
helmfile apply -f test/ksm/helmfile.yaml
```

Check that KSM exposes the custom metric:

```shell
kubectl run --rm -it --image curlimages/curl --restart=Never test -- curl kube-state-metrics:8080/metrics
```

Delete the K3D cluster:

```shell
k3d cluster delete --config test/ksm/k3d.yaml
```

## Author

Jiri Tyr
