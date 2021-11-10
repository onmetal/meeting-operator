# Documentation Setup

The documentation of the [machine-operator](https://github.com/onmetal/machine-operator) project is written primarily using Markdown.
All documentation related content can be found in the `/docs` folder. New content also should be added there.
[MkDocs](https://www.mkdocs.org/) and [MkDocs Material](https://squidfunk.github.io/mkdocs-material/) are then used to render the contents of the `/docs` folder to have a more user-friendly experience when browsing the projects' documentation.

## Extending API

### Adding a new version

One should not modify API once it got to be used.

Instead, in order to introduce breaking changes to the API, a new API version should be created.

First, move the existing controller to the different file, as generator will try to put a new controller
into the same location, e.g.

    mv controllers/machine/controller.go controllers/machine/v1alpha1_controller.go

After that, add a new API version:

    operator-sdk create api --version v1alpha2 --kind Machine --resource --controller

Do modifications in a new CR, add a new controller to `main.go`.

Following actions should be applied to other parts of project:
- regenerate code and configs with `make generate install`
- add a client to client set for the new API version
- alter Helm chart with new CRD spec

### Deprecating old APIs

Since there is no version deprecation marker available now, old APIs may be deprecated with `kustomize` patches

Describe deprecation status and deprecation warning in patch file, e.g. `crd_patch.yaml`

```
- op: add
  path: "/spec/versions/0/deprecated"
  value: true
- op: add
  path: "/spec/versions/0/deprecationWarning"
  value: "This API version is deprecated. Check documentation for migration instructions."
```

Add patch instructions to `kustomization.yaml`

```
patchesJson6902:
  - target:
      version: v1
      group: apiextensions.k8s.io
      kind: CustomResourceDefinition
      name: machines.machine.onmetal.de
    path: crd_patch.yaml
```

When you are ready to drop the support for the API version, give CRD a `+kubebuilder:skipversion` marker,
or just remove it completely from the code base.

This includes:
- API objects
- client
- controller

## Versioning

We use [SemVer](http://semver.org/) for versioning. For the versions available, see the [link to tags on this repository](https://github.com/onmetal/machine-operator/tags).

