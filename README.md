# Kubernetes Admission Webhook Server
[![Version](https://img.shields.io/github/v/release/liangrog/admission-webhook-server)](https://github.com/liangrog/admission-webhook-server/releases)[![GoDoc](https://godoc.org/github.com/liangrog/admission-webhook-server?status.svg)](https://godoc.org/github.com/liangrog/admission-webhook-server)
![](https://github.com/liangrog/admission-webhook-server/workflows/Release/badge.svg)

---

API server providing webhook endpoints for Kubernetes admission controller to mutate objects. 

Currently it can handle mutating `nodeSelector` based on namespaces. This same functionality exists in standard Kubernetes cluster installation if enabled. However it's not enabled in EKS. 

The server can be easily extended by adding more handlers for different mutations needs.

The repo also includes a Helm chart for easy deployment to your Kubernetes cluster.

---

## Installation
Firstly you need to determine what your SSL CN is. The self-signed ssl CN follows the format of `[service name].[namespace].svc`. For example, the default service name is `admission-webhook` (It can be changed in helm value). You want to deploy to namespace tools. The CN will be `admission-webhook.tools.svc`. Below steps will use this CN for demo purpose.

Note: below example using Helm v3. However the chart is compatible with helm version older than v3.

```sh
$ git clone https://github.com/liangrog/admission-webhook-server
$ cd admission-webhook-server
$
$ sh ssl.sh admission-webhook.tools.svc
$
$ cd chart
$ helm install admission-webhook-server .
```

## Helm 
The following table lists the configuration parameters for the helm chart.

| Parameter  | Description  | Default  | 
|---|---|---|
| nameOverride  | Override general resource name   |   |
| basePathOverride  | Url base path   | mutate  | 
| podNodesSelectorPathOverride  | Url sub path for podnodesselector  | pod-nodes-selector  |
| podNodesSelectorConfig  | Configuration for podnodesselector. The namespace and labels are set here following the format: namespace: key=label,key=label; namespace2: key=label. Multiple namespaces seperate by ";". Example: devel: node-role.kubernetes.io/development=true, beta.kubernetes.io/instance-type=t3.large  |   |
| service.name  | Name of the service. It forms part of the ssl CN  | admission-webhook  |
| service.annotations  | Anotation for the service  | {} |
| replicas | Number of replicas  | 1  |
| strategy.type  | Type of update strategy  | RollingUpdate  |
| image  | Docker image name  | liangrog/admission-webhook-server  |
| imageTag  | Docker image tag  | latest  |
| imagePullPolicy  | Docker image pull policy  | Always  |
