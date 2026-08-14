# ADR 0003: Prefer low-level Git-compatible native object storage

Status: accepted; byte contract completed by ADR 0005

## Decision

Standalone `.sealgraph/objects` should use low-level Git-compatible loose-object concepts where practical.

Compatibility is an implementation/forensics property, not a user-facing Git semantic contract.

Sealgraph does not promise that `git log`, `git merge`, `git checkout`, or other porcelain commands make sense against standalone metadata.

## Constraints

- keep canonical v1 storage loose and merge-friendly,
- avoid canonical packfiles and packed refs,
- use sealgraph-specific ref namespace such as `refs/seals`,
- do not model seals as Git commits merely to gain porcelain compatibility.

## Consequences

Low-level tooling and SDK code may be reused while sealgraph retains its own model.

ADR 0005 fixes the native v1 envelope as a SHA-256 Git blob object while
keeping seal payload semantics independent from Git commits and refs.
