# Roadmap

## User facing roadmap

### Support for Pipeline v1

- Add support for tektoncd/pipeline v1
- Warn usage of PipelineResource in v1beta1 (no error but warn that it
  is not supported aka will work but can be broken in a later release)

### Remove unsupported APIs

- Remove tektoncd/pipeline v1alpha1
- Remove tektoncd/triggers v1alpha1 (later, most likely in 2023)

### Advanced integration of chains

Today, the `chains` sub-command support a relative subset of what
chains can provide.

### Results integration

The `results` project comes up with it's own cli
[`tkn-results`](https://github.com/tektoncd/results/tree/main/tools/tkn-results). We
should aim to integrate that cli directly into `tkn`.

As a "possible next step", we could exploration transparent behavior
between list (from cluster & results). This means, when listing
Pipelines or Task, we would *transparently* query the k8s API as well
as the result API to get informations.

### Operator integration

User should be able to get `tkn` and install, upgrade and manage the
operator lifecycle directly from it. *This should help adoption as well*.

*There is an item in the Operator ROADMAP about this*.

### Pipeline Breakpoint integration [#1476](https://github.com/tektoncd/cli/issues/1476)

Once the Breakpoint feature is stable, `tkn` should provide a way to
easily enter into debug mode and into a task waiting to be "debugged".

### Integrate Catlin with tkn.

Catlin is a command-line tool that Lints Tekton Resources and
Catalogs. It will be it's own cli, but we should aim to integrate
`tkn` with it (or at least part of it), be it through the extension
mechanism or not.

See https://github.com/tektoncd/community/issues/636.

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

### Local to cluster build

- From a user working directory, be able to run a task or pipeline with the content of this working directory
  - Use volume (pvc, …) with `workspace` support
  - Create / Update volume and start the pipeline
- Related to : Binary (local) input type
  [#924](https://github.com/tektoncd/pipeline/issues/924)
- Discovery on creation
  - Similar to oc new-app
  - Detect stuff, create task, …

### Source to Pipeline/Task

- be able from the cli auto generate some Pipeline/tasks from a source
  code folder detecting what kind of source code it is (i.e: go, java,
  python, ruby, rust, project etc...) and automatically apply the
  standard from task the catalog.

### Local Execution

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
