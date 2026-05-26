package main

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

const (
	uppercaseChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	lowercaseChars = "abcdefghijklmnopqrstuvwxyz"
	numberChars    = "0123456789"
	symbolChars    = "!\"#$%&'()*+,./:;<=>?@[\\]^`{|}~-"
	urlSafeChars   = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_.~"
)

type options struct {
	uppercase bool
	lowercase bool
	numbers   bool
	symbols   bool
	urlsafe   bool
	include   string
	exclude   string
	length    int
	count     int
	jsonOut   bool
	showPool  bool
	out       string
}

func main() {
	opts := defaultOptions()
	cmd := newRootCmd(&opts)

	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func defaultOptions() options {
	return options{
		uppercase: true,
		lowercase: true,
		numbers:   true,
		symbols:   true,
		urlsafe:   false,
		length:    16,
		count:     1,
	}
}

func newRootCmd(opts *options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "passgen",
		Short: "Generate cryptographically secure passwords",
		Long: "passgen generates cryptographically secure passwords using crypto/rand. " +
			"You can tune the character pool with class flags and include/exclude rules.",
		Example: strings.Join([]string{
			"  passgen",
			"  passgen --length 32",
			"  passgen --count 5 --json",
			"  passgen --urlsafe --no-symbols",
			"  passgen --include \"@#\" --exclude \"O0Il\"",
			"  passgen --out ./secret.txt",
		}, "\n"),
		RunE: func(cmd *cobra.Command, _ []string) error {
			if opts.length <= 0 {
				return errors.New("--length must be greater than 0")
			}
			if opts.count <= 0 {
				return errors.New("--count must be greater than 0")
			}

			pool, err := buildPool(*opts)
			if err != nil {
				return err
			}

			passwords := make([]string, 0, opts.count)
			for i := 0; i < opts.count; i++ {
				pass, err := generatePassword(pool, opts.length)
				if err != nil {
					return err
				}
				passwords = append(passwords, pass)
			}

			output, err := renderOutput(*opts, passwords, pool)
			if err != nil {
				return err
			}

			if opts.out != "" {
				if err := writeOutputFile(opts.out, output, cmd.ErrOrStderr()); err != nil {
					return err
				}
				return nil
			}

			fmt.Fprint(cmd.OutOrStdout(), output)
			return nil
		},
	}

	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	cmd.Flags().BoolVarP(&opts.uppercase, "uppercase", "u", true, "include uppercase letters in base pool")
	cmd.Flags().BoolVarP(&opts.lowercase, "lowercase", "l", true, "include lowercase letters in base pool")
	cmd.Flags().BoolVarP(&opts.numbers, "numbers", "n", true, "include numbers in base pool")
	cmd.Flags().BoolVarP(&opts.symbols, "symbols", "s", true, "include symbols in base pool")
	cmd.Flags().BoolVarP(&opts.urlsafe, "urlsafe", "z", false, "filter base pool to URL-safe characters")

	noUpper := cmd.Flags().BoolP("no-uppercase", "U", false, "remove uppercase letters from base pool")
	noLower := cmd.Flags().BoolP("no-lowercase", "L", false, "remove lowercase letters from base pool")
	alpha := cmd.Flags().BoolP("alpha", "a", false, "enable both uppercase and lowercase")
	noAlpha := cmd.Flags().BoolP("no-alpha", "A", false, "disable both uppercase and lowercase")
	noNumbers := cmd.Flags().BoolP("no-numbers", "N", false, "remove numbers from base pool")
	noSymbols := cmd.Flags().BoolP("no-symbols", "S", false, "remove symbols from base pool")
	noURLSafe := cmd.Flags().BoolP("no-urlsafe", "Z", false, "disable URL-safe filtering")

	cmd.Flags().StringVarP(&opts.include, "include", "i", "", "add specific characters after filtering")
	cmd.Flags().StringVarP(&opts.exclude, "exclude", "x", "", "remove specific characters before include")
	cmd.Flags().IntVarP(&opts.length, "length", "k", 16, "password length")
	cmd.Flags().IntVarP(&opts.count, "count", "c", 1, "number of passwords to generate")
	cmd.Flags().BoolVar(&opts.jsonOut, "json", false, "output as JSON")
	cmd.Flags().BoolVar(&opts.showPool, "show-pool", false, "print effective character pool")
	cmd.Flags().StringVar(&opts.out, "out", "", "write output to file (mode 600), suppress stdout")

	cmd.PreRun = func(cmd *cobra.Command, _ []string) {
		if *alpha {
			opts.uppercase = true
			opts.lowercase = true
		}
		if *noAlpha {
			opts.uppercase = false
			opts.lowercase = false
		}
		if *noUpper {
			opts.uppercase = false
		}
		if *noLower {
			opts.lowercase = false
		}
		if *noNumbers {
			opts.numbers = false
		}
		if *noSymbols {
			opts.symbols = false
		}
		if *noURLSafe {
			opts.urlsafe = false
		}
	}

	return cmd
}

func renderOutput(opts options, passwords []string, pool string) (string, error) {
	if opts.jsonOut {
		var payload any
		if opts.count == 1 {
			if opts.showPool {
				payload = struct {
					Password string `json:"password"`
					Pool     string `json:"pool"`
				}{Password: passwords[0], Pool: pool}
			} else {
				payload = struct {
					Password string `json:"password"`
				}{Password: passwords[0]}
			}
		} else {
			if opts.showPool {
				payload = struct {
					Passwords []string `json:"passwords"`
					Pool      string   `json:"pool"`
				}{Passwords: passwords, Pool: pool}
			} else {
				payload = struct {
					Passwords []string `json:"passwords"`
				}{Passwords: passwords}
			}
		}

		encoded, err := json.Marshal(payload)
		if err != nil {
			return "", fmt.Errorf("json output failed: %w", err)
		}
		return string(encoded) + "\n", nil
	}

	var builder strings.Builder
	if opts.showPool {
		builder.WriteString(pool)
		builder.WriteString("\n")
	}
	for _, pass := range passwords {
		builder.WriteString(pass)
		builder.WriteString("\n")
	}
	return builder.String(), nil
}

func writeOutputFile(path, output string, errWriter io.Writer) error {
	if err := os.WriteFile(path, []byte(output), 0o600); err != nil {
		return fmt.Errorf("failed writing output file %q: %w", path, err)
	}
	fprintfErr(errWriter, "Wrote output to %s with mode 600\n", path)
	return nil
}

func fprintfErr(w io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(w, format, args...)
}

func buildPool(opts options) (string, error) {
	overlap := overlapCharacters(opts.include, opts.exclude)
	if overlap != "" {
		return "", fmt.Errorf("include/exclude overlap: %q", overlap)
	}

	var builder strings.Builder
	if opts.uppercase {
		builder.WriteString(uppercaseChars)
	}
	if opts.lowercase {
		builder.WriteString(lowercaseChars)
	}
	if opts.numbers {
		builder.WriteString(numberChars)
	}
	if opts.symbols {
		builder.WriteString(symbolChars)
	}

	pool := uniqueChars(builder.String())
	if opts.urlsafe {
		pool = filterAllowed(pool, urlSafeChars)
	}

	pool = removeChars(pool, opts.exclude)
	pool = addChars(pool, opts.include)

	if pool == "" {
		return "", errors.New("character pool is empty after applying rules")
	}

	return pool, nil
}

func overlapCharacters(a, b string) string {
	bSet := runeSet(b)
	seen := make(map[rune]struct{})
	var overlap []rune
	for _, r := range a {
		if _, ok := bSet[r]; ok {
			if _, added := seen[r]; !added {
				overlap = append(overlap, r)
				seen[r] = struct{}{}
			}
		}
	}
	return string(overlap)
}

func runeSet(s string) map[rune]struct{} {
	set := make(map[rune]struct{})
	for _, r := range s {
		set[r] = struct{}{}
	}
	return set
}

func uniqueChars(s string) string {
	set := make(map[rune]struct{})
	var out []rune
	for _, r := range s {
		if _, ok := set[r]; ok {
			continue
		}
		set[r] = struct{}{}
		out = append(out, r)
	}
	return string(out)
}

func filterAllowed(pool, allowed string) string {
	allowedSet := runeSet(allowed)
	var out []rune
	for _, r := range pool {
		if _, ok := allowedSet[r]; ok {
			out = append(out, r)
		}
	}
	return string(out)
}

func removeChars(pool, toRemove string) string {
	removeSet := runeSet(toRemove)
	var out []rune
	for _, r := range pool {
		if _, ok := removeSet[r]; ok {
			continue
		}
		out = append(out, r)
	}
	return string(out)
}

func addChars(pool, toAdd string) string {
	set := runeSet(pool)
	out := []rune(pool)
	for _, r := range toAdd {
		if _, ok := set[r]; ok {
			continue
		}
		set[r] = struct{}{}
		out = append(out, r)
	}
	return string(out)
}

func generatePassword(pool string, length int) (string, error) {
	runes := []rune(pool)
	max := big.NewInt(int64(len(runes)))
	var out strings.Builder
	out.Grow(length)

	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", fmt.Errorf("secure random generation failed: %w", err)
		}
		out.WriteRune(runes[n.Int64()])
	}

	return out.String(), nil
}
