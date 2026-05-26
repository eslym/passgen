# passgen

`passgen` is a small Go CLI for generating cryptographically secure passwords.

> [!WARNING]
> This project is vibe-coded, but it is safe to use as a small utility when you need a quick, safe password in the terminal.

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

Inspect the effective pool used for generation:

```bash
passgen --show-pool
```

## Flags

```text
  -a, --alpha            enable uppercase and lowercase
  -c, --count int        number of passwords to generate (default 1)
  -x, --exclude string   exclude specific characters
  -h, --help             help for passgen
  -i, --include string   include specific characters
      --json             output as JSON
  -k, --length int       password length (default 16)
  -l, --lowercase        enable lowercase letters (default true)
  -A, --no-alpha         disable uppercase and lowercase
  -L, --no-lowercase     disable lowercase letters
  -N, --no-numbers       disable numbers
      --out string       write output to file with mode 600
  -S, --no-symbols       disable symbols
  -U, --no-uppercase     disable uppercase letters
  -Z, --no-urlsafe       disable URL-safe filtering
  -n, --numbers          enable numbers (default true)
  -s, --symbols          enable symbols (default true)
      --show-pool        print effective character pool
  -u, --uppercase        enable uppercase letters (default true)
  -z, --urlsafe          only keep URL-safe chars in base pool
```

## Precedence and pool order

Character pool construction is applied in this order:

1. Start from enabled base classes (`--uppercase`, `--lowercase`, `--numbers`, `--symbols`).
2. Apply URL-safe filtering if `--urlsafe` is enabled.
3. Remove characters from `--exclude`.
4. Add characters from `--include`.

`--include` and `--exclude` are validated first. If they overlap, the command exits with an error.

## Validation behavior

- `--length` must be greater than `0`.
- When `--out` is set, output is written only to the file (mode `600`) and a status line is printed to stderr.
- If the same character appears in both `--include` and `--exclude`, the command exits with an error.
- If your rules produce an empty character pool, the command exits with an error.

## Development checks

```bash
go test ./...
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```
