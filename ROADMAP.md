# Roadmap

## User facing roadmap

### pipeline API v1 support

- Add support for v1 API when it will be publish
- Support for fields, …

### triggers API v1beta1 support

- Add support for v1beta1 triggers API

### results integration

Evaluate and provide an integration with the
[results](https://github.com/tektoncd/results) project.

### chains integration

Evaluate and provide an integration with the
[chains](https://github.com/tektoncd/chains) project.

### operator integration

Evaluate and provide an integration with the
[operator](https://github.com/tektoncd/operator) project.

### Integration with Hub

Update support for the hub in the CLI as the hub evolves.

### OCI Bundle support

Implement
[TEP-0031](https://github.com/tektoncd/community/blob/main/teps/0031-tekton-bundles-cli.md)
that propose to add support for managing Tekton OCI bundles through =.

### Consistency across all CLI resources for list, delete, and describe subcommands

- We should work to make sure each resource introduced to the CLI always features these three subcommands

### Filtering for list and delete commands

- We should provide filtering solutions for all list/delete commands associated with CLI

### Local to cluster build

- From a user working directory, be able to run a task or pipeline with the content of this working directory
  - Use volume (pvc, …) with `workspace` support
  - Create / Update volume and start the pipeline
- Related to : Binary (local) input type
  [#924](https://github.com/tektoncd/pipeline/issues/924)
- Discovery on creation
  - Similar to oc new-app
  - Detect stuff, create task, …

### Source to Pipeline resources

- be able from the cli auto generate some Pipeline/tasks from a source
  code folder detecting what kind of source code it is (i.e: go, java,
  python, ruby, rust, project etc...) and automatically apply the
  standard from task the catalog.

## Technical roadmap

### More Regular Release Process to Test Our Release Process

- A common problem with releases has been unexpected issues during release attempts:
  - https://github.com/tektoncd/cli/pull/673 (0.7.1)
  - https://github.com/tektoncd/cli/pull/757 (0.8.0)
- We should have a way of more regularly testing the CLI release process pipeline

### Re-evaluate plug-in execution model

- Easier support for experiments
  - `tkn install …` could detect and install tekton using different mechanisms (operator on openshift or if it detects OLM, etc…)
    - Could be a separate project (hosted by tektoncd/operator or elsewhere)
  - Initial catalog / oci integration ?
    - Helper to author a catalog task / resource ? (tkn catalog task init … ?)
    - Integration with Tekdoc
- How to drive usage and feedback from it ?
