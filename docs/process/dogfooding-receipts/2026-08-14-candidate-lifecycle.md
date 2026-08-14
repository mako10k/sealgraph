# Candidate lifecycle and safe-output dogfood receipt

Status: focused temporary-repository round passed on 2026-08-14.

## Implementation under test

- base commit: `1faf75d2237981355168e2c2a1e0449bac54c9c7`
- source state: reviewable uncommitted ADR 0008 implementation worktree
- built `sealgraph` SHA-256:
  `2e09722f4143c1dde5f1a4881886c5a032bc170275a3f13bfb4925b8a2e2cf5a`
- product mode: standalone only
- no Git sidecar command or project-root `.sealgraph` mutation

This receipt is pre-commit implementation evidence, not a frozen-release or
remote-publication claim.

## Scenario

The binary was built from the current worktree and run in a new temporary
directory. The normalized lifecycle was:

```text
init
add ROOT --root --content-file -
candidate show ROOT
candidate diff ROOT
candidate show ROOT --raw-content
seal ROOT -m ...
show ROOT
show ROOT --raw-content
add DESIGN --draft --content ... --depend-on ROOT
candidate show DESIGN
unlink DESIGN --upstream ROOT@<12-character-prefix>
candidate diff DESIGN
link DESIGN --depend-on ROOT -m ...
seal DESIGN -m ...
add THROWAWAY --root --content ...
candidate discard THROWAWAY
candidate show THROWAWAY
```

ROOT exact content was the nine bytes `A`, LF, NUL, ESC, `0xff`, and `tail`.
Both candidate and sealed raw modes matched those bytes with `cmp`. Default
human output emitted the bounded preview:

```text
CONTENT_PREVIEW "A\n\x00\x1b\xfftail" truncated=false
```

No raw NUL, ESC, or invalid UTF-8 byte was mixed into human metadata output.

## Readback

- initial ROOT candidate reported `BASE_STATE INITIAL`;
- initial candidate diff used `FROM -` and `TO CANDIDATE`;
- root seal ID was
  `596df2b477b77856cff62d1f6cd20d73b50a38a2a73430771a693667670978c2`;
- guarded unlink resolved the 12-character prefix and removed exactly the ROOT
  edge from DESIGN;
- candidate diff then showed `LINKS SET count=0` without changing content,
  root, or draft;
- explicit link restored the one edge before DESIGN was sealed;
- candidate discard removed THROWAWAY only;
- subsequent candidate show failed with `no working candidate` rather than
  reporting a false empty candidate;
- no automatic relink, root conversion, seal, or recursive repair occurred.

The temporary directory
`/tmp/sealgraph-candidate-dogfood.5tLvTT` was moved to the desktop trash with
`gio trash` after readback; the cleanup was recoverable rather than a direct
recursive deletion.

## Automated evidence associated with this round

```text
go vet ./...        PASS
go test ./...       PASS
go test -race ./... PASS
npm run clone-check PASS after common diff-presentation extraction; 0 clones
llmthink audit      fatal=0 error=0 warning=0
perttool check      PASS
```

Final required validation was rerun after this receipt and planning updates and
passed with the results above.
