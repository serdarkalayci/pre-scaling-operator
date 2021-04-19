# Releases

This page describes the release process and the currently planned schedule for upcoming releases as well as the respective release shepherd.

## Release schedule

We don't have a release frequency yet. Once the operator becomes more widely used, we will move towards this direction.

| release series | date of first pre-release (year-month-day) | release shepherd                            |
|----------------|--------------------------------------------|---------------------------------------------|
| v1.0           | 2021-04-19                                 | Andreas Krivas (GitHub: @andreas131989) |


## Release shepherd responsibilities

The release shepherd is responsible for the entire release series of a minor release, meaning all pre- and patch releases of a minor release. The process formally starts with the initial pre-release, but some preparations should be done a few days in advance.

* We aim to keep the main branch in a working state at all times. In principle, it should be possible to cut a release from main at any time. In practice, things might not work out as nicely. A few days before the pre-release is scheduled, the shepherd should check the state of main. Following their best judgement, the shepherd should try to expedite bug fixes that are still in progress but should make it into the release. On the other hand, the shepherd may hold back merging last-minute invasive and risky changes that are better suited for the next minor release.
* On the date listed in the table above, the release shepherd cuts the first pre-release (using the suffix `-rc.0`) and creates a new branch called  `v<major>.<minor>` starting at the commit tagged for the pre-release. In general, a pre-release is considered a release candidate (that's what `rc` stands for) and should therefore not contain any known bugs that are planned to be fixed in the final release.
* With the pre-release, the release shepherd is responsible for running and monitoring a benchmark run of the pre-release, after which, if successful, the pre-release is promoted to a stable release.
* If regressions or critical bugs are detected, they need to get fixed before cutting a new pre-release (called `-rc.1`, `-rc.2`, etc.).

### Branch management and versioning strategy

We use [Semantic Versioning](https://semver.org/).

We maintain a separate branch for each minor release, named `v<major>.<minor>`, e.g. `v1.1`, `v2.0`.

Note that branch protection kicks in automatically for any branches whose name starts with `v`. Never use names starting with `v` for branches that are not release branches.

The usual flow is to merge new features and changes into the main branch and to merge bug fixes into the latest release branch. Bug fixes are then merged into main from the latest release branch. The main branch should always contain all commits from the latest release branch. As long as main hasn't deviated from the release branch, new commits can also go to main, followed by merging main back into the release branch.

If a bug fix got accidentally merged into main after non-bug-fix changes in main, the bug-fix commits have to be cherry-picked into the release branch, which then have to be merged back into main. Try to avoid that situation.

Maintaining the release branches for older minor releases happens on a best effort basis.


### Prepare your release

At the start of a new major or minor release cycle create the corresponding release branch based on the main branch. For example if we're releasing `2.17.0` and the previous stable release is `2.16.0` we need to create a `release-2.17` branch. Note that all releases are handled in protected release branches.
Changes for a patch release or release candidate should be merged into the previously mentioned release branch via pull request.

Update `CHANGELOG.md`. Do this in a proper PR pointing to the release branch as this gives others the opportunity to chime in on the release in general and on the addition to the changelog in particular. For a release candidate, append something like `-rc.0` to the version (with the corresponding changes to the tag name, the release name etc.).

For release candidates still update `CHANGELOG.md`, but when you cut the final release later, merge all the changes from the pre-releases into the one final update.
