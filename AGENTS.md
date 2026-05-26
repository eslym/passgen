# AGENTS

## Scope
- This is a single-package Go CLI app (`module passgen`) with one source file: `main.go`.
- There are no subpackages, no test files, and no CI/workflow configs in this repo.

## Entry points and structure
- CLI entrypoint is `main()` in `main.go`; commands and flags are wired with Cobra.
- Core logic also lives in `main.go` (`buildPool`, character filtering helpers, `generatePassword`).

## Dev commands (verified)
- Run CLI without building: `go run . --help`
- Build binary: `go build -o passgen .`
- Run module checks: `go test ./...`
- Run vulnerability scan before finishing dependency or security-relevant changes: `go run golang.org/x/vuln/cmd/govulncheck@latest ./...`

## Workflow rules
- Prefix every commit message with a conventional tag that matches the change type (for example: `feat:`, `fix:`, `chore:`, `docs:`, `refactor:`, `test:`, `ci:`, `build:`).
- Use lowercase tags and commit title format: `<tag>: <imperative summary>`.

## Repo-specific gotchas
- Built binaries are expected at repo root and gitignored: `passgen`, `passgen.exe` (`.gitignore`).
- Flag behavior is controlled in `cmd.PreRun`; when changing flags, verify interactions between enable flags (`--uppercase`, `--lowercase`, `--numbers`, `--symbols`, `--urlsafe`) and disable/override flags (`--no-uppercase`, `--no-lowercase`, `--alpha`, `--no-alpha`, `--no-numbers`, `--no-symbols`, `--no-urlsafe`).
- `buildPool` rejects overlapping `--include` and `--exclude`; preserve this validation when editing character-pool logic.
