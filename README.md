# passgen

`passgen` is a small Go CLI for generating cryptographically secure passwords.

> [!WARNING]
> This project is vibe-coded, but it is safe to use as a small utility when you need a quick, safe password in the terminal.

## Features

- Uses `crypto/rand` for secure random generation.
- Preset character pools for common formats like Base64 and Base58.
- Configurable character sets (uppercase, lowercase, numbers, symbols).
- Optional URL-safe pool filtering.
- Supports explicit character include/exclude rules.

## Requirements

- Go 1.26.3+

## Build and run

Install with Go:

```bash
go install github.com/eslym/passgen@latest
```

The binary is installed to your Go binary directory, usually `$(go env GOPATH)/bin`. Ensure that directory is on your `PATH`, then run:

```bash
passgen --help
```

Run directly:

```bash
go run . --help
```

Build a local binary:

```bash
go build -o passgen .
./passgen
```

## Usage

```bash
passgen [flags]
```

### Common examples

Generate a default 16-character password:

```bash
passgen
```

Generate a 32-character password:

```bash
passgen --length 32
```

Generate 5 passwords at once:

```bash
passgen --count 5
```

Generate JSON output for scripts:

```bash
passgen --json
```

Write generated output to a file with mode `600` (prints status to stderr only):

```bash
passgen --out ./secret.txt
```

Overwrite an existing output file without prompting:

```bash
passgen --out ./secret.txt --force
```

Disable symbols:

```bash
passgen --symbols=false
```

Use only URL-safe characters:

```bash
passgen --urlsafe --symbols=false
```

Seed the pool from a preset:

```bash
passgen --preset b58 --symbols=false --uppercase=false --lowercase=false --numbers=false
```

Force a few characters into the pool:

```bash
passgen --include "@#"
```

Remove ambiguous characters:

```bash
passgen --exclude "O0Il"
```

Inspect the effective pool used for generation:

```bash
passgen --show-pool
```

## Flags

```text
  -a, --alpha            enable both uppercase and lowercase
  -c, --count int        number of passwords to generate (default 1)
  -x, --exclude string   exclude specific characters
  -f, --force            overwrite existing output file without confirmation
  -h, --help             help for passgen
  -i, --include string   add specific characters after filtering
      --json             output as JSON
  -k, --length int       password length (default 16)
  -l, --lowercase        include lowercase letters in base pool (default true)
  -n, --numbers          include numbers in base pool (default true)
      --out string       write output to file (mode 600), suppress stdout
  -p, --preset string    seed pool with preset characters (base64/b64, base64url/b64url, base58/b58, hex, alnum)
  -s, --symbols          include symbols in base pool (default true)
      --show-pool        print effective character pool
  -u, --uppercase        include uppercase letters in base pool (default true)
  -z, --urlsafe          filter base pool to URL-safe characters
```

Available presets:

- `base64`, alias `b64`: `A-Z`, `a-z`, `0-9`, `+`, `/`
- `base64url`, alias `b64url`: `A-Z`, `a-z`, `0-9`, `-`, `_`
- `base58`, alias `b58`: Bitcoin Base58 alphabet, excluding `0`, `O`, `I`, and `l`
- `hex`: `0-9`, `a-f`
- `alnum`: `A-Z`, `a-z`, `0-9`

## Precedence and pool order

Character pool construction is applied in this order:

1. Start from `--preset` if set.
2. Add enabled base classes (`--uppercase`, `--lowercase`, `--numbers`, `--symbols`).
3. Apply URL-safe filtering if `--urlsafe` is enabled.
4. Remove characters from `--exclude`.
5. Add characters from `--include`.

`--include` and `--exclude` are validated first. If they overlap, the command exits with an error.

When `--preset` is used with any other effective pool modifier, `passgen` prints a warning to stderr because presets are only the starting point of the pool. This includes enabled character classes and `--urlsafe`, `--exclude`, or `--include`.

## Validation behavior

- `--length` must be greater than `0`.
- When `--out` is set, output is written only to the file (mode `600`) and a status line is printed to stderr.
- Existing `--out` files require confirmation unless `--force` is set.
- Existing `--out` files with a mode other than `600` print a warning to stderr before the mode is updated.
- If the same character appears in both `--include` and `--exclude`, the command exits with an error.
- If your rules produce an empty character pool, the command exits with an error.

## Development checks

```bash
go test ./...
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```
