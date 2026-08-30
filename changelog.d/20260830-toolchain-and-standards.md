### Fixed

#### `toolchain go1.24.2` sat below `go 1.26.0`, and `.standards` was three versions stale

`go.mod` declared `go 1.26.0` while pinning `toolchain go1.24.2` — a toolchain
*older* than the language version the module requires. That combination is
self-contradictory, and it is invisible on a developer machine: `GOTOOLCHAIN`
defaults to `auto`, so Go quietly downloads a newer toolchain and the build
succeeds.

It bites where something resolves the toolchain **from that line** and is then
forbidden to upgrade. CodeQL's default setup does exactly that: its `Setup Go`
step reads go.mod and runs with `GOTOOLCHAIN=local`. In `overnight-burndown`
the identical inversion installed the pinned-but-too-old Go and failed
extraction outright with `go: go.mod requires go >= ... (GOTOOLCHAIN=local)`.

Here it has been latent rather than breaking, and only by luck: this repo has
no root build script, so CodeQL's autobuild fell through to `go get ./...`,
which self-healed (`go: downloading go1.26.0`, `go: removed toolchain
go1.24.2`). A green `Analyze (go)` therefore meant only that nothing invoked
the pinned toolchain — not that go.mod was right. Removed rather than bumped:
the `go` directive already names the version we build with, so the line is
redundant.

Also bumps the `.standards` submodule from `664ae68` (2026-06-12) to `7bdfd13`.
It is a pinned commit, not a live link, and had not moved in eleven weeks while
`instructions/go.md` went from v1.0.0 to v1.3.0 — the Go version policy and the
1.26 minimum, the `io/ioutil` ban, the `wg.Go` rule, the testing-isolation
table, and `omitempty` vs `omitzero`. CLAUDE.md points contributors at that
directory as authoritative, so the stale pin was serving rules three versions
out of date. A `gitsubmodule` entry in `.github/dependabot.yml` now keeps it
current on the existing weekly schedule.

Verified: `go build ./...` and `go vet ./...` both exit 0, and nothing in CI,
no workflow or script, reads `.standards` — it is documentation only, so the
jump carries no build risk.
