# Contributing to UIAI Engine

UIAI Engine is currently source-available and commercially licensed.

External contributions are not accepted unless WPUIAI / Startempire Wire has
explicitly approved the contribution path. Approved contributors may be
required to sign a Contributor License Agreement or assignment before code,
docs, designs, tests, issues, or other materials are incorporated.

## How contributions are reviewed

Approved contributions go through:

1. A public issue describing the change, the use case, and the proposed
   implementation.
2. Internal review by a WPUIAI / Startempire Wire maintainer with context on
   the existing code, the commercial license path, and the platform contract.
3. A private CI run against the current `main` to confirm the change does not
   break the loopback evaluation path, the remote authenticated path, the
   license enforcement, the credit meter, or the tool registry.
4. A signed CLA or assignment on file before the change is merged.
5. A signed commit on `main` with the agreed tag.

## Style

UIAI Engine is Go. Code style follows `gofmt` and `go vet`. New routes and
feature gates follow the existing patterns in `internal/routes/` and the
license-aware feature helper once it lands in `internal/license/`. Prefer
small, surgical commits over batched rewrites.

## Reporting issues

Public issues are accepted for evaluation, documentation, and security reports.
For commercial use cases, agency use, or licensing questions, email
`licensing@startempirewire.com` rather than opening a public issue.

## Code of conduct

Contributors and reviewers are expected to keep the discussion focused on the
work, the platform, and the buyer outcomes. Off-topic or abusive comments are
removed. Repeated violations result in the contributor being blocked from the
project.

## No expectation of merge

WPUIAI / Startempire Wire accepts contributions at its sole discretion. There
is no service-level agreement on review or merge timing, and no guarantee that
any contribution will be accepted even after CLA signature.
