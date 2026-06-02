package main

import (
	"bufio"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const (
	uppercaseChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	lowercaseChars = "abcdefghijklmnopqrstuvwxyz"
	numberChars    = "0123456789"
	symbolChars    = "!\"#$%&'()*+,./:;<=>?@[\\]^`{|}~-"
	urlSafeChars   = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_.~"
	base64Chars    = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	base64URLChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	base58Chars    = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
	hexChars       = "0123456789abcdef"
	alnumChars     = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
)

var presetPools = map[string]string{
	"base64":    base64Chars,
	"b64":       base64Chars,
	"base64url": base64URLChars,
	"b64url":    base64URLChars,
	"base58":    base58Chars,
	"b58":       base58Chars,
	"hex":       hexChars,
	"alnum":     alnumChars,
}

type options struct {
	uppercase bool
	lowercase bool
	numbers   bool
	symbols   bool
	urlsafe   bool
	preset    string
	include   string
	exclude   string
	length    int
	count     int
	jsonOut   bool
	showPool  bool
	out       string
	force     bool
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
			"  passgen --preset b58 --symbols=false --uppercase=false --lowercase=false --numbers=false",
			"  passgen --urlsafe --symbols=false",
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
			warnPresetPoolModifiers(*opts, cmd.ErrOrStderr())

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
				if err := writeOutputFile(opts.out, output, cmd.ErrOrStderr(), cmd.InOrStdin(), opts.force); err != nil {
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
	cmd.Flags().StringVarP(&opts.preset, "preset", "p", "", "seed pool with preset characters (base64/b64, base64url/b64url, base58/b58, hex, alnum)")

	alpha := cmd.Flags().BoolP("alpha", "a", false, "enable both uppercase and lowercase")

	cmd.Flags().StringVarP(&opts.include, "include", "i", "", "add specific characters after filtering")
	cmd.Flags().StringVarP(&opts.exclude, "exclude", "x", "", "remove specific characters before include")
	cmd.Flags().IntVarP(&opts.length, "length", "k", 16, "password length")
	cmd.Flags().IntVarP(&opts.count, "count", "c", 1, "number of passwords to generate")
	cmd.Flags().BoolVar(&opts.jsonOut, "json", false, "output as JSON")
	cmd.Flags().BoolVar(&opts.showPool, "show-pool", false, "print effective character pool")
	cmd.Flags().StringVar(&opts.out, "out", "", "write output to file (mode 600), suppress stdout")
	cmd.Flags().BoolVarP(&opts.force, "force", "f", false, "overwrite existing output file without confirmation")

	cmd.PreRun = func(cmd *cobra.Command, _ []string) {
		if *alpha {
			var overridden []string
			if cmd.Flags().Changed("uppercase") && !opts.uppercase {
				overridden = append(overridden, "--uppercase=false")
			}
			if cmd.Flags().Changed("lowercase") && !opts.lowercase {
				overridden = append(overridden, "--lowercase=false")
			}
			if len(overridden) > 0 {
				fprintfErr(cmd.ErrOrStderr(), "Warning: --alpha overrides explicit case flags: %s\n", strings.Join(overridden, ", "))
			}
			opts.uppercase = true
			opts.lowercase = true
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

func writeOutputFile(path, output string, errWriter io.Writer, in io.Reader, force bool) error {
	info, err := os.Lstat(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed stating output file %q: %w", path, err)
	}

	if err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("output path %q is a symlink", path)
		}
		if info.IsDir() {
			return fmt.Errorf("output path %q is a directory", path)
		}
		if !force {
			confirmed, err := confirmOverwrite(path, errWriter, in)
			if err != nil {
				return err
			}
			if !confirmed {
				return fmt.Errorf("refusing to overwrite existing file %q", path)
			}
		}
		if info.Mode().Perm() != 0o600 {
			fprintfErr(errWriter, "Warning: existing output file %s has mode %o, expected 600; updating mode to 600\n", path, info.Mode().Perm())
		}
	}

	fileWritten := false
	if err == nil {
		fileWritten, err = atomicWriteFile(path, []byte(output), 0o600)
	} else {
		fileWritten, err = atomicCreateFile(path, []byte(output), 0o600)
	}
	if err != nil {
		if fileWritten {
			return fmt.Errorf("wrote output file %q, but failed finalizing output write: %w", path, err)
		}
		return fmt.Errorf("failed writing output file %q: %w", path, err)
	}
	fprintfErr(errWriter, "Wrote output to %s with mode 600\n", path)
	return nil
}

func atomicWriteFile(path string, data []byte, mode os.FileMode) (bool, error) {
	return atomicWriteFileWithSync(path, data, mode, syncDir)
}

func atomicCreateFile(path string, data []byte, mode os.FileMode) (bool, error) {
	return atomicCreateFileWithSync(path, data, mode, syncDir)
}

func atomicWriteFileWithSync(path string, data []byte, mode os.FileMode, syncDirFunc func(string) error) (bool, error) {
	dir := filepath.Dir(path)
	base := filepath.Base(path)

	tmp, err := os.CreateTemp(dir, "."+base+"-*")
	if err != nil {
		return false, err
	}
	tmpPath := tmp.Name()
	removeTmp := true
	defer func() {
		if removeTmp {
			_ = os.Remove(tmpPath)
		}
	}()

	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return false, err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return false, err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return false, err
	}
	if err := tmp.Close(); err != nil {
		return false, err
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return false, err
	}
	removeTmp = false

	if err := syncDirFunc(dir); err != nil {
		return true, err
	}
	return true, nil
}

func atomicCreateFileWithSync(path string, data []byte, mode os.FileMode, syncDirFunc func(string) error) (bool, error) {
	dir := filepath.Dir(path)
	base := filepath.Base(path)

	tmp, err := os.CreateTemp(dir, "."+base+"-*")
	if err != nil {
		return false, err
	}
	tmpPath := tmp.Name()
	removeTmp := true
	defer func() {
		if removeTmp {
			_ = os.Remove(tmpPath)
		}
	}()

	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return false, err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return false, err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return false, err
	}
	if err := tmp.Close(); err != nil {
		return false, err
	}

	if err := os.Link(tmpPath, path); err != nil {
		return false, err
	}
	if err := os.Remove(tmpPath); err != nil {
		return true, err
	}
	removeTmp = false

	if err := syncDirFunc(dir); err != nil {
		return true, err
	}
	return true, nil
}

func confirmOverwrite(path string, errWriter io.Writer, in io.Reader) (bool, error) {
	fprintfErr(errWriter, "Output file %s already exists. Overwrite? [y/N]: ", path)
	line, err := bufio.NewReader(in).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, fmt.Errorf("failed reading overwrite confirmation: %w", err)
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes", nil
}

func fprintfErr(w io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(w, format, args...)
}

func warnPresetPoolModifiers(opts options, errWriter io.Writer) {
	modifiers := presetPoolModifiers(opts)
	if len(modifiers) == 0 {
		return
	}
	fprintfErr(errWriter, "Warning: --preset seeds the pool; additional pool flags also modify it: %s\n", strings.Join(modifiers, ", "))
}

func presetPoolModifiers(opts options) []string {
	if opts.preset == "" {
		return nil
	}

	var modifiers []string
	if opts.uppercase {
		modifiers = append(modifiers, "--uppercase")
	}
	if opts.lowercase {
		modifiers = append(modifiers, "--lowercase")
	}
	if opts.numbers {
		modifiers = append(modifiers, "--numbers")
	}
	if opts.symbols {
		modifiers = append(modifiers, "--symbols")
	}
	if opts.urlsafe {
		modifiers = append(modifiers, "--urlsafe")
	}
	if opts.exclude != "" {
		modifiers = append(modifiers, "--exclude")
	}
	if opts.include != "" {
		modifiers = append(modifiers, "--include")
	}
	return modifiers
}

func buildPool(opts options) (string, error) {
	overlap := overlapCharacters(opts.include, opts.exclude)
	if overlap != "" {
		return "", fmt.Errorf("include/exclude overlap: %q", overlap)
	}

	var builder strings.Builder
	if opts.preset != "" {
		presetPool, ok := presetPools[opts.preset]
		if !ok {
			return "", fmt.Errorf("unknown preset %q", opts.preset)
		}
		builder.WriteString(presetPool)
	}
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

// generatePassword requires a non-empty pool and a positive length. CLI callers
// enforce both invariants before generation via flag validation and buildPool,
// so further validation is unnecessary unless future caller changes break that contract.
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
