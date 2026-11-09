# okf-go

Tools for [Open Knowledge Format](https://github.com/GoogleCloudPlatform/knowledge-catalog/tree/main/okf) (OKF) v0.2 bundles: markdown docs with YAML frontmatter, kept in a repository.

- `okf check` checks a bundle for conformance and for the specification's guidance: frontmatter, links, index files, and the trust and lifecycle fields.
- `okf viz` renders a bundle as one HTML page with a graph of its concepts and a detail pane, colored by top-level folder.

## Install

Pin it as a Go tool in the module that holds the bundle:

```sh
go get -tool github.com/katalystng/okf-go/cmd/okf@latest
go tool okf check docs
go tool okf viz docs
```

Or install the binary:

```sh
go install github.com/katalystng/okf-go/cmd/okf@latest
```

## Usage

```sh
okf check [-strict] <bundle>
okf viz [-o file] [-name name] [-layers a,b,c] [-check] <bundle>
```

`okf viz -check` writes nothing and fails when the committed page differs from a fresh render, so CI can keep the page current. See [docs/product/usage.md](docs/product/usage.md) for wiring both into a Makefile.

## Docs

`docs/` is this project's own OKF bundle, and CI checks it with `okf check -strict`. [docs/viz.html](docs/viz.html) is its graph.

## License

Apache License 2.0. The viewer is adapted from the bundle viewer in [knowledge-catalog](https://github.com/GoogleCloudPlatform/knowledge-catalog); see [NOTICE](NOTICE).
