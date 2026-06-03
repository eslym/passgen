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

Use a preset pool:

```bash
passgen --preset b58
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
  -c, --count int        number of passwords to generate, 1-1000 (default 1)
  -x, --exclude string   exclude specific characters
  -f, --force            overwrite existing output file without confirmation
  -h, --help             help for passgen
  -i, --include string   add specific characters after filtering
      --json             output as JSON
  -k, --length int       password length, 1-4096 (default 16)
  -l, --lowercase        include lowercase letters in base pool (default true)
  -n, --numbers          include numbers in base pool (default true)
      --out string       write output to file (mode 600), suppress stdout
  -p, --preset string    replace pool with preset characters (base64/b64, base64url/b64url, base58/b58, hex, alnum)
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

1. Add enabled base classes (`--uppercase`, `--lowercase`, `--numbers`, `--symbols`).
2. Apply URL-safe filtering if `--urlsafe` is enabled.
3. Replace with `--preset` if set.
4. Remove characters from `--exclude`.
5. Add characters from `--include`.

`--include` and `--exclude` are validated first. If they overlap, the command exits with an error.

When `--preset` is used, default base classes are not added. Explicit base-class flags, `--alpha`, or `--urlsafe` with `--preset` print a warning to stderr because `--preset` replaces the earlier pool. Preset characters are preserved unless removed by `--exclude`.

## Validation behavior

- Positional arguments are not accepted.
- `--length` must be between `1` and `4096`.
- `--count` must be between `1` and `1000`.
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
