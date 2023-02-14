## Install

### Requirements
Following tools are required to work on that package.

- [make](https://www.gnu.org/software/make/) - to execute build goals
- [golang](https://golang.org/) - to compile the code
- [kind](https://kind.sigs.k8s.io/) or access to k8s cluster - to deploy and test operator
- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) - to interact with k8s cluster via CLI
- [kustomize](https://kustomize.io/) - to generate deployment configs
- [kubebuilder](https://book.kubebuilder.io) - framework to build operators
- [operator framework](https://operatorframework.io/) - framework to maintain project structure
- [helm](https://helm.sh/) - to work with helm charts

If you have to build Docker images on your host,
you also need to have [Docker](https://www.docker.com/) or its alternative installed.

### Prepare environment

If you have access to the docker registry and k8s installation that you can use for development purposes, you may skip
corresponding steps.

Otherwise, create a local instance of k8s.
```
    kind create cluster
    Creating cluster "kind" ...
    ✓ Ensuring node image (kindest/node:v1.20.2) 🖼
    ✓ Preparing nodes 📦
    ✓ Writing configuration 📜
    ✓ Starting control-plane 🕹️
    ✓ Installing CNI 🔌
    ✓ Installing StorageClass 💾
    Set kubectl context to "kind-kind"
    You can now use your cluster with:

    kubectl cluster-info --context kind-kind

    Thanks for using kind! 😊
```

## Prerequisites
```
kubectl create -f config/samples/jitsi-config.yaml
```
## Install
You can use helm for deploy meeting-operator in the cluster.
```
helm install meeting-operator ./deploy/helm/meeting-chart
```
or
```
helm repo add onmetal https://onmetal.github.io/meeting-operator/
helm install meeting-operator onmetal/meeting-chart
```
### Use the published chart

Add this to your `Chart.yaml`
```yaml
dependencies:
  - name: meeting-chart
    version: '0.20.1'
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
### Install non-jisti components.
kubectl apply -f config/samples/_v1alpha2_etherpad.yaml
kubectl apply -f config/samples/_v1alpha2_whiteboard.yaml

### Install jitsi components.
kubectl apply -f config/samples/_v1beta1_jitsi_jibri.yaml
kubectl apply -f config/samples/_v1beta1_jitsi_jicofo.yaml
kubectl apply -fconfig/samples/_v1beta1_jitsi_jigasi.yaml
kubectl apply -f config/samples/_v1beta1_jitsi_jvb.yaml
kubectl apply -fconfig/samples/_v1beta1_jitsi_prosody.yaml
kubectl apply -fconfig/samples/_v1beta1_jitsi_web.yaml
```

If you need to change default values, you should check values.yml

### Examples
Folder ``` config/samples``` contain crds, ingress, config examples. It's enough to
start up with jitsi.
```
 ll config/samples
_v1alpha1_jas.yaml
_v1alpha2_etherpad.yaml
_v1alpha2_whiteboard.yaml
_v1beta1_jitsi_jibri.yaml
_v1beta1_jitsi_jicofo.yaml
_v1beta1_jitsi_jigasi.yaml
_v1beta1_jitsi_jvb.yaml
_v1beta1_jitsi_prosody.yaml
_v1beta1_jitsi_web.yaml
ingress.yaml
jitsi-config.yaml
kustomization.yaml
telegraf-cm.yaml
```

### Release Guide

- meeting operator
1. Make your changes to the operator in a feature branch
2. Create, review, test and merge the PR
3. Review the automatically created draft release, release notes are generated automatically from the PRs summary
4. Publish the draft release
5. A repository tag (e.g. `meeting-operator-v0.50.0`) will be created
6. Based on the tag above, a docker image will be built, tagged and pushed. The tag will drop the `meeting-operator-v`-prefix, so the docker image will be called `meeting-operator:0.50.0`

- helm chart
1. Make your changes to the chart
2. Edit the [`Chart.yml`](/deploy/helm/meeting-chart/Chart.yaml). The `version` must always be incremented, the `appVersion` must match a valid meeting operator version
3. Commit and push to default branch, e.g. by merging a PR
4. The automated helm release will create a tag with the specified helm chart version, e.g. `meeting-chart-v0.20.1`
