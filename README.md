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

To install the Kwok Operator, follow these steps:

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

3. Apply the Kubernetes manifests:
   ```shell
   kubectl apply -k config/default
   ```

## Usage

To use the Kwok Operator, follow these steps:

1. Define a NodePool custom resource (CR) with your desired configuration. Example:

   ```yaml
    apiVersion: kwok.sigs.k8s.io/v1beta1
    kind: NodePool
    metadata:
    labels:
        app.kubernetes.io/instance: nodepool-sample
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

## Troubleshooting

If you encounter any issues with the Kwok Operator, please check the following:

- Ensure that the Kwok is infrastructure properly configured and accessible from the Kubernetes cluster. 
  https://kwok.sigs.k8s.io/docs/user/kwok-in-cluster/
- Check the logs of the Kwok Operator pod for any error messages under namespace kwok-operaotr.