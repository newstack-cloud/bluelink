# Contributing to the blueprint framework

## Getting set up

### Prerequisites

- [Go](https://golang.org/dl/) >=1.23

Dependencies are managed with Go modules (go.mod) and will be installed automatically when you first
run tests.

If you want to install dependencies manually you can run:

```bash
go mod download
```

## Running tests

```bash
bash ./scripts/run-tests.sh

# to re-generate snapshots (For spec/schema tests)
bash scripts/run-tests.sh --update-snapshots
```

## Benchmarking link deployment throughput

`container/link_throughput_bench_test.go` measures how many links a deployment runs at
once. It exists because link throughput is not visible from a passing test suite, a
deployment that runs every link one after another is correct, and only a benchmark shows
that it is also serial.

Benchmarks do not run under a plain `go test`, so run them explicitly from `libs/blueprint`:

```bash
# all shapes, one deployment each
go test -run XXX -bench BenchmarkLinkThroughput -benchtime=1x ./container/

# one shape
go test -run XXX -bench 'BenchmarkLinkThroughput/functions=8/linksPerFunction=3' \
  -benchtime=1x ./container/

# several runs per shape, to see how stable the figures are
go test -run XXX -bench BenchmarkLinkThroughput -benchtime=5x -timeout 900s ./container/
```

`-run XXX` matches no ordinary test, so only the benchmarks run. `-benchtime=1x` gives one
deployment per shape; the default is time-based and would repeat each shape until it had
accumulated a second of samples, which takes minutes here for no extra information. Raise
`-timeout` when using a higher `-benchtime`.

### Reading the output

```
functions=8/linksPerFunction=3/links=24-18   1   7553350250 ns/op
    7.551 deployment-secs      2.547 link-phase-secs        24.00 links
    9.425 links-per-sec        0.9856 mean-links-in-flight   1.000 peak-links-in-flight
    0.2520 worst-resource-busy-secs
```

Ignore `ns/op`. It is whole-deployment wall clock and is dominated by resource deployment
rather than by links.

| Metric | What it means |
| --- | --- |
| `mean-links-in-flight` | Time-weighted concurrency across the link phase. **The headline number.** 1.0 means the deployment is running links one at a time. |
| `links-per-sec` | Link throughput. Flat across shapes means throughput does not improve when more independent work is available. |
| `link-phase-secs` | First link entering a phase to the last one leaving, rather than the whole deployment. Divided by `links` it should stay flat if the phase is serial. |
| `peak-links-in-flight` | Momentary maximum, median across runs. Noisier than the mean and easy to over-read. |
| `worst-resource-busy-secs` | The longest any one resource spent with a link writing it, which is the per-resource serialisation a resource lock enforces. |
| `deployment-secs` | Whole deployment, for context. |

### Changing what it measures

The blueprint is generated rather than a fixture: `linkThroughputShape` gives the number of
functions, the number of tables each links to, and a per-phase latency, and
`generateLinkThroughputBlueprint` turns that into a spec loaded with `LoadString`. Tables
are private to each function so that links on unrelated resources are genuinely independent,
which is the thing being measured. A table is the priority resource of a function-to-table
link, so the function deploys last and its links are all released together, matching how
link batches form in a real deployment.

Latency defaults to 20ms per phase, set in `BenchmarkLinkThroughput`. Wall clock scales with
it directly; concurrency should not, and if it does that is a finding rather than noise.

## Generating protobuf code

The blueprint framework uses protobuf to store and transmit an expanded version of a blueprint. Expanded blueprints include AST-like expansions of substitutions that can be cached with an implementation of the `cache.BlueprintCache` interface.

1. Follow the instructions [here](https://grpc.io/docs/protoc-installation/#install-using-a-package-manager) to install the `protoc` compiler.

2. Install the Go protoc plugin:

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
```

3. Run the following command from the `libs/blueprint` directory to generate the protobuf code:

```bash
protoc --go_out=./schemapb --go_opt=paths=source_relative ./schema.proto
```

## Releasing

Releases are automated using [release-please](https://github.com/googleapis/release-please).

### How it works

1. **Conventional commits drive releases** - Commits with scopes matching this library (e.g., `feat(blueprint): ...` or `fix(blueprint): ...`) are tracked by release-please.

2. **Release PRs are created automatically** - When releasable commits land on `main`, release-please opens/updates a PR with:
   - Version bump based on commit types (feat = minor, fix = patch)
   - CHANGELOG.md updates

3. **Merging creates the release** - When the release PR is merged:
   - A GitHub release is created
   - Two git tags are created:
     - `blueprint/v{version}` - Used internally by release-please for tracking. Do not use this tag.
     - `libs/blueprint/v{version}` - The canonical Go module tag. Use this for dependencies and references.

### Go module indexing

When a library release tag is pushed, the `index-go-library.yml` workflow automatically indexes the new version with the Go module proxy (pkg.go.dev).

### Tag format

Tags follow Go module conventions: `libs/blueprint/vX.Y.Z`

Example: `libs/blueprint/v0.37.0`

## Commit scope

**blueprint**

Example commit:

```bash
git commit -m 'fix(blueprint): correct cyclic dependency bug'
```
