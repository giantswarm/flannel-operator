# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed

- Fix typo in `configmap` labels field name.

## [1.1.0] - 2020-04-29

### Changed

- Use `flannel-operator` as common resource name due to legacy hard-coded references.

## [1.0.0] - 2020-04-29

### Changed

- Push `flannel-operator` chart into `control-plane` catalog instead of quay.io.
- Push `flannel-operator` app CRs into `<provider>-app-collection` repository.

[Unreleased]: https://github.com/giantswarm/flannel-operator/compare/v1.1.0...HEAD

[1.1.0]: https://github.com/giantswarm/flannel-operator/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/giantswarm/flannel-operator/tag/v1.0.0
