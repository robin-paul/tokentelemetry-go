package models

import (
	"testing"
)

func TestCanonicalProject(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "windows backslash path",
			input:    `C:\Users\dev\myproject`,
			expected: "C:/Users/dev/myproject",
		},
		{
			name:     "windows backslash trailing slash",
			input:    `C:\Users\dev\myproject\`,
			expected: "C:/Users/dev/myproject",
		},
		{
			name:     "windows forward slash trailing slash",
			input:    "C:/Users/dev/myproject/",
			expected: "C:/Users/dev/myproject",
		},
		{
			name:     "windows mixed separators",
			input:    `C:\Users/dev\myproject/`,
			expected: "C:/Users/dev/myproject",
		},
		{
			name:     "unc root path",
			input:    `\\server\share\repo`,
			expected: "//server/share/repo",
		},
		{
			name:     "posix path with trailing slash",
			input:    "/home/user/project/",
			expected: "/home/user/project",
		},
		{
			name:     "posix backslash in filename preserved",
			input:    `/data/weird\name`,
			expected: `/data/weird\name`,
		},
		{
			name:     "sentinel unknown",
			input:    "unknown",
			expected: "unknown",
		},
		{
			name:     "sentinel antigravity unassigned",
			input:    "Antigravity / unassigned",
			expected: "Antigravity / unassigned",
		},
		{
			name:     "relative path",
			input:    "relative/path",
			expected: "relative/path",
		},
		{
			name:     "root slash",
			input:    "/",
			expected: "/",
		},
		{
			name:     "double root slash",
			input:    "//",
			expected: "//",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := CanonicalProject(tc.input)
			if got != tc.expected {
				t.Errorf("CanonicalProject(%q) = %q, expected %q", tc.input, got, tc.expected)
			}
		})
	}
}
