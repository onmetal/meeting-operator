# Meeting-operator
**Project status: *beta*** Not all planned features are completed.

## Overview
Meeting-operator provides deployment and management of [Jitsi](https://jitsi.org/) 
and related components (only [Etherpad](https://etherpad.org/) right now).
The Meeting-operator includes the following features:

* **Kubernetes Custom Resources**: Use Kubernetes custom resources to deploy and manage Jitsi, Etherpad,
  and related components.

## Prerequisites
```
kubectl create secret generic jitsi-config \
--from-literal=JICOFO_COMPONENT_SECRET=2 \
--from-literal=JICOFO_AUTH_PASSWORD=1 \
--from-literal=JVB_AUTH_PASSWORD=1
```
## Install
You can use helm for deploy meeting-operator in the cluster.
```
helm install meeting-operator ./deploy/helm/meeting-operator
```
### Manual install
1. crd
```
make install
```
or
```
kubectl apply -f config/crd/bases/*
```
2. custom resources
```
kubectl apply -f config/sample/_v1alpha1_etherpad.yaml
kubectl apply -f config/sample/_v1alpha1_jitsi.yaml
```

If you need to change default values, you should check values.yml

#### Examples
Folder ``` config/samples``` contain crds, ingress, config examples. It's enough to 
start up with jitsi.
```
 ll config/samples
 _v1alpha1_etherpad.yaml
 _v1alpha1_jitsi.yaml
 ingress.yaml
 jitsi-config.yaml
 kustomization.yaml

```
