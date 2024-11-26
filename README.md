Kubernetes Mutator Webhook: NodeSelector
----------------------------------------

A solution for locking Pods which reside in a specific namespace to a sepcific node group.

## Why 

We are waiting for EKS to ship support for `scheduler.alpha.kubernetes.io/node-selector`.

```yaml
apiVersion: v1
kind: Namespace
metadata:
 name: your-namespace
 annotations:
   scheduler.alpha.kubernetes.io/node-selector: env=test
```

Link to the issue can be found here:

[https://github.com/aws/containers-roadmap/issues/304](https://github.com/aws/containers-roadmap/issues/304)

## Usage

```yaml
apiVersion: v1
kind: Namespace
metadata:
 name: your-namespace
 annotations:
   k8s-mutate-nodeselector.skpr.io/namespace: env=test
```
