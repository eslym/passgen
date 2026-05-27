# AGENTS

## Scope
- Single-package Go CLI (`module github.com/eslym/passgen`); runtime code is in `main.go`.
- Tests are in `main_test.go`; there are no subpackages and no CI workflow files.

## Entry points and structure
- CLI entrypoint is `main()` in `main.go`; commands and flags are wired with Cobra.
- Core logic also lives in `main.go`: pool construction (`buildPool`) and password generation (`generatePassword`).

## Dev commands (verified)
- Run CLI without building: `go run . --help`
- Build binary: `go build -o passgen .`
- Run all tests: `go test ./...`
- Run one test quickly while editing: `go test ./... -run TestBuildPool`
- Run vulnerability scan before finishing dependency or security-relevant changes: `go run golang.org/x/vuln/cmd/govulncheck@latest ./...`

## Workflow rules
- Prefix every commit message with a conventional tag that matches the change type (for example: `feat:`, `fix:`, `chore:`, `docs:`, `refactor:`, `test:`, `ci:`, `build:`).
- Use lowercase tags and commit title format: `<tag>: <imperative summary>`.

## Repo-specific gotchas
- Built binaries are expected at repo root and gitignored: `passgen`, `passgen.exe` (`.gitignore`).
- Flag behavior is controlled in `cmd.PreRun`; when changing flags, re-check `--alpha` interactions with `--uppercase` and `--lowercase`.
- `buildPool` rejects overlapping `--include` and `--exclude`; preserve this validation when editing character-pool logic.
- Pool construction order is intentional and documented in `README.md`: base classes -> URL-safe filter -> exclude -> include.
- `--out` writes to file and suppresses stdout; keep tests aligned with this behavior (`TestOutFlagSuppressesStdout`).
