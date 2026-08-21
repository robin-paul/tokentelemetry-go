package parsers

import (
	"bufio"
	"bytes"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// ExtractProjectName extracts a clean project name from a file path or directory string.
func ExtractProjectName(filePath string) string {
	clean := filepath.Clean(filePath)
	dir := filepath.Dir(clean)

	// Check if any component in the path looks like a URL-encoded path (e.g. %2FUsers%2Fdev%2Fmyproject)
	parts := strings.Split(clean, string(os.PathSeparator))
	for _, p := range parts {
		if strings.Contains(p, "%2F") || strings.Contains(p, "%2f") || strings.HasPrefix(p, "-") {
			decoded, err := url.QueryUnescape(p)
			if err == nil && strings.Contains(decoded, "/") {
				subparts := strings.Split(strings.Trim(decoded, "/"), "/")
				if len(subparts) > 0 {
					return subparts[len(subparts)-1]
				}
			}
		}
	}

	// Try checking common project directories in path
	for i := len(parts) - 1; i >= 0; i-- {
		part := parts[i]
		if part == ".claude" || part == ".gemini" || part == ".codex" || part == ".cursor" ||
			part == ".copilot" || part == ".hermes" || part == ".grok" || part == ".pi" ||
			part == ".dsh" || part == ".muse" || part == ".prime" || part == ".qwen" ||
			part == ".cline" || part == ".smallcode" || part == ".vibe" || part == "sessions" ||
			part == "projects" || part == "brain" || part == "chats" || part == "transcripts" ||
			part == "agent-transcripts" || part == "logs" {
			if i > 0 {
				return parts[i-1]
			}
		}
	}

	base := filepath.Base(dir)
	if base == "." || base == "/" || base == "" {
		base = filepath.Base(clean)
	}
	return base
}

// ExtractSessionID extracts a session ID from a file path or filename.
func ExtractSessionID(filePath string) string {
	base := filepath.Base(filePath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	// If there's another extension like .jsonl.zstd
	if secondExt := filepath.Ext(name); secondExt != "" {
		name = strings.TrimSuffix(name, secondExt)
	}

	// If name starts with rollout-2026-06-10-123456-uuid
	if strings.HasPrefix(name, "rollout-") {
		parts := strings.Split(name, "-")
		if len(parts) >= 5 {
			return strings.Join(parts[4:], "-")
		}
	}

	// If name is transcript.jsonl, return parent directory name
	if name == "transcript" || name == "transcript_full" || name == "session" || name == "events" || name == "summary" || name == "signals" {
		parent := filepath.Base(filepath.Dir(filePath))
		if parent != "." && parent != "/" && parent != "" {
			return parent
		}
	}

	return name
}

// ReadLines calls the provided callback for each line read from r, tracking byte offsets.
func ReadLines(r io.Reader, startOffset int64, fn func(line []byte, offset int64) error) (int64, error) {
	br := bufio.NewReaderSize(r, 64*1024)
	currentOffset := startOffset

	for {
		line, isPrefix, err := br.ReadLine()
		if err != nil {
			if err == io.EOF {
				if len(line) > 0 {
					currentOffset += int64(len(line))
					_ = fn(line, currentOffset)
				}
				break
			}
			return currentOffset, err
		}

		var fullLine []byte
		if isPrefix {
			buf := bytes.NewBuffer(line)
			for isPrefix {
				var chunk []byte
				chunk, isPrefix, err = br.ReadLine()
				if err != nil && err != io.EOF {
					return currentOffset, err
				}
				buf.Write(chunk)
			}
			fullLine = buf.Bytes()
		} else {
			fullLine = line
		}

		currentOffset += int64(len(fullLine)) + 1 // +1 for newline character
		trimmed := bytes.TrimSpace(fullLine)
		if len(trimmed) == 0 {
			continue
		}

		if err := fn(trimmed, currentOffset); err != nil {
			return currentOffset, err
		}
	}

	return currentOffset, nil
}
