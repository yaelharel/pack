We're glad you are interested in contributing to this project. We hope that this
document helps you get started.

If something is missing, incorrect, or made irrelevant please feel free to make
a PR to keep it up-to-date.

## Prerequisites

- [Go](https://golang.org/dl/)
    - including [Git](https://git-scm.com/)
- [Docker](https://www.docker.com/)

## Development
| ENV_VAR      | Description                                                            | Default |
|--------------|------------------------------------------------------------------------|---------|
| GOCMD        | Change the `go` executable. For example, [richgo][rgo] for testing.    | go      |
### Building

To build pack:
```
make build
```

This will output the binary to the directory `out/`.

Options:

| ENV_VAR      | Description                                                            | Default |
|--------------|------------------------------------------------------------------------|---------|
| PACK_BIN     | Change the name or location of the binary relative to `out/`.          | pack    |
| PACK_VERSION | Tell `pack` what version to consider itself                            | `dev-<timestamp>` |

[rgo]: https://github.com/kyoh86/richgo

> NOTE: This project uses [go modules](https://github.com/golang/go/wiki/Modules) for dependency management.

To build and then archive `pack` into a TGZ file:
```
make build package
```

> NOTE: The same options as `make build` apply.

This will output a file at the path `out/pack-<pack_version>.tgz`.

### Testing

To run unit and integration tests:

```bash
make unit
```

To run acceptance tests:
```bash
make acceptance
```

Options:

| ENV_VAR   | Description                                                                   | Default |
|-----------|-------------------------------------------------------------------------------|----------|
| PACK_PATH | Path to the `pack` binary to test | `pack` is compiled from source if omitted |
| COMPILE_PACK_WITH_VERSION | If `PACK_PATH` is not set, the version `pack` is compiled with | `0.0.0` |

Alternately, to run all tests:
```bash
make test
```

### Formatting

To format the code:

```bash
make format
```

### Verification

To verify formatting and vet:
```bash
make verify
```

## Pull Requests

1. **[Fork]((https://help.github.com/en/articles/fork-a-repo)) the repo**
2. **Code, Test, Commit...**

    _Don't forget utilize the convenient make functions above._

3. **Preparing a Branch**

    We prefer to have PRs that are encompassed in a single commit. This might
    require that you execute some of these commands:

    If you are no up-to-date with master:
    ```bash
    # rebase from master (applies your changes on top of master)
    git pull -r origin master
    ```

    If you made more than one commit:
    ```bash
    # squash multiple commits, if applicable
    # set the top most commit to `pick` and all subsequent to `squash`
    git rebase -i origin/master
    ```

    Another requirement is that you sign your work. See [DCO](https://probot.github.io/apps/dco/) for more details.
    ```bash
    git commit --amend --signoff
    ```

4. **Submit a Pull Request**

    Submitting the pull request is done in [GitHub](https://github.com/buildpacks/pack/compare/) by selecting
    your branch as the `compare` branch.