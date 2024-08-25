# Kwok Operator

## Overview

The Kwok Operator is a Kubernetes operator designed to create virtual nodes within a Kubernetes cluster using Kwok, by applying custom resource definitions (CRDs) for node pools.

## Tested
The Kwok operator test on top the following kuberenetes flavors 
  - Vanila 
  - EKS ( Elastic Kubernetes Service )
  - GKE ( Goole Kubernetes Engine )
  - AKS ( Azure Kubernetes Service ) 
  - RKE1 
  - RKE2 
  - Openshift 
  - Kind 

## Features

- Automatically creates virtual nodes on Kwok infrastructure.
- Utilizes Kubernetes Custom Resource Definitions (CRDs) for easy configuration.
- Provides seamless integration with Kubernetes clusters.

## Prerequisites

Before using the Kwok Operator, ensure you have the following prerequisites installed:

- tested on Kubernetes cluster (version 1.24 or later)
- Kwok infrastructure set up and accessible from the cluster
- kubectl CLI installed and configured to access the Kubernetes cluster

## Installation

To install Kwok CRDs and the Kwok Operator, follow these steps:

1. Clone the Kwok Operator repository:

   ```shell
   git clone git@github.com:run-ai/kwok-operator.git
   ```
2. enter to kwok-operator directory
   ```shell
   cd kwok-operator
   ```
3. make sure kwok installed in your cluster from the URL: https://kwok.sigs.k8s.io/docs/user/kwok-in-cluster/
   or install by the script install_kwok.sh
   ```shell
   ./install_kwok.sh
   ```

3. Apply the kwok-operator Kubernetes manifests:
   ```shell
   kubectl apply --server-side -k config/default
   ```
   or 
   ```shell
   kubectl apply --server-side -f https://github.com/run-ai/kwok-operator/releases/download/1.0.0/kwok-operator.yaml
   ```
## Usage

To use the Kwok Operator to provision nodes, follow these steps:

1. Define a NodePool custom resource (CR) with your desired configuration. Example:

```yaml
apiVersion: kwok.sigs.run-ai.com/v1beta1
kind: NodePool
metadata:
  labels:
    app.kubernetes.io/name: nodepool
    app.kubernetes.io/instance: nodepool-sample
    app.kubernetes.io/part-of: kwok-operator
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/created-by: kwok-operator
  name: nodepool-sample
spec:
  nodeCount: 15
  nodeTemplate:
    apiVersion: v1
    metadata:
      annotations:
        node.alpha.kubernetes.io/ttl: "0"
      labels:
        kubernetes.io/hostname: kwok-node
        kubernetes.io/role: agent
        type: kwok
    spec: {}
    status:
      allocatable:
        cpu: 32
        memory: 256Gi
        pods: 110
      capacity:
        cpu: 32
        memory: 256Gi
        pods: 110
      nodeInfo:
        architecture: amd64
        bootID: ""
        containerRuntimeVersion: ""
        kernelVersion: ""
        kubeProxyVersion: fake
        kubeletVersion: fake
        machineID: ""
        operatingSystem: linux
        osImage: ""
        systemUUID: ""
      phase: Running
   ```

2. Apply the NodePool CR to your Kubernetes cluster:

   ```shell
   kubectl apply -f path/to/your/nodepool.yaml
   ```

3. Monitor the status of the created virtual nodes using:
   ```shell
   kubectl get nodes 
   ```

## Configuration

The Kwok Operator can be configured via the NodePool CR.
   ```shell
   kubectl edit nodepool nodepool-sample
   ```

----
To use the Kwok Operator to manage deployments and run the pods on top the nodes you provisioned above, follow these steps:
1. ensure the namespace is exist  
2. Define a DeploymentPool custom resource (CR) with your desired configuration. Example:
```yaml
apiVersion: kwok.sigs.run-ai.com/v1beta1
kind: DeploymentPool
metadata:
  labels:
    app.kubernetes.io/name: deploymentpool
    app.kubernetes.io/instance: deploymentpool-sample
    app.kubernetes.io/part-of: kwok-operator
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/created-by: kwok-operator
  name: deploymentpool-sample
  namespace: default
spec:  
  deploymentTemplate:
    apiVersion: apps/v1 
    metadata:
      name: kwok-operator
      labels:
        app.kubernetes.io/name: deployment
        app.kubernetes.io/instance: deployment-sample
        app.kubernetes.io/part-of: kwok-operator
        app.kubernetes.io/managed-by: kustomize
        app.kubernetes.io/created-by: kwok-operator
    spec:
      replicas: 3
      selector:
        matchLabels: 
          app.kubernetes.io/name: deployment
          app.kubernetes.io/instance: deployment-sample
          app.kubernetes.io/part-of: kwok-operator
          app.kubernetes.io/managed-by: kustomize
          app.kubernetes.io/created-by: kwok-operator
      template:
        metadata:
          labels:
            app.kubernetes.io/name: deployment
            app.kubernetes.io/instance: deployment-sample
            app.kubernetes.io/part-of: kwok-operator
            app.kubernetes.io/managed-by: kustomize
            app.kubernetes.io/created-by: kwok-operator
        spec:
          containers:
          - image: nginx
            name: nginx
          restartPolicy: Always
   ```
---
To use the Kwok Operator to manage pods on top the nodes you provisioned above, follow these steps:
1. ensure the namespace is exist
2. Define a PodPool custom resource (CR) with your desired configuration. Example:
```yaml
apiVersion: kwok.sigs.run-ai.com/v1beta1
kind: PodPool
metadata:
  labels:
    app.kubernetes.io/name: podpool
    app.kubernetes.io/instance: podpool-sample
    app.kubernetes.io/part-of: kwok-operator
    app.kubernetes.io/created-by: kwok-operator
  name: podpool-sample
  namespace: default
spec:
  podCount: 5
  podTemplate:
    metadata:
      name: kwok-operator
      labels:
        app.kubernetes.io/name: pod
        app.kubernetes.io/instance: pod-sample
        app.kubernetes.io/part-of: kwok-operator
        app.kubernetes.io/managed-by: kustomize
        app.kubernetes.io/created-by: kwok-operator
    spec:
      containers:
      - image: nginx
        name: nginx
      restartPolicy: Always
```
Added in version 0.0.5
To use the Kwok Operator to manage jobs on top the nodes you provisioned above, follow these steps:
1. ensure the namespace is exist
2. Define a JobPool custom resource (CR) with your desired configuration. Example:
```yaml
apiVersion: kwok.sigs.run-ai.com/v1beta1
kind: JobPool
metadata:
  labels:
    app.kubernetes.io/name: jobpool
    app.kubernetes.io/instance: jobpool-sample
    app.kubernetes.io/part-of: kwok-operator
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/created-by: kwok-operator
  name: jobpool-sample
spec:
  jobCount: 5
  jobTemplate:
    metadata:
      name: kwok-operator
      labels:
        app.kubernetes.io/name: job
        app.kubernetes.io/instance: job-sample
        app.kubernetes.io/part-of: kwok-operator
        app.kubernetes.io/managed-by: kustomize
        app.kubernetes.io/created-by: kwok-operator
    spec:
      template:
        metadata:
          labels:
            app.kubernetes.io/name: job
            app.kubernetes.io/instance: job-sample
            app.kubernetes.io/part-of: kwok-operator
            app.kubernetes.io/managed-by: kustomize
            app.kubernetes.io/created-by: kwok-operator
        spec:
          containers:
          - name: job
            image: busybox
            command: ["sh", "-c", "echo Hello, Kubernetes! && sleep 3600"]
          restartPolicy: Never
```
Added in version 0.0.7
To use the Kwok Operator to manage Daemonset on top the nodes you provisioned above, follow these steps:
1. ensure the namespace is exist
2. Define a DaemonsetPool custom resource (CR) with your desired configuration. Example:
```yaml
apiVersion: kwok.sigs.run-ai.com/v1beta1
kind: DaemonsetPool
metadata:
  labels:
    app.kubernetes.io/name: daemonsetpool
    app.kubernetes.io/instance: daemonsetpool-sample
    app.kubernetes.io/part-of: kwok-operator
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/created-by: kwok-operator
  name: daemonsetpool-sample
  namespace: default
spec:
  daemonsetTemplate:
    metadata:
      name: kwok-operator
      labels:
        app.kubernetes.io/name: daemonset
        app.kubernetes.io/instance: daemonset-sample
        app.kubernetes.io/part-of: kwok-operator
        app.kubernetes.io/managed-by: kustomize
        app.kubernetes.io/created-by: kwok-operator
    spec:
      selector:
        matchLabels: 
          app.kubernetes.io/name: deployment
          app.kubernetes.io/instance: deployment-sample
          app.kubernetes.io/part-of: kwok-operator
          app.kubernetes.io/managed-by: kustomize
          app.kubernetes.io/created-by: kwok-operator
      template:
        metadata:
          labels:
            app.kubernetes.io/name: daemonset
            app.kubernetes.io/instance: daemonset-sample
            app.kubernetes.io/part-of: kwok-operator
            app.kubernetes.io/managed-by: kustomize
            app.kubernetes.io/created-by: kwok-operator
        spec:
          containers:
          - image: nginx
            name: nginx
          restartPolicy: Always
```
## Troubleshooting 

If you encounter any issues with the Kwok Operator, please check the following:

- Ensure that the Kwok is infrastructure properly configured and accessible from the Kubernetes cluster. 
  https://kwok.sigs.k8s.io/docs/user/kwok-in-cluster/
- Check the logs of the Kwok Operator pod for any error messages under namespace kwok-operaotr.

From version 1.0.0 the Kwok Operator is able to manage Statefuleset. To include PVC on top of the nodes you have provisioned above, follow these steps:
1. ensure the namespace is exist
2. ensure that storage class is installed and working as expected in the cluster 
2. Define a statefulesetPool custom resource (CR) with your desired configuration. Example:
```yaml
apiVersion: kwok.sigs.run-ai.com/v1beta1
kind: StatefulsetPool
metadata:
  labels:
    app.kubernetes.io/name: statefulsetpool
    app.kubernetes.io/instance: statefulsetpool-sample
    app.kubernetes.io/part-of: kwok-operator
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/created-by: kwok-operator
  name: statefulsetpool-sample
spec:
  createPV: true
  StatefulsetTemplate:
    metadata:
      labels:
        app.kubernetes.io/name: statefulsetpool
        app.kubernetes.io/instance: statefulsetpool-sample
        app.kubernetes.io/part-of: kwok-operator
        app.kubernetes.io/managed-by: kustomize
        app.kubernetes.io/created-by: kwok-operator
    spec:
      serviceName: "nginx"
      replicas: 45
      selector:
        matchLabels:
          app: nginx
      template:
        metadata:
          labels:
            app: nginx
        spec:
          containers:
          - name: nginx
            image: registry.k8s.io/nginx-slim:0.21
            ports:
            - containerPort: 80
              name: web
            volumeMounts:
            - name: www
              mountPath: /usr/share/nginx/html
      volumeClaimTemplates:
      - metadata:
          name: www
        spec:
          accessModes: [ "ReadWriteOnce" ]
          resources:
            requests:
              storage: 1Gi
```
## Contributing

Contributions to the Kwok Operator are welcome! To contribute, please follow the guidelines outlined in [CONTRIBUTING.md](./CONTRIBUTING.md).

---

Feel free to customize and expand upon this template to suit your specific needs and preferences!