# release-notes
[![Build Status](https://travis-ci.com/gsquared94/release-notes.svg?branch=master)](https://travis-ci.com/gsquared94/release-notes)
[![LICENSE](https://img.shields.io/github/license/gsquared94/release-notes.svg)](https://github.com/gsquared94/release-notes/blob/master/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/gsquared94/release-notes)](https://goreportcard.com/report/gsquared94/release-notes)
<!--
[![Code Coverage](https://codecov.io/gh/gsquared94/release-notes/branch/master/graph/badge.svg)](https://codecov.io/gh/gsquared94/release-notes)
[![Releases](https://img.shields.io/github/release-pre/gsquared94/release-notes.svg)](https://github.com/gsquared94/release-notes/releases)
-->
Forked from https://github.com/corneliusweig/release-notes

Generates a markdown changelog of merged pull requests since last release.
It consists of sections `New Features`, `Fixes`, `Updates & Refactors`, and `Docs updates` based on the PRs being labelled with `release/new-feature`, `release/fixes`, `releases/refactor` and `release/doc-updates` respectively

The script uses the GitHub API to retrieve a list of all merged pull
requests since the last release. The found pull requests are then
printed as markdown changelog with their commit summary and a link
to the pull request on GitHub.  

The idea and original implementation of this script is due to Bálint Pató
([@balopat](https://github.com/balopat)) while working on
[minikube](https://github.com/kubernetes/minikube) and
[Skaffold](https://github.com/GoogleContainerTools/skaffold).

## Examples

The binary expects two parameters:

1. The GitHub organization which your repository is part of.
2. The repository name.

For example:
```sh
./release-notes GoogleContainerTools skaffold
```

which will output
```text
New Features:
* add github pull request template [#2894](https://github.com/googlecontainertools/skaffold/pull/2894)


Fixes:
* Add Jib-Gradle support for Kotlin buildscripts [#2914](https://github.com/googlecontainertools/skaffold/pull/2914)


Updates & Refactors:
* Add missing T.Helper() in testutil.Check* as required [#2913](https://github.com/googlecontainertools/skaffold/pull/2913)


Docs updates:
* Move buildpacks tutorial to buildpacks example README [#2908](https://github.com/googlecontainertools/skaffold/pull/2908)
...
```

## Options

##### `--token`

Specify a personal Github Token if you are hitting a rate limit anonymously (see https://github.com/settings/tokens). Otherwise, set the environment variable `GITHUB_TOKEN` (already set when running in a Github Action) 

##### `--since`

The tag of the last release up to which PRs should be collected (one of `any`, `patch`, `minor`, `major`, or a valid semver). Defaults to 'patch'. 

For example:

|  |`2.3.4-alpha.1+1234`|`2.3.4-alpha.1`|`2.3.4`|`2.3.0`|`2.0.0`|
|---|---|---|---|---|---|
|`any`|true|true|true|true|true|
|`patch`|false|false|true|true|true|
|`minor`|false|false|false|true|true|
|`major`|false|false|false|false|true|


## Installation

Currently, you need a working Go compiler to build this script:

```sh
go get github.com/gsquared94/release-notes
```
