---
type: Specification
title: "viz"
description: "The requirements for okf viz: the page a bundle renders as, and how a committed page stays current."
tags:
  - viz
sources:
  - resource: /intent/okf-go.md
attribution:
  derivation: original
  model: "claude-opus-5-5"
  authors:
    - "claude"
---

# viz

Rests on [the intent](../intent/okf-go.md).

## Governed paths

```yaml
paths:
  - viz/**
  - cmd/okf/**
```

## Requirements

- Page: a bundle renders as one HTML file that holds its data and its code, with each concept a node and a detail pane for the selected one.
- Edges: a relative link and a bundle-relative link to a concept each draw an edge. A `sources` entry that names a concept draws a dashed edge.
- Layers: each concept takes the color of its top-level folder. `-layers` orders the folders, and the legend dims every other layer on a click.
- Attribution: a concept whose frontmatter holds `attribution` shows it in the detail pane.
- Stability: the page depends on the bundle alone. Staleness, which depends on the time of viewing, is computed in the page.
- Currency: `okf viz -check` writes nothing and exits 1 when the page on disk differs from a fresh render, so a repository can commit its page and check it.
