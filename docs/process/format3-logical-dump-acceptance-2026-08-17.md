# Format-3 logical dump acceptance — 2026-08-17

Status: `FORMAT3_LOGICAL_DUMP` complete. This receipt does not authorize
format-4 runtime or load implementation.

## Accepted contract

The operator accepted the complete logical dump and future load-boundary
proposal on 2026-08-17. ADR 0012 records the decision.

Implemented format-3 surface:

```sh
sealgraph dump --format logical-v1
```

The slice includes the canonical logical-v1 encoder, strict complete loose
object and tag-tree inventory, candidate rejection, REF/tag-rooted parent and
Cause closure, dependency-first ordering, excluded-object reporting, final
observation revalidation, buffered CLI output, and focused tests.

## Automated validation

The final working tree passed:

```text
gofmt -w .                       OK
go vet ./...                     OK
go test ./...                    OK
go test -race ./...              OK
npm ci                           OK; 0 vulnerabilities
npm run clone-check              OK; 0 clones
git diff --check                 OK
llmthink logical-dump audit      fatal=0 error=0 warning=0
perttool document check          OK
perttool dag analyze             OK; PTDAG-302 path-display truncation only
```

Focused tests cover exact empty and binary logical-v1 fixtures, attachment
bytes, deterministic repeat output, dependency-first diamond ordering,
canonical ownership, valid/corrupt candidates, corrupt unreachable objects,
orphan tag scopes, final-observation change, zero stdout on rejection, and
source-tree byte/mode preservation.

## Tracked dogfood read-only receipt

The checked-in format-3 `.sealgraph` repository was dumped twice with the new
command. Both outputs were byte-identical and parsed as:

```text
schema            sealgraph/logical-dump/v1
objects           4
seals             4
refs              4
tags              4
excluded_objects  0
dump sha256        f63f44f7017884a11978be48eb323a33dc487dc41983c25d796cd4f32d2c1125
```

Complete `.sealgraph` relative paths, entry types/modes, and every regular-file
SHA-256 digest matched before and after both runs. The dump artifact was kept
outside the repository during validation and moved to the desktop trash after
the receipt values were captured.

This is format-3 export evidence only. It does not convert, rewrite, reseal, or
advance tracked dogfood.

## Remaining boundary

- format-4 Seal/candidate runtime and load are not started;
- `TAG_CONTRACT` now precedes tag-bearing `FORMAT4_DOGFOOD_LOAD`;
- tracked `.sealgraph` remains exact format 3;
- Git sidecar, release, publication, and external Issue state are unchanged.
