# Tekton CLI Releases

## Release Frequency

Tekton CLI follows the Tekton community [release policy][release-policy]
as follows:

- Versions are numbered according to semantic versioning: `vX.Y.Z`
- A new release is produced on a monthly basis
- Four releases a year are chosen for [long term support (LTS)](https://github.com/tektoncd/community/blob/main/releases.md#support-policy).
  All remaining releases are supported for approximately 1 month (until the next
  release is produced)
    - LTS releases take place in January, April, July and October every year
    - The first Tekton CLI LTS release will be **v0.30.0** in January 2023
    - Releases happen towards the middle of the month, but the exact date may vary,
      depending on week-ends and readiness

Tekton CLI produces nightly builds, publicly available on
`gcr.io/tekton-nightly`. 

### Transition Process

Before release v0.28 Tekton CLI has worked on the basis of an undocumented
support period of four months, which will be maintained for the releases between
v0.26 and v0.27.

## Release Process

Read about releasing the Tekton CLI in the [release process documentation]
[tekton-release-process].

Further documentation available:

- [Tekton resources][tekton-releases-docs]
- Standard for [release notes][release-notes-standards]

## Releases

### v0.38 (LTS)

- **Latest Release**: [v0.38.1][v0-38-1] (2024-09-10) ([docs][v0-38-1-docs])
- **Initial Release**: [v0.38.0][v0-38-0] (2024-08-21) ([docs][v0-38-0-docs])
- **End of Life**: 2025-08-20

### v0.37 (LTS)

- **Latest Release**: [v0.37.0][v0-37-0] (2024-05-13) ([docs][v0-37-0-docs])
- **Initial Release**: [v0.37.0][v0-37-0] (2024-05-13) ([docs][v0-37-0-docs])
- **End of Life**: 2025-05-12

### v0.36

- **Latest Release**: [v0.36.0][v0-36-0] (2024-03-21) ([docs][v0-36-0-docs])
- **Initial Release**: [v0.36.0][v0-36-0] (2024-03-21) ([docs][v0-36-0-docs])
- **End of Life**: 2024-04-20

### v0.35 (LTS)

- **Latest Release**: [v0.35.2][v0-35-2] (2024-03-25) ([docs][v0-35-2-docs])
- **Initial Release**: [v0.35.0][v0-35-0] (2024-02-07) ([docs][v0-35-0-docs])
- **End of Life**: 2025-02-06

### v0.34

- **Latest Release**: [v0.34.0][v0-34-0] (2023-01-12) ([docs][v0-34-0-docs])
- **Initial Release**: [v0.34.0][v0-34-0] (2024-01-12) ([docs][v0-34-0-docs])
- **End of Life**: 2024-02-11

### v0.33 (LTS)

- **Latest Release**: [v0.33.0][v0-33-0] (2023-11-16) ([docs][v0-33-0-docs])
- **Initial Release**: [v0.33.0][v0-33-0] (2023-11-16) ([docs][v0-33-0-docs])
- **End of Life**: 2024-12-15

### v0.32 (LTS)

- **Latest Release**: [v0.32.2][v0-32-2] (2023-10-20) ([docs][v0-32-2-docs])
- **Initial Release**: [v0.32.0][v0-32-0] (2023-09-08) ([docs][v0-32-0-docs])
- **End of Life**: 2024-09-07

### v0.31 (LTS)

- **Latest Release**: [v0.31.3][v0-31-3] (2023-10-21) ([docs][v0-31-3-docs])
- **Initial Release**: [v0.31.0][v0-31-0] (2023-05-10) ([docs][v0-31-0-docs])
- **End of Life**: 2024-05-10

### v0.30 (LTS)

- **Latest Release**: [v0.30.2][v0-30-2] (2023-10-21) ([docs][v0-30-2-docs])
- **Initial Release**: [v0.30.0][v0-30-0] (2023-03-27) ([docs][v0-30-0-docs])
- **End of Life**: 2024-03-27

### v0.29

- **Latest Release**: [v0.29.1][v0-29-1] (2022-02-02) ([docs][v0-29-1-docs])
- **Initial Release**: [v0.29.0][v0-29-0] (2022-01-05) ([docs][v0-29-0-docs])
- **End of Life**: 2023-02-04
- **Patch Releases**: [v0.29.1][v0-29-1]

### v0.28

- **Latest Release**: [v0.28.0][v0-28-0] (2022-11-25) ([docs][v0-28-0-docs])
- **Initial Release**: [v0.28.0][v0-28-0] (2022-11-25)
- **End of Life**: 2023-01-24
- **Patch Releases**: [v0.28.0][v0-28-0]

### v0.27

- **Latest Release**: [v0.27.0][v0-27-0] (2022-10-11) ([docs][v0-27-0-docs])
- **Initial Release**: [v0.27.0][v0-27-0] (2022-10-11)
- **End of Life**: 2023-02-10
- **Patch Releases**: [v0.27.0][v0-27-0]

### v0.26

- **Latest Release**: [v0.26.0][v0-26-0] (2022-09-02) ([docs][v0-26-0-docs])
- **Initial Release**: [v0.26.0][v0-26-0] (2022-09-02)
- **End of Life**: 2023-01-08
- **Patch Releases**: [v0.26.0][v0-26-0]

### v0.25

- **Latest Release**: [v0.25.0][v0-25-0] (2022-07-22) ([docs][v0-25-0-docs])
- **Initial Release**: [v0.25.0][v0-25-0] (2022-07-22)
- **End of Life**: 2022-11-21
- **Patch Releases**: [v0.25.0][v0-25-0]

## End of Life Releases

Older releases are EOL and available on [GitHub][tekton-cli-releases].


[release-policy]: https://github.com/tektoncd/community/blob/main/releases.md
[tekton-chains]: https://github.com/tektoncd/chains
[tekton-cli-releases]: https://github.com/tektoncd/cli/releases
[tekton-releases-docs]: tekton/README.md
[release-notes-standards]:
    https://github.com/tektoncd/community/blob/main/standards.md#release-notes
[tekton-release-process]: RELEASE_PROCESS.md

[v0-38-1]: https://github.com/tektoncd/cli/releases/tag/v0.38.1
[v0-38-0]: https://github.com/tektoncd/cli/releases/tag/v0.38.0
[v0-37-0]: https://github.com/tektoncd/cli/releases/tag/v0.37.0
[v0-36-0]: https://github.com/tektoncd/cli/releases/tag/v0.36.0
[v0-35-2]: https://github.com/tektoncd/cli/releases/tag/v0.35.2
[v0-35-0]: https://github.com/tektoncd/cli/releases/tag/v0.35.0
[v0-34-0]: https://github.com/tektoncd/cli/releases/tag/v0.34.0
[v0-33-0]: https://github.com/tektoncd/cli/releases/tag/v0.33.0
[v0-32-2]: https://github.com/tektoncd/cli/releases/tag/v0.32.2
[v0-32-0]: https://github.com/tektoncd/cli/releases/tag/v0.32.0
[v0-31-3]: https://github.com/tektoncd/cli/releases/tag/v0.31.3
[v0-31-0]: https://github.com/tektoncd/cli/releases/tag/v0.31.0
[v0-30-2]: https://github.com/tektoncd/cli/releases/tag/v0.30.2
[v0-30-0]: https://github.com/tektoncd/cli/releases/tag/v0.30.0
[v0-29-1]: https://github.com/tektoncd/cli/releases/tag/v0.29.1
[v0-29-0]: https://github.com/tektoncd/cli/releases/tag/v0.29.0
[v0-28-0]: https://github.com/tektoncd/cli/releases/tag/v0.28.0
[v0-27-0]: https://github.com/tektoncd/cli/releases/tag/v0.27.0
[v0-26-0]: https://github.com/tektoncd/cli/releases/tag/v0.26.0
[v0-25-0]: https://github.com/tektoncd/cli/releases/tag/v0.25.0

[v0-38-1-docs]: https://github.com/tektoncd/cli/tree/v0.38.1/docs
[v0-38-0-docs]: https://github.com/tektoncd/cli/tree/v0.38.0/docs
[v0-37-0-docs]: https://github.com/tektoncd/cli/tree/v0.37.0/docs
[v0-36-0-docs]: https://github.com/tektoncd/cli/tree/v0.36.0/docs
[v0-35-2-docs]: https://github.com/tektoncd/cli/tree/v0.35.2/docs
[v0-35-0-docs]: https://github.com/tektoncd/cli/tree/v0.35.0/docs
[v0-34-0-docs]: https://github.com/tektoncd/cli/tree/v0.34.0/docs
[v0-33-0-docs]: https://github.com/tektoncd/cli/tree/v0.33.0/docs
[v0-32-2-docs]: https://github.com/tektoncd/cli/tree/v0.32.2/docs
[v0-32-0-docs]: https://github.com/tektoncd/cli/tree/v0.32.0/docs
[v0-31-3-docs]: https://github.com/tektoncd/cli/tree/v0.31.3/docs
[v0-31-0-docs]: https://github.com/tektoncd/cli/tree/v0.31.0/docs
[v0-30-2-docs]: https://github.com/tektoncd/cli/tree/v0.30.2/docs
[v0-30-0-docs]: https://github.com/tektoncd/cli/tree/v0.30.0/docs
[v0-29-1-docs]: https://github.com/tektoncd/cli/tree/v0.29.1/docs
[v0-29-0-docs]: https://github.com/tektoncd/cli/tree/v0.29.0/docs
[v0-28-0-docs]: https://github.com/tektoncd/cli/tree/v0.28.0/docs
[v0-27-0-docs]: https://github.com/tektoncd/cli/tree/v0.27.0/docs
[v0-26-0-docs]: https://github.com/tektoncd/cli/tree/v0.26.0/docs
[v0-25-0-docs]: https://github.com/tektoncd/cli/tree/v0.25.0/docs
