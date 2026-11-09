---
type: Specification
title: "check"
description: "The requirements for okf check: what a bundle is checked for and how a finding fails a run."
tags:
  - check
sources:
  - resource: /intent/okf-go.md
attribution:
  derivation: original
  model: "claude-opus-5-5"
  authors:
    - "claude"
---

# check

Rests on [the intent](../intent/okf-go.md).

## Governed paths

```yaml
paths:
  - bundle/**
  - check/**
  - cmd/okf/**
```

## Requirements

- Conformance: a failure of OKF v0.2 section 11 is an error. A concept document lacks frontmatter, its frontmatter fails to parse, or it lacks `type`.
- Guidance: a departure from guidance the specification gives as SHOULD is a warning. Examples are a relative or bundle-relative link that resolves to no file, an index that leaves out a file or folder beside it, and an index entry that differs from the concept's `description`.
- Optional fields: a trust, lifecycle, or provenance field is checked when it is present, against OKF v0.2 section 5.
- Extensions: a field the bundle adds of its own is checked when it is present. `attribution` is one.
- Exit: a run exits 1 on an error. With `-strict` it also exits 1 on a warning.
- Output: each finding prints as `path:line: severity: message`, sorted by path and line.
