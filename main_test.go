package main

import (
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
