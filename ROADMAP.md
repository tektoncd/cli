# Roadmap

## User facing roadmap

### V1beta1 support and GA

- Add support for v1beta1 and later on GA
- Warn usage of PipelineResource in v1beta1 (no error but warn that it is not supported aka will work but can be broken in a later release)
- Support for workspace, result, …

### Improve e2e Testing Coverage for CLI

- We should do an assessment of what scenarios we have covered for e2e testing and figure out where additional testing is needed

### Consistency across all CLI resources for list, delete, and describe subcommands

- We should work to make sure each resource introduced to the CLI always features these three subcommands

### Filtering for list and delete commands

- We should provide filtering solutions for all list/delete commands associated with CLI

### Define and carry out strategy around creation/deletion for the CLI:

- There are two issues open around this:
  - https://github.com/tektoncd/cli/issues/574
  - https://github.com/tektoncd/cli/issues/575
- There have been requests for a create command for the CLI similar to kubectl create/apply
- We should support one of the above strategies, or we will not support creation/updating like kubectl create/apply -f moving forward
- If we do not support a -f option, we should remove the create -f commands from pipeline/task/resource

### Catalog integration

- Related to the OCI experiment
  ([#2137](https://github.com/tektoncd/pipeline/issues/2137)) and the
  catalog / hub, integrate better with the/a catalog
  - Allow searching catalog for a Task, Pipeline, …
  - Allow importing a task from the catalog into the cluster

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
