---
type: Product Documentation
title: "Usage"
description: "How to install okf, run its commands, and wire them into a repository."
tags:
  - usage
sources:
  - resource: /spec/check.md
  - resource: /spec/viz.md
attribution:
  derivation: original
  model: "claude-opus-5-5"
  authors:
    - "claude"
---

# Usage

The commands implement [check](../spec/check.md) and [viz](../spec/viz.md).

## Install as a Go tool

A Go module pins okf with the `tool` directive, so the version sits in `go.mod` and Dependabot proposes bumps:

```sh
go get -tool github.com/katalystng/okf-go/cmd/okf@latest
go tool okf check docs
```

## check

```sh
okf check [-strict] <bundle>
```

## viz

```sh
okf viz [-o file] [-name name] [-layers a,b,c] [-check] <bundle>
```

The page defaults to `<bundle>/viz.html`. The name defaults to the bundle folder's name, or to its parent's name when the folder is `docs`.

## Wire into a repository

A Makefile target regenerates the page, and the verify target checks both the bundle and the page:

```make
docs-viz:
	go tool okf viz -layers intent,spec,decisions,product docs

verify-docs:
	go tool okf check docs
	go tool okf viz -check -layers intent,spec,decisions,product docs
```

## The viewer

The page is a port of the bundle viewer in GoogleCloudPlatform/knowledge-catalog. It loads Cytoscape 3.28.1 and marked 12.0.0 from jsdelivr, so a reader needs network access to view it.
