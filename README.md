# Kubernetes Admission Webhook Server
[![Version](https://img.shields.io/github/v/release/liangrog/admission-webhook-server)](https://github.com/liangrog/admission-webhook-server/releases)
[![GoDoc](https://godoc.org/github.com/liangrog/admission-webhook-server?status.svg)](https://godoc.org/github.com/liangrog/admission-webhook-server)
![](https://github.com/liangrog/admission-webhook-server/workflows/Release/badge.svg)

---

API server providing webhook endpoints for Kubernetes admission controller to mutate objects. 

Currently it can handle mutating `nodeSelector` based on namespaces. This same functionality exists in standard Kubernetes cluster installation if enabled. However it's not enabled in EKS. 

The server can be easily extended by adding more handlers for different mutations needs.

The repo also includes a Helm chart for easy deployment to your Kubernetes cluster.

---

## Installation
Firstly you need to determine what your SSL CN is. The self-signed ssl CN follows the format of `[service name].[namespace].svc`. For example, the default service name is `admission-webhook` (It can be changed in helm value). You want to deploy to namespace tools. The CN will be `admission-webhook.tools.svc`. Below steps will use this CN for demo purpose.

Secondly you need to update helm value `namespaceAnnotationsToProcess` and `ignorePodsWithLabels` in `chart/values.yaml` so it can use the value to mutate the pods. 

Note: below example using Helm v3. However the chart is compatible with helm version older than v3.

```sh
$ git clone https://github.com/trilogy-group/admission-webhook-server
$ cd admission-webhook-server
$
$ sh ssl.sh admission-webhook.tools.svc
$
$ cd chart
$ helm install . --name  admission-webhook-server --namespace tools
```

OR

You can also generate YAML files using `helm template` and commit those files to `gitops` or perform `kubectl apply`

```sh
$ git clone https://github.com/trilogy-group/admission-webhook-server
$ cd admission-webhook-server
$
$ sh ssl.sh admission-webhook.tools.svc
$
$ cd chart
$ helm template . -n admission-webhook-server --namespace tools --output-dir admission-webhook-server
```

## Helm 
The following table lists the configuration parameters for the helm chart.

| Parameter  | Description  | Default  | 
|---|---|---|
| nameOverride  | Override general resource name   |   |
| basePathOverride  | Url base path   | mutate  | 
| podNodesSelectorPathOverride  | Url sub path for podnodesselector  | pod-nodes-selector  |
| namespaceAnnotationsToProcess  | Confiruation for which annotations to be read from namespace of pod and assign its value as nodeselectors for pod. The annotations to process are seperated by comma (,) Examples: x/y,a/b  |  devflows/node-selector |
| blacklistedNamespaces | Configuration for disallowing podnodeselector from adding node selectors to pods if it belongs to one of these namespaces. Namespaces are separated by comma (,) : Example: ns-1,ns-2 | devflows-utilities-master |
| ignorePodsWithLabels | Configuration for disallowing podnodeselector from adding node selectors to pods. Specify labels to match (one out of all match would do) here : Example: l=m | fargate=true,eventing.knative.dev/broker=default |
| service.name  | Name of the service. It forms part of the ssl CN  | admission-webhook  |
| service.annotations  | Anotation for the service  | {} |
| replicas | Number of replicas  | 1  |
| strategy.type  | Type of update strategy  | RollingUpdate  |
| image  | Docker image name  | liangrog/admission-webhook-server  |
| imageTag  | Docker image tag  | latest  |
| imagePullPolicy  | Docker image pull policy  | Always  |
