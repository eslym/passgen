package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
			name: "include and exclude overlap is rejected",
			opts: options{uppercase: true, include: "A", exclude: "A"},
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
			name: "empty pool after rules is rejected",
			opts: options{uppercase: true, exclude: uppercaseChars},
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

		err := writeOutputFile(outPath, content, &errBuf)
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
