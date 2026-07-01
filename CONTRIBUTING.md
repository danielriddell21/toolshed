# Contributing to toolshed

## Requirements

* [Go](https://go.dev) (stable — version from `go.mod`)
* [just](https://github.com/casey/just)
* [golangci-lint](https://golangci-lint.run/welcome/install/) (for `just lint`)
* [gremlins](https://github.com/go-gremlins/gremlins) (for mutation testing)

## Development workflow

```
just build      # build all binaries
just run NAME   # run a demo by name, e.g. just run life
just test       # run unit tests
just lint       # golangci-lint
just fmt        # golangci-lint fmt (gofumpt + goimports)
just tidy       # go mod tidy
```

Run `just --list` to see every recipe. Run `just ci` (lint + test + build) before each commit. CI runs the same gate on every push to `trunk` and every pull request targeting `trunk`.

## Conventions

Each binary's CLI entrypoint follows the structure shared across the tool family
(unum is the reference). See [CONVENTIONS.md](CONVENTIONS.md).

## Project layout

toolshed is a collection of small, self-contained demo programs — one binary per
directory under `cmd/`.

```
cmd/<demo>/      one entry point per demo (life, maze, markov, …)
internal/        shared helper packages
scripts/         helper scripts
docs/            documentation
```

## Adding a new demo

1. Create `cmd/<name>/main.go`
2. Keep it self-contained and runnable with `just run <name>`
3. Add a short entry to the README

## Commit style

```
type(scope): short imperative description
```

Types: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`. No period at the end of the subject line; keep it under 72 characters.

## Releases

Releases are triggered by pushing a semver tag — maintainers only. A GitHub Actions workflow runs GoReleaser to build the binaries and update the Homebrew tap; it requires the tap app credentials configured as repository secrets.
