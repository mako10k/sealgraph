# Operator contract acceptance — 2026-08-17

Status: `OPERATOR_CONTRACT` complete. SG-BL-005, SG-BL-006, SG-BL-007, and
SG-BL-011 are resolved by ADR 0015 and the checked-in implementation.

## Accepted behavior

- Help defines `CLEAN`, logical REF, structural impact, root, history, and the
  standalone/Git boundary. Human status, impact, and graph output identify
  their semantic domains explicitly.
- `show`, `status`, `stale`, `graph`, `impact`, `log`, `linklog`, and `diff`
  provide command-specific version-1 JSON documents. Full IDs and Cause paths
  remain structured. ADR 0010's REF-only bytes are unchanged.
- Explicit init reports initialized, runtime-bootstrap, and already-complete
  outcomes without printing the checkout path. Bootstrap changes runtime
  directories only.
- Distinct dependency messages retain the explicit repeated-command contract.
  One invocation resolves all selectors before one normalized candidate save;
  it does not introduce semantic Link kinds.

No persisted Seal, object, REF, tag, or candidate format changed. Standalone
code gained no Git discovery. This slice does not authorize recurring dogfood,
Git-sidecar work, push, release, or external Issue mutation.

## Validation

```text
gofmt -w .                    OK
go vet ./...                  OK
go test ./...                 OK
go test -race ./...           OK
npm ci                        OK; 0 vulnerabilities
npm run clone-check           OK; 47 files, 0 clones
make complexity-check         OK; no function above 20
make deadcode-check           OK; 0 reported unreachable functions
```

Focused CLI tests cover all eight schema identifiers, structured impact paths,
the JSON/REF-only conflict, standalone help semantics, and all three init
outcomes. Existing `stale --frontier --refs-only --scan` still emits exactly
`middle` plus LF in its revision fixture.

With `CONTENT_INGEST` already complete, reaching `OPERATOR_CONTRACT` completes
the `USABLE` milestone. The next PERT frontier is `DOGFOOD_RECURRING`; it remains
unstarted and separately gated.
