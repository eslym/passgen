# passgen

`passgen` is a small Go CLI for generating cryptographically secure passwords.

## Features

- Uses `crypto/rand` for secure random generation.
- Configurable character sets (uppercase, lowercase, numbers, symbols).
- Optional URL-safe pool filtering.
- Supports explicit character include/exclude rules.

## Requirements

- Go 1.26.3+

## Build and run

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

Disable symbols:

```bash
passgen --no-symbols
```

Use only URL-safe characters:

```bash
passgen --urlsafe --no-symbols
```

Force a few characters into the pool:

```bash
passgen --include "@#"
```

Remove ambiguous characters:

```bash
passgen --exclude "O0Il"
```

## Flags

```text
  -a, --alpha            enable uppercase and lowercase
  -x, --exclude string   exclude specific characters
  -h, --help             help for passgen
  -i, --include string   include specific characters
  -k, --length int       password length (default 16)
  -l, --lowercase        enable lowercase letters (default true)
  -A, --no-alpha         disable uppercase and lowercase
  -L, --no-lowercase     disable lowercase letters
  -N, --no-numbers       disable numbers
  -S, --no-symbols       disable symbols
  -U, --no-uppercase     disable uppercase letters
  -Z, --no-urlsafe       disable URL-safe filtering
  -n, --numbers          enable numbers (default true)
  -s, --symbols          enable symbols (default true)
  -u, --uppercase        enable uppercase letters (default true)
  -z, --urlsafe          only keep URL-safe chars in base pool
```

## Validation behavior

- `--length` must be greater than `0`.
- If the same character appears in both `--include` and `--exclude`, the command exits with an error.
- If your rules produce an empty character pool, the command exits with an error.

## Development checks

```bash
go test ./...
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```
