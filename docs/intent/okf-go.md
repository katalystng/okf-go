---
type: Intent
title: "okf-go"
description: "The purpose of okf-go: shared tooling that checks and renders the OKF bundles repositories keep as their docs."
tags:
  - okf
  - purpose
attribution:
  derivation: paraphrased
  model: "claude-opus-5-5"
  authors:
    - "claude"
  contributors:
    - "human:travissutton"
---

# okf-go

## Purpose

Repositories keep their docs in the repository as an Open Knowledge Format (OKF) v0.2 bundle. okf-go is the one shared tool those repositories pin, so each one checks and renders its bundle the same way.

## What it does

- check: a bundle is checked for conformance with OKF v0.2 and for the guidance the specification gives, so a broken link or an index that drifts from its files is caught before it merges.
- viz: a bundle renders as one HTML page with a graph of its concepts, a quick way for a reader to view and navigate the docs.

## Defaults and enrichment

- The tool implements plain OKF v0.2 as its default.
- Layers color the graph by default, from each concept's top-level folder.
- Attribution shows when a concept's frontmatter holds it.

## Distribution

- The repository is public, under the Apache License 2.0.
- The project carries little organization branding.
- A consuming repository pins a tagged release in its `go.mod`.

## Its own docs

okf-go keeps its docs as an OKF bundle in `docs/`, and its CI checks that bundle with the tool itself.
