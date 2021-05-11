# Meeting-operator
**Project status: *beta*** Not all planned features are completed.

## Overview
Meeting-operator provides deployment and management of [Jitsi](https://jitsi.org/) 
and related components: 
* [Etherpad](https://etherpad.org/) - Etherpad is a real-time collaborative editor scalable to thousands of simultaneous real time users. It provides full data export capabilities, and runs on your server, under your control.
* [Excalidraw](https://excalidraw.com/) - Virtual whiteboard for sketching hand-drawn like diagrams.

The Meeting-operator includes the following features:

* **Kubernetes Custom Resources**: Use Kubernetes custom resources to deploy and manage Jitsi,
  and related components.

## Prerequisites
```
kubectl create -f config/samples/jitsi-config.yaml
```
## Install
You can use helm for deploy meeting-operator in the cluster.
```
helm install meeting-operator ./deploy/helm/meeting-operator
```
### Use the published chart

Add this to your `Chart.yaml`
```yaml
dependencies:
  - name: meeting-operator
    version: '0.3.11'
    repository: 'https://onmetal.github.io/meeting-operator'
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
kubectl apply -f config/samples/_v1alpha1_etherpad.yaml
kubectl apply -f config/samples/_v1alpha1_whiteboard.yaml 
kubectl apply -f config/samples/_v1alpha1_jitsi.yaml
```

If you need to change default values, you should check values.yml

#### Examples
Folder ``` config/samples``` contain crds, ingress, config examples. It's enough to 
start up with jitsi.
```
 ll config/samples
 _v1alpha1_etherpad.yaml
 _v1alpha1_jitsi.yaml
 _v1alpha_whiteboard.yaml
 ingress.yaml
 jitsi-config.yaml
 kustomization.yaml
```

TODO: docs for folders

## Release Guide

1. Make you Changes
2. Edit the [`Chart.yml`](/deploy/helm/meeting-operator/Chart.yaml) and update the `version` and `appVersion` accordingly. 
3. Commit and push (to master, e.g. by merging the PR)
4. The automated Helm release will create a tag with the specified version
5. After the Tag is pushed the Docker Release will create the Images specified by the tag. 
