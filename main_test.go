package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func executeCommand(args ...string) (string, string, error) {
	opts := defaultOptions()
	cmd := newRootCmd(&opts)
	cmd.SetArgs(args)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := cmd.Execute()
	return stdout.String(), stderr.String(), err
}

func TestBuildPool(t *testing.T) {
	t.Parallel()

	basePool := uppercaseChars + lowercaseChars + numberChars + symbolChars
	filteredBase := filterAllowed(uniqueChars(basePool), urlSafeChars)

	tests := []struct {
		name    string
		opts    options
		want    string
		wantErr string
	}{
		{
			name: "default classes build expected base pool",
			opts: options{uppercase: true, lowercase: true, numbers: true, symbols: true},
			want: basePool,
		},
		{
			name:    "include and exclude overlap is rejected",
			opts:    options{uppercase: true, include: "A", exclude: "A"},
			wantErr: "include/exclude overlap",
		},
		{
			name: "urlsafe filters base pool first",
			opts: options{uppercase: true, lowercase: true, numbers: true, symbols: true, urlsafe: true},
			want: filteredBase,
		},
		{
			name: "exclude removes from filtered pool",
			opts: options{uppercase: true, lowercase: true, numbers: true, symbols: true, urlsafe: true, exclude: "A-z0"},
			want: strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(filteredBase, "A", ""), "z", ""), "0", ""), "-", ""),
		},
		{
			name: "include has final priority and can add non-urlsafe chars",
			opts: options{uppercase: true, lowercase: true, numbers: true, symbols: true, urlsafe: true, include: "!@"},
			want: filteredBase + "!@",
		},
		{
			name: "include has priority over exclude when no overlap",
			opts: options{uppercase: true, lowercase: true, include: "1", exclude: "A"},
			want: strings.ReplaceAll(uppercaseChars+lowercaseChars, "A", "") + "1",
		},
		{
			name: "preset seeds pool before class flags",
			opts: options{preset: "hex", uppercase: true},
			want: hexChars + uppercaseChars,
		},
		{
			name: "preset only when classes are disabled",
			opts: options{preset: "base58"},
			want: base58Chars,
		},
		{
			name: "base64 alias matches canonical preset",
			opts: options{preset: "b64"},
			want: base64Chars,
		},
		{
			name: "base64url alias matches canonical preset",
			opts: options{preset: "b64url"},
			want: base64URLChars,
		},
		{
			name: "base58 alias matches canonical preset",
			opts: options{preset: "b58"},
			want: base58Chars,
		},
		{
			name: "urlsafe filters preset and class flags",
			opts: options{preset: "base64", urlsafe: true},
			want: filterAllowed(base64Chars, urlSafeChars),
		},
		{
			name: "include and exclude apply after preset",
			opts: options{preset: "hex", exclude: "0a", include: "Z"},
			want: strings.ReplaceAll(strings.ReplaceAll(hexChars, "0", ""), "a", "") + "Z",
		},
		{
			name:    "unknown preset is rejected",
			opts:    options{preset: "base32"},
			wantErr: "unknown preset",
		},
		{
			name:    "empty pool after rules is rejected",
			opts:    options{uppercase: true, exclude: uppercaseChars},
			wantErr: "character pool is empty",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := buildPool(tc.opts)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("expected error containing %q, got %q", tc.wantErr, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("buildPool returned error: %v", err)
			}

			if got != tc.want {
				t.Fatalf("unexpected pool\nwant: %q\ngot:  %q", tc.want, got)
			}
		})
	}
}

func TestWriteOutputFile(t *testing.T) {
	t.Parallel()

	t.Run("writes file with mode 600 and stderr message", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		outPath := filepath.Join(tmpDir, "secret.txt")
		content := "supersecret\n"
		var errBuf bytes.Buffer

		err := writeOutputFile(outPath, content, &errBuf, strings.NewReader(""), false)
		if err != nil {
			t.Fatalf("writeOutputFile returned error: %v", err)
		}

		gotBytes, err := os.ReadFile(outPath)
		if err != nil {
			t.Fatalf("failed reading output file: %v", err)
		}
		if string(gotBytes) != content {
			t.Fatalf("unexpected file content\nwant: %q\ngot:  %q", content, string(gotBytes))
		}

		info, err := os.Stat(outPath)
		if err != nil {
			t.Fatalf("failed stating output file: %v", err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("unexpected file mode\nwant: %o\ngot:  %o", 0o600, info.Mode().Perm())
		}

		msg := errBuf.String()
		if !strings.Contains(msg, outPath) {
			t.Fatalf("stderr message should include output path, got: %q", msg)
		}
		if !strings.Contains(msg, "mode 600") {
			t.Fatalf("stderr message should mention mode 600, got: %q", msg)
		}
	})

	t.Run("refuses existing file without confirmation", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		outPath := filepath.Join(tmpDir, "secret.txt")
		original := "keep me\n"
		if err := os.WriteFile(outPath, []byte(original), 0o600); err != nil {
			t.Fatalf("failed preparing output file: %v", err)
		}

		var errBuf bytes.Buffer
		err := writeOutputFile(outPath, "replace me\n", &errBuf, strings.NewReader("n\n"), false)
		if err == nil {
			t.Fatalf("expected overwrite refusal, got nil")
		}
		if !strings.Contains(err.Error(), "refusing to overwrite") {
			t.Fatalf("expected overwrite refusal, got: %v", err)
		}
		if !strings.Contains(errBuf.String(), "Overwrite?") {
			t.Fatalf("expected confirmation prompt, got: %q", errBuf.String())
		}

		got, err := os.ReadFile(outPath)
		if err != nil {
			t.Fatalf("failed reading output file: %v", err)
		}
		if string(got) != original {
			t.Fatalf("existing file should not be changed\nwant: %q\ngot:  %q", original, string(got))
		}
	})

	t.Run("overwrites existing file after confirmation", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		outPath := filepath.Join(tmpDir, "secret.txt")
		content := "new secret\n"
		if err := os.WriteFile(outPath, []byte("old secret\n"), 0o600); err != nil {
			t.Fatalf("failed preparing output file: %v", err)
		}

		var errBuf bytes.Buffer
		err := writeOutputFile(outPath, content, &errBuf, strings.NewReader("yes\n"), false)
		if err != nil {
			t.Fatalf("writeOutputFile returned error: %v", err)
		}
		if !strings.Contains(errBuf.String(), "Overwrite?") {
			t.Fatalf("expected confirmation prompt, got: %q", errBuf.String())
		}

		got, err := os.ReadFile(outPath)
		if err != nil {
			t.Fatalf("failed reading output file: %v", err)
		}
		if string(got) != content {
			t.Fatalf("unexpected file content\nwant: %q\ngot:  %q", content, string(got))
		}
	})

	t.Run("force overwrites existing file without confirmation", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		outPath := filepath.Join(tmpDir, "secret.txt")
		content := "forced secret\n"
		if err := os.WriteFile(outPath, []byte("old secret\n"), 0o600); err != nil {
			t.Fatalf("failed preparing output file: %v", err)
		}

		var errBuf bytes.Buffer
		err := writeOutputFile(outPath, content, &errBuf, strings.NewReader(""), true)
		if err != nil {
			t.Fatalf("writeOutputFile returned error: %v", err)
		}
		if strings.Contains(errBuf.String(), "Overwrite?") {
			t.Fatalf("did not expect confirmation prompt with force, got: %q", errBuf.String())
		}

		got, err := os.ReadFile(outPath)
		if err != nil {
			t.Fatalf("failed reading output file: %v", err)
		}
		if string(got) != content {
			t.Fatalf("unexpected file content\nwant: %q\ngot:  %q", content, string(got))
		}

		info, err := os.Stat(outPath)
		if err != nil {
			t.Fatalf("failed stating output file: %v", err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("unexpected file mode\nwant: %o\ngot:  %o", 0o600, info.Mode().Perm())
		}
	})

	t.Run("rejects symlink output path", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		targetPath := filepath.Join(tmpDir, "target.txt")
		linkPath := filepath.Join(tmpDir, "secret.txt")
		original := "keep me\n"
		if err := os.WriteFile(targetPath, []byte(original), 0o600); err != nil {
			t.Fatalf("failed preparing symlink target: %v", err)
		}
		if err := os.Symlink(targetPath, linkPath); err != nil {
			t.Fatalf("failed preparing symlink: %v", err)
		}

		var errBuf bytes.Buffer
		err := writeOutputFile(linkPath, "replace me\n", &errBuf, strings.NewReader(""), true)
		if err == nil {
			t.Fatalf("expected symlink rejection, got nil")
		}
		if !strings.Contains(err.Error(), "is a symlink") {
			t.Fatalf("expected symlink rejection, got: %v", err)
		}

		got, err := os.ReadFile(targetPath)
		if err != nil {
			t.Fatalf("failed reading symlink target: %v", err)
		}
		if string(got) != original {
			t.Fatalf("symlink target should not be changed\nwant: %q\ngot:  %q", original, string(got))
		}
	})

	t.Run("warns and fixes existing file mode", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		outPath := filepath.Join(tmpDir, "secret.txt")
		content := "secret\n"
		if err := os.WriteFile(outPath, []byte("old secret\n"), 0o644); err != nil {
			t.Fatalf("failed preparing output file: %v", err)
		}

		var errBuf bytes.Buffer
		err := writeOutputFile(outPath, content, &errBuf, strings.NewReader("y\n"), false)
		if err != nil {
			t.Fatalf("writeOutputFile returned error: %v", err)
		}

		msg := errBuf.String()
		if !strings.Contains(msg, "Warning: existing output file") {
			t.Fatalf("expected mode warning, got: %q", msg)
		}
		if !strings.Contains(msg, "mode 644") {
			t.Fatalf("expected warning to mention old mode, got: %q", msg)
		}

		info, err := os.Stat(outPath)
		if err != nil {
			t.Fatalf("failed stating output file: %v", err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("unexpected file mode\nwant: %o\ngot:  %o", 0o600, info.Mode().Perm())
		}
	})
}

func TestAtomicWriteFileReportsPostRenameSyncFailure(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "secret.txt")
	syncErr := errors.New("sync failed")

	fileReplaced, err := atomicWriteFileWithSync(outPath, []byte("secret\n"), 0o600, func(dir string) error {
		if dir != tmpDir {
			t.Fatalf("unexpected sync directory\nwant: %q\ngot:  %q", tmpDir, dir)
		}
		return syncErr
	})
	if !fileReplaced {
		t.Fatalf("expected fileReplaced to be true after rename")
	}
	if !errors.Is(err, syncErr) {
		t.Fatalf("expected sync error, got: %v", err)
	}

	got, readErr := os.ReadFile(outPath)
	if readErr != nil {
		t.Fatalf("expected renamed file to exist: %v", readErr)
	}
	if string(got) != "secret\n" {
		t.Fatalf("unexpected file content\nwant: %q\ngot:  %q", "secret\n", string(got))
	}
}

func TestOutFlagSuppressesStdout(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "secret.txt")

	opts := defaultOptions()
	cmd := newRootCmd(&opts)
	cmd.SetArgs([]string{"--length", "8", "--count", "1", "--out", outPath})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("command execution failed: %v", err)
	}

	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout when --out is used, got: %q", stdout.String())
	}

	stderrMsg := stderr.String()
	if !strings.Contains(stderrMsg, outPath) {
		t.Fatalf("stderr should include output path, got: %q", stderrMsg)
	}

	written, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("expected output file to exist: %v", err)
	}
	if len(strings.TrimSpace(string(written))) == 0 {
		t.Fatalf("expected output file to contain generated password")
	}
}

func TestOutFlagExistingFileConfirmation(t *testing.T) {
	t.Parallel()

	t.Run("refuses existing file when confirmation is declined", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		outPath := filepath.Join(tmpDir, "secret.txt")
		original := "existing\n"
		if err := os.WriteFile(outPath, []byte(original), 0o600); err != nil {
			t.Fatalf("failed preparing output file: %v", err)
		}

		opts := defaultOptions()
		cmd := newRootCmd(&opts)
		cmd.SetArgs([]string{"--length", "8", "--out", outPath})
		cmd.SetIn(strings.NewReader("n\n"))

		var stdout bytes.Buffer
		var stderr bytes.Buffer
		cmd.SetOut(&stdout)
		cmd.SetErr(&stderr)

		err := cmd.Execute()
		if err == nil {
			t.Fatalf("expected command error, got nil")
		}
		if !strings.Contains(stderr.String(), "Overwrite?") {
			t.Fatalf("expected confirmation prompt, got: %q", stderr.String())
		}
		if stdout.Len() != 0 {
			t.Fatalf("expected no stdout on refusal, got: %q", stdout.String())
		}

		got, err := os.ReadFile(outPath)
		if err != nil {
			t.Fatalf("failed reading output file: %v", err)
		}
		if string(got) != original {
			t.Fatalf("existing file should not be changed\nwant: %q\ngot:  %q", original, string(got))
		}
	})

	t.Run("force overwrites existing file without prompt", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		outPath := filepath.Join(tmpDir, "secret.txt")
		if err := os.WriteFile(outPath, []byte("existing\n"), 0o600); err != nil {
			t.Fatalf("failed preparing output file: %v", err)
		}

		opts := defaultOptions()
		cmd := newRootCmd(&opts)
		cmd.SetArgs([]string{"--length", "8", "--out", outPath, "--force"})
		cmd.SetIn(strings.NewReader(""))

		var stdout bytes.Buffer
		var stderr bytes.Buffer
		cmd.SetOut(&stdout)
		cmd.SetErr(&stderr)

		if err := cmd.Execute(); err != nil {
			t.Fatalf("command execution failed: %v", err)
		}
		if strings.Contains(stderr.String(), "Overwrite?") {
			t.Fatalf("did not expect confirmation prompt, got: %q", stderr.String())
		}
		if stdout.Len() != 0 {
			t.Fatalf("expected no stdout with --out, got: %q", stdout.String())
		}

		written, err := os.ReadFile(outPath)
		if err != nil {
			t.Fatalf("expected output file to exist: %v", err)
		}
		if len(strings.TrimSpace(string(written))) != 8 {
			t.Fatalf("expected 8-character generated password, got: %q", string(written))
		}
	})
}

func TestPresetModifierWarning(t *testing.T) {
	t.Parallel()

	t.Run("warns when preset is combined with effective pool modifiers", func(t *testing.T) {
		t.Parallel()

		opts := defaultOptions()
		cmd := newRootCmd(&opts)
		cmd.SetArgs([]string{"--preset", "b58", "--length", "1"})

		var stdout bytes.Buffer
		var stderr bytes.Buffer
		cmd.SetOut(&stdout)
		cmd.SetErr(&stderr)

		if err := cmd.Execute(); err != nil {
			t.Fatalf("command execution failed: %v", err)
		}

		stderrMsg := stderr.String()
		if !strings.Contains(stderrMsg, "Warning: --preset seeds the pool") {
			t.Fatalf("expected preset warning, got: %q", stderrMsg)
		}
		for _, flag := range []string{"--uppercase", "--lowercase", "--numbers", "--symbols"} {
			if !strings.Contains(stderrMsg, flag) {
				t.Fatalf("expected warning to mention %s, got: %q", flag, stderrMsg)
			}
		}
	})

	t.Run("does not warn when preset is the only pool source", func(t *testing.T) {
		t.Parallel()

		opts := defaultOptions()
		cmd := newRootCmd(&opts)
		cmd.SetArgs([]string{"--preset", "b58", "--uppercase=false", "--lowercase=false", "--numbers=false", "--symbols=false", "--length", "1"})

		var stdout bytes.Buffer
		var stderr bytes.Buffer
		cmd.SetOut(&stdout)
		cmd.SetErr(&stderr)

		if err := cmd.Execute(); err != nil {
			t.Fatalf("command execution failed: %v", err)
		}

		if stderr.Len() != 0 {
			t.Fatalf("expected no stderr warning, got: %q", stderr.String())
		}
	})
}

func TestAlphaOverrideWarning(t *testing.T) {
	t.Parallel()

	stdout, stderr, err := executeCommand("--alpha", "--uppercase=false", "--lowercase=false", "--numbers=false", "--symbols=false", "--length", "4")
	if err != nil {
		t.Fatalf("command execution failed: %v", err)
	}
	if !strings.Contains(stderr, "Warning: --alpha overrides explicit case flags") {
		t.Fatalf("expected alpha override warning, got: %q", stderr)
	}
	for _, flag := range []string{"--uppercase=false", "--lowercase=false"} {
		if !strings.Contains(stderr, flag) {
			t.Fatalf("expected warning to mention %s, got: %q", flag, stderr)
		}
	}

	password := strings.TrimSpace(stdout)
	if len(password) != 4 {
		t.Fatalf("expected 4-character password, got %q", password)
	}
	for _, r := range password {
		if !strings.ContainsRune(uppercaseChars+lowercaseChars, r) {
			t.Fatalf("password contains rune %q outside alpha pool: %q", r, password)
		}
	}
}

func TestPresetCLIBehavior(t *testing.T) {
	t.Parallel()

	classOffArgs := []string{"--uppercase=false", "--lowercase=false", "--numbers=false", "--symbols=false"}

	t.Run("show-pool prints preset-only pool before password", func(t *testing.T) {
		t.Parallel()

		args := append([]string{"--preset", "hex", "--show-pool", "--length", "4"}, classOffArgs...)
		stdout, stderr, err := executeCommand(args...)
		if err != nil {
			t.Fatalf("command execution failed: %v", err)
		}
		if stderr != "" {
			t.Fatalf("expected no stderr, got: %q", stderr)
		}

		lines := strings.Split(strings.TrimSuffix(stdout, "\n"), "\n")
		if len(lines) != 2 {
			t.Fatalf("expected pool and password lines, got %d lines in %q", len(lines), stdout)
		}
		if lines[0] != hexChars {
			t.Fatalf("unexpected pool line\nwant: %q\ngot:  %q", hexChars, lines[0])
		}
		if len(lines[1]) != 4 {
			t.Fatalf("expected 4-character password, got %q", lines[1])
		}
		for _, r := range lines[1] {
			if !strings.ContainsRune(hexChars, r) {
				t.Fatalf("password contains rune %q outside hex preset: %q", r, lines[1])
			}
		}
	})

	t.Run("alias produces canonical show-pool output", func(t *testing.T) {
		t.Parallel()

		args := append([]string{"--preset", "b64url", "--show-pool", "--length", "1"}, classOffArgs...)
		stdout, stderr, err := executeCommand(args...)
		if err != nil {
			t.Fatalf("command execution failed: %v", err)
		}
		if stderr != "" {
			t.Fatalf("expected no stderr, got: %q", stderr)
		}

		poolLine := strings.SplitN(stdout, "\n", 2)[0]
		if poolLine != base64URLChars {
			t.Fatalf("unexpected pool line\nwant: %q\ngot:  %q", base64URLChars, poolLine)
		}
	})

	t.Run("json includes effective preset pool", func(t *testing.T) {
		t.Parallel()

		args := append([]string{"--preset", "b58", "--json", "--show-pool", "--length", "6"}, classOffArgs...)
		stdout, stderr, err := executeCommand(args...)
		if err != nil {
			t.Fatalf("command execution failed: %v", err)
		}
		if stderr != "" {
			t.Fatalf("expected no stderr, got: %q", stderr)
		}

		var payload struct {
			Password string `json:"password"`
			Pool     string `json:"pool"`
		}
		if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
			t.Fatalf("failed decoding json output %q: %v", stdout, err)
		}
		if payload.Pool != base58Chars {
			t.Fatalf("unexpected json pool\nwant: %q\ngot:  %q", base58Chars, payload.Pool)
		}
		if len(payload.Password) != 6 {
			t.Fatalf("expected 6-character password, got %q", payload.Password)
		}
	})

	t.Run("urlsafe filters preset output", func(t *testing.T) {
		t.Parallel()

		args := append([]string{"--preset", "base64", "--urlsafe", "--show-pool", "--length", "1"}, classOffArgs...)
		stdout, stderr, err := executeCommand(args...)
		if err != nil {
			t.Fatalf("command execution failed: %v", err)
		}
		if !strings.Contains(stderr, "--urlsafe") {
			t.Fatalf("expected warning to mention --urlsafe, got: %q", stderr)
		}

		wantPool := filterAllowed(base64Chars, urlSafeChars)
		poolLine := strings.SplitN(stdout, "\n", 2)[0]
		if poolLine != wantPool {
			t.Fatalf("unexpected pool line\nwant: %q\ngot:  %q", wantPool, poolLine)
		}
		if strings.ContainsAny(poolLine, "+/") {
			t.Fatalf("urlsafe pool should not contain + or /, got: %q", poolLine)
		}
	})

	t.Run("unknown preset returns command error", func(t *testing.T) {
		t.Parallel()

		stdout, stderr, err := executeCommand("--preset", "base32", "--length", "1")
		if err == nil {
			t.Fatalf("expected command error, got nil")
		}
		if !strings.Contains(err.Error(), "unknown preset") {
			t.Fatalf("expected unknown preset error, got: %v", err)
		}
		if stdout != "" {
			t.Fatalf("expected no stdout on error, got: %q", stdout)
		}
		if stderr != "" {
			t.Fatalf("expected no command stderr on returned error, got: %q", stderr)
		}
	})
}
