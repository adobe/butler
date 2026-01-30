# Release v1.5.0

## What's Changed

This release adds a new **watch-only mode** for ConfigMap monitoring, comprehensive test coverage improvements, and infrastructure updates.

### New Features

#### Watch-Only Mode for ConfigMap Monitoring

Added a new `watch-only` configuration option that enables butler to monitor files for changes using hash comparison and trigger reloads **without writing any files to disk**. This is ideal for Kubernetes ConfigMap monitoring scenarios where files are mounted read-only.

```toml
[jenkins]
  repos = ["jcasc-local"]
  watch-only = "true"
  skip-butler-header = "true"
  primary-config-name = "jcasc-config.yaml"

  [jenkins.jcasc-local]
    method = "file"
    repo-path = "/usr/share/jenkins/init.jcasc.d"
    primary-config = ["config.yaml"]

  [jenkins.reloader]
    method = "https"
    [jenkins.reloader.https]
      host = "localhost"
      port = "8080"
      uri = "/reload-configuration-as-code/?casc-reload-token=mytoken"
      method = "post"
```

When enabled:
- Butler **hashes** source files from `repo-path` instead of copying them
- Compares hashes to previous run (stored in memory)
- Triggers reloader if hashes differ
- **Never writes** to `dest-path` (which becomes optional)
- First run always triggers reload (no previous hashes)
- Container restart triggers reload (in-memory hashes lost)

### Test Infrastructure Improvements

- **Regenerated TLS certificates** with proper Subject Alternative Names (SANs) for `localhost` and `127.0.0.1`, fixing acceptance test failures with Go 1.15+
- **Added comprehensive unit tests** for previously untested code:
  - `internal/config/status_test.go` - Status file operations
  - `internal/config/objects_test.go` - ValidateOpts and RepoFileEvent
  - `internal/config/chan_test.go` - ConfigChanEvent operations
  - `internal/config/manager_test.go` - Manager methods
  - `internal/reloaders/reloaders_test.go` - Reloader error handling
  - `internal/reloaders/http_test.go` - HTTP reloader functionality
  - `internal/alog/alog_test.go` - Apache logging handler
  - `internal/methods/methods_test.go` - Method factory
- **Added tests for watch-only mode** in `helpers_test.go` and `config_test.go`

### CI/CD Improvements

- **Parallel test execution** in GitHub Actions workflows - unit tests and acceptance tests now run concurrently
- **Updated Dockerfile** to include tests for `internal/environment` and `internal/alog` packages
- **Added certificate generation script** (`files/certs/generate_certs.sh`) for reproducible test certificate creation

### Build System Updates

- Updated `make test` to run only unit tests by default (faster CI)
- Added `make test-all` to run both unit and acceptance tests
- Removed legacy Dockerfile references to non-existent files

## Breaking Changes

None - this release is fully backward compatible.

## Full Changelog

https://github.com/adobe/butler/compare/v1.4.0...v1.5.0

---

# Release v1.4.0

## What's Changed

This release includes significant improvements to the build system, new configuration options, and various enhancements.

### New Features

#### `skip-butler-header` Configuration Option
Added a new per-manager configuration option that allows skipping the `#butlerstart` and `#butlerend` header/footer validation. This is useful for managing files like Kubernetes ConfigMaps or JCasC configurations that cannot easily include butler markers.

```toml
[mymanager]
  repos = ["myrepo"]
  skip-butler-header = "true"
  dest-path = "/path/to/configs"
```

When enabled:
- Butler will **not** require `#butlerstart` and `#butlerend` markers
- YAML syntax validation still occurs for `.yaml`/`.yml` files
- JSON syntax validation still occurs for `.json` files

#### GitHub Actions Release Workflows
Added automated release workflows using GitHub Actions with label-based versioning:
- `release:major` - Breaking changes (v1.0.0 → v2.0.0)
- `release:minor` - New features (v1.0.0 → v1.1.0)
- `release:patch` - Bug fixes (v1.0.0 → v1.0.1)
- `release:skip` - Skip automatic release

### Build System Modernization
- Migrated from `dep` to Go modules (`go.mod`)
- Consolidated multiple Dockerfiles into a single multi-stage `docker/Dockerfile`
- Added Docker Buildx Bake configuration (`docker-bake.hcl`)
- Updated to Go 1.21
- Removed vendor directory (dependencies fetched at build time)

### Other Improvements
- Added blob account CLI flags for Azure Blob storage
- S3 improvements
- Added `InsecureSkipVerify` option to ignore etcd SSL warnings
- Updated metrics handling
- Added default HTTP options
- Code linting and cleanup
- Added Travis CI configuration

## Commits

- `2cd59dc` Add skip header option (#45) (Stegen Smith)
- `7a10661` Updating base container / improving container build / adding workflows (#44) (Stegen Smith)
- `135d32e` moved govender to dep (vs glide) (#38) (Stegen Smith)
- `e04aa73` added blob account cli flags (#37) (Stegen Smith)
- `26ef1a6` S3 improvements (#36) (Stegen Smith)
- `fe50d2a` Tidying up how we're handling the different methods (#35) (Stegen Smith)
- `fa4c10c` updating metrics (#34) (Stegen Smith)
- `95711e8` Add InsecureSkipVerify to ignore etcd ssl warnings (#33) (Friedrich Gonzalez)
- `43812aa` Doing some linting and other cleanup (#30) (Stegen Smith)
- `85d928f` Adding travis ci stuff (#29) (Stegen Smith)
- `7d241f9` Adding default http options (#26) (Stegen Smith)

## Breaking Changes

- The old `make build` command now uses Docker Buildx Bake instead of direct Docker build
- Vendor directory has been removed; builds now require network access to fetch dependencies

## Full Changelog

https://github.com/adobe/butler/compare/v1.3.0...v1.4.0
