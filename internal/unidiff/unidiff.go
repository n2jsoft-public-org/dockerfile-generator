package unidiff

import (
	"fmt"
	"strings"
)

// Unified returns a unified diff of old vs new content for a single file path.
// It implements a simple line-based LCS diff sufficient for small Dockerfiles.
func Unified(oldText, newText, filePath string) string {
	oldLines := splitKeepNL(oldText)
	newLines := splitKeepNL(newText)

	// LCS dynamic programming table
	m, n := len(oldLines), len(newLines)
	lcs := make([][]int, m+1)
	for i := range lcs {
		lcs[i] = make([]int, n+1)
	}
	for i := m - 1; i >= 0; i-- {
		for j := n - 1; j >= 0; j-- {
			if oldLines[i] == newLines[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else if lcs[i+1][j] >= lcs[i][j+1] {
				lcs[i][j] = lcs[i+1][j]
			} else {
				lcs[i][j] = lcs[i][j+1]
			}
		}
	}

	// Backtrack to produce diff ops
	type op struct {
		kind string
		line string
	}
	var ops []op
	i, j := 0, 0
	for i < m && j < n {
		if oldLines[i] == newLines[j] {
			ops = append(ops, op{"=", oldLines[i]})
			i++
			j++
		} else if lcs[i+1][j] >= lcs[i][j+1] { // deletion
			ops = append(ops, op{"-", oldLines[i]})
			i++
		} else { // insertion
			ops = append(ops, op{"+", newLines[j]})
			j++
		}
	}
	for i < m {
		ops = append(ops, op{"-", oldLines[i]})
		i++
	}
	for j < n {
		ops = append(ops, op{"+", newLines[j]})
		j++
	}

	// Coalesce into single hunk (file is small). Compute counts.
	oldCount, newCount := 0, 0
	for _, o := range ops {
		if o.kind != "+" {
			oldCount++
		}
		if o.kind != "-" {
			newCount++
		}
	}
	if oldCount == 0 {
		oldCount = 1
	}
	if newCount == 0 {
		newCount = 1
	}

	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "--- %s (old)\n", filePath)
	_, _ = fmt.Fprintf(&b, "+++ %s (new)\n", filePath)
	_, _ = fmt.Fprintf(&b, "@@ -1,%d +1,%d @@\n", oldCount, newCount)
	for _, o := range ops {
		switch o.kind {
		case "=":
			b.WriteString(" " + ensureNoExtraNewline(o.line))
		case "+":
			b.WriteString("+" + ensureNoExtraNewline(o.line))
		case "-":
			b.WriteString("-" + ensureNoExtraNewline(o.line))
		}
	}
	return b.String()
}

func splitKeepNL(s string) []string {
	if s == "" {
		return []string{}
	}
	lines := strings.SplitAfter(s, "\n")
	// If original text didn't end with newline, keep final segment
	if !strings.HasSuffix(s, "\n") {
		return lines
	}
	return lines
}

func ensureNoExtraNewline(line string) string {
	// line already contains trailing newline if originally present.
	return line
}
